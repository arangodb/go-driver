//go:build resiliency

//
// DISCLAIMER
//
// Copyright 2026 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

package tests

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

const (
	// Slow-query kill tests only need well more than coordinatorKillCursorAfterDocs rows.
	coordinatorKillSlowQueryDocCount          = 200
	coordinatorKillReadAfterDocs              = 1
	coordinatorKillCursorAfterDocs            = 30
	coordinatorKillOperationTimeout           = 90 * time.Second
	coordinatorKillCursorOpenTimeout          = 2 * time.Minute
	coordinatorSlowReadQuerySleepSecondsLocal = 0.05
	coordinatorSlowReadQuerySleepSecondsK8s   = 0.25
)

func coordinatorSlowReadQuerySleepSeconds() float64 {
	if isK8S() {
		return coordinatorSlowReadQuerySleepSecondsK8s
	}
	return coordinatorSlowReadQuerySleepSecondsLocal
}

// coordinatorSlowReadQuery returns a streaming AQL query with per-document SLEEP so the
// cursor stays open long enough for coordinator kills through ingress load balancing.
// SLEEP is available in ArangoDB 3.x native AQL (used by k8s resiliency tests on 3.12+).
func coordinatorSlowReadQuery(collectionName string) string {
	_ = collectionName
	return fmt.Sprintf(
		`FOR i IN 1..%d RETURN { i: i, burn: SLEEP(%g) }`,
		coordinatorKillSlowQueryDocCount,
		coordinatorSlowReadQuerySleepSeconds(),
	)
}

// TestResiliency_CoordinatorRestartWhileIdle validates that a connected client remains usable
// after all coordinators are restarted while no concurrent workload is running.
func TestResiliency_CoordinatorRestartWhileIdle(t *testing.T) {
	requireResiliencyK8sCoordinatorMode(t)
	runResiliencyWithHTTPProtocols(t, testCoordinatorRestartWhileIdle)
}

func testCoordinatorRestartWhileIdle(t *testing.T, connFactory resiliencyConnectionFactory) {
	client := prepareResiliencyClient(t, connFactory)

	restartAllCoordinators(t)

	waitForSuccessfulVersion(t, client, 2*time.Minute)
}

// TestResiliency_CoordinatorRestartDuringActiveWorkload validates that a client issuing
// continuous requests survives a coordinator restart. Temporary failures during restart are expected.
func TestResiliency_CoordinatorRestartDuringActiveWorkload(t *testing.T) {
	requireResiliencyK8sCoordinatorMode(t)
	runResiliencyWithHTTPProtocols(t, testCoordinatorRestartDuringActiveWorkload)
}

func testCoordinatorRestartDuringActiveWorkload(t *testing.T, connFactory resiliencyConnectionFactory) {
	client := prepareResiliencyClient(t, connFactory)

	stats := &versionWorkloadStats{}
	workloadCtx, stopWorkload := context.WithCancel(context.Background())
	workloadDone := make(chan struct{})

	go func() {
		defer close(workloadDone)
		runVersionWorkload(workloadCtx, client, stats)
	}()

	waitForWorkloadSuccesses(t, stats, true, 1, 2*time.Minute)

	stats.markRestartStarted()
	restartAllCoordinators(t)
	stats.markIngressReady()

	waitForWorkloadSuccesses(t, stats, false, 1, 3*time.Minute)

	stopWorkload()

	select {
	case <-workloadDone:
	case <-time.After(30 * time.Second):
		require.Fail(t, "workload goroutine did not stop after cancellation; possible hang")
	}

	assertWorkloadRecovered(t, stats)
}

// TestResiliency_CoordinatorKillDuringRead validates that killing the coordinator handling an
// active read fails the cursor cleanly and the client recovers afterward.
func TestResiliency_CoordinatorKillDuringRead(t *testing.T) {
	requireResiliencyK8sCoordinatorMode(t)
	runResiliencyWithHTTPProtocols(t, testCoordinatorKillDuringRead)
}

func testCoordinatorKillDuringRead(t *testing.T, connFactory resiliencyConnectionFactory) {
	client := prepareResiliencyClient(t, connFactory)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				runCoordinatorKillDuringCursor(tb, client, ctx, col, coordinatorKillReadAfterDocs)
			})
		})
	})
}

// TestResiliency_CoordinatorKillDuringInsert validates that killing the coordinator handling
// active inserts causes temporary failures but the driver remains usable after recovery.
func TestResiliency_CoordinatorKillDuringInsert(t *testing.T) {
	requireResiliencyK8sCoordinatorMode(t)
	runResiliencyWithHTTPProtocols(t, testCoordinatorKillDuringInsert)
}

func testCoordinatorKillDuringInsert(t *testing.T, connFactory resiliencyConnectionFactory) {
	client := prepareResiliencyClient(t, connFactory)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				stats := &insertWorkloadStats{}
				workloadCtx, stopWorkload := context.WithCancel(ctx)
				workloadDone := make(chan struct{})

				go func() {
					defer close(workloadDone)
					runInsertWorkload(workloadCtx, col, stats)
				}()

				waitForInsertSuccesses(tb, stats, true, 5, 2*time.Minute)

				stats.markRestartStarted()
				killCoordinatorForClient(tb, client)
				ensureCoordinatorsRecovered(tb, client)
				stats.markRecoveryReady()

				waitForInsertSuccesses(tb, stats, false, 5, 3*time.Minute)

				stopWorkload()
				select {
				case <-workloadDone:
				case <-time.After(30 * time.Second):
					require.Fail(tb, "insert workload goroutine did not stop after cancellation; possible hang")
				}

				assertInsertWorkloadRecovered(tb, stats)
			})
		})
	})
}

// TestResiliency_CoordinatorKillDuringCursorIteration validates that killing the coordinator
// during cursor iteration fails cleanly instead of panicking or hanging.
func TestResiliency_CoordinatorKillDuringCursorIteration(t *testing.T) {
	requireResiliencyK8sCoordinatorMode(t)
	runResiliencyWithHTTPProtocols(t, testCoordinatorKillDuringCursorIteration)
}

func testCoordinatorKillDuringCursorIteration(t *testing.T, connFactory resiliencyConnectionFactory) {
	client := prepareResiliencyClient(t, connFactory)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				runCoordinatorKillDuringCursor(tb, client, ctx, col, coordinatorKillCursorAfterDocs)
			})
		})
	})
}

// runCoordinatorKillDuringCursor seeds a slow streaming cursor, kills the serving coordinator
// after killAfterDocs documents, and verifies the cursor fails cleanly and cannot resume.
func runCoordinatorKillDuringCursor(
	tb testing.TB,
	client arangodb.Client,
	ctx context.Context,
	col arangodb.Collection,
	killAfterDocs int32,
) {
	tb.Helper()

	query := coordinatorSlowReadQuery(col.Name())
	readCtx, cancelRead := context.WithCancel(ctx)
	defer cancelRead()

	iterationFailed := make(chan error, 1)
	docsRead := atomic.Int32{}
	killNow := make(chan struct{}, 1)
	killAck := make(chan struct{})
	cursorReady := make(chan arangodb.Cursor, 1)

	go func() {
		var cursor arangodb.Cursor
		openDeadline := time.Now().Add(coordinatorKillCursorOpenTimeout)
		for {
			c, err := col.Database().Query(readCtx, query, &arangodb.QueryOptions{
				BatchSize: 1,
				Options: arangodb.QuerySubOptions{
					Stream: true,
				},
			})
			if err == nil {
				cursor = c
				break
			}
			// After a prior kill/recover, Query can hit "Coordinator soft shutdown ongoing"
			// even when pods are Ready and CreateDatabase already succeeded.
			if !isPreCursorOpenTransientError(err) || time.Now().After(openDeadline) || readCtx.Err() != nil {
				iterationFailed <- err
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		cursorReady <- cursor

		for {
			var doc map[string]any
			_, err := cursor.ReadDocument(readCtx, &doc)
			if shared.IsNoMoreDocuments(err) {
				iterationFailed <- fmt.Errorf("cursor finished before coordinator kill (read %d docs)", docsRead.Load())
				return
			}
			if err != nil {
				iterationFailed <- err
				return
			}

			if docsRead.Add(1) == killAfterDocs {
				killNow <- struct{}{}
				<-killAck
			}
		}
	}()

	var cursor arangodb.Cursor
	select {
	case cursor = <-cursorReady:
	case err := <-iterationFailed:
		require.Fail(tb, fmt.Sprintf("cursor failed before coordinator kill: %v", err))
	case <-time.After(coordinatorKillCursorOpenTimeout):
		require.Fail(tb, "cursor did not open before timeout; possible hang")
	}

	select {
	case <-killNow:
		killAllCoordinators(tb)
		close(killAck)
	case err := <-iterationFailed:
		require.Fail(tb, fmt.Sprintf("cursor finished before kill threshold (%d docs): %v", killAfterDocs, err))
	case <-time.After(5 * time.Minute):
		require.Fail(tb, fmt.Sprintf("cursor iteration did not reach kill threshold (%d docs) before timeout", killAfterDocs))
	}

	select {
	case err := <-iterationFailed:
		assertCoordinatorKillInterrupted(tb, err)
	case <-time.After(coordinatorKillOperationTimeout):
		require.Fail(tb, "expected cursor iteration to fail after coordinator kill; possible hang")
	}

	// Dead-cursor checks run while coordinators may still be down (503/410 are expected).
	assertDeadCursorDoesNotResume(tb, cursor, ctx)

	tb.Cleanup(func() {
		ensureCoordinatorsRecovered(tb, client)
	})
	ensureCoordinatorsRecovered(tb, client)
	closeDeadCursor(tb, cursor)

	cancelRead()
}

// closeDeadCursor closes a cursor after coordinator kill. The server may already have dropped it (404/410).
func closeDeadCursor(tb testing.TB, cursor arangodb.Cursor) {
	tb.Helper()
	err := cursor.Close()
	if err == nil {
		return
	}
	tb.Logf("cursor close on dead cursor: %v", err)
	require.True(tb, isDeadCursorError(err), "unexpected cursor close error: %v", err)
}

// assertCoordinatorKillInterrupted verifies the in-flight cursor failed cleanly after coordinator kill.
func assertCoordinatorKillInterrupted(tb testing.TB, err error) {
	tb.Helper()
	require.Error(tb, err)
	if strings.Contains(err.Error(), "finished before coordinator kill") {
		require.Fail(tb, "coordinator kill happened too late: %v", err)
	}
	tb.Logf("cursor interrupted after coordinator kill: %v", err)
	require.True(tb, isCoordinatorKillInterruptedError(err),
		"expected streaming interrupt or cursor-gone error after coordinator kill, got: %v", err)
}

// assertDeadCursorDoesNotResume verifies a killed cursor cannot continue after cluster recovery.
func assertDeadCursorDoesNotResume(tb testing.TB, cursor arangodb.Cursor, parentCtx context.Context) {
	tb.Helper()
	require.NotNil(tb, cursor)

	resumeCtx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	var doc map[string]any
	_, err := cursor.ReadDocument(resumeCtx, &doc)
	require.Error(tb, err, "expected dead cursor to fail on resume after coordinator kill")
	tb.Logf("dead cursor resume error: %v", err)
	require.True(tb, isDeadCursorError(err),
		"expected dead cursor resume error, got: %v", err)
}
