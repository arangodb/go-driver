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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

const (
	// Slow-query kill tests only need well more than coordinatorKillCursorAfterDocs rows.
	// Keep this modest to limit insert/SORT cost (each test runs for HTTP/1 and HTTP/2).
	coordinatorKillSlowQueryDocCount = 200
	coordinatorKillCursorAfterDocs   = 30
	coordinatorKillOperationTimeout  = 90 * time.Second
	coordinatorKillCursorOpenTimeout = 2 * time.Minute
	// Native AQL burn loop per document (no V8/SLEEP). Increase if kills happen too early on fast CI.
	coordinatorSlowReadQueryBurnIterations = 100
)

// coordinatorSlowReadQuery returns an AQL query that keeps the coordinator busy long enough
// to kill it mid-cursor. Uses native AQL only (no SLEEP/V8) so it stays compatible with
// ArangoDB 4.0 where JavaScript is not available.
func coordinatorSlowReadQuery(collectionName string) string {
	return fmt.Sprintf(
		"FOR doc IN `%s` SORT doc.value LET burn = (FOR i IN 1..%d LET x = MD5(CONCAT(TO_STRING(doc.value), TO_STRING(i))) FILTER x != null RETURN 1) RETURN doc",
		collectionName,
		coordinatorSlowReadQueryBurnIterations,
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
				err := arangodb.CreateDocuments(ctx, col, coordinatorKillSlowQueryDocCount, func(index int) any {
					return map[string]any{"value": index}
				})
				require.NoError(tb, err)

				query := coordinatorSlowReadQuery(col.Name())
				readCtx, cancelRead := context.WithCancel(ctx)
				defer cancelRead()

				readFailed := make(chan error, 1)
				cursorOpen := make(chan struct{})
				docsRead := atomic.Int32{}

				go func() {
					cursor, err := db.Query(readCtx, query, &arangodb.QueryOptions{
						BatchSize: 1,
					})
					if err != nil {
						readFailed <- err
						return
					}
					defer cursor.Close()
					close(cursorOpen)

					for {
						var doc map[string]any
						_, err := cursor.ReadDocument(readCtx, &doc)
						if shared.IsNoMoreDocuments(err) {
							readFailed <- fmt.Errorf("cursor finished before coordinator kill (read %d docs)", docsRead.Load())
							return
						}
						if err != nil {
							readFailed <- err
							return
						}
						docsRead.Add(1)
					}
				}()

				select {
				case <-cursorOpen:
				case err := <-readFailed:
					require.Fail(tb, "cursor failed before coordinator kill: %v", err)
				case <-time.After(coordinatorKillCursorOpenTimeout):
					require.Fail(tb, "cursor did not open before timeout; possible hang")
				}

				killCoordinatorForClient(tb, client)

				select {
				case err := <-readFailed:
					require.Error(tb, err)
					tb.Logf("read failed as expected after coordinator kill: %v", err)
				case <-time.After(coordinatorKillOperationTimeout):
					require.Fail(tb, "expected active read to fail after coordinator kill; possible hang")
				}

				cancelRead()
				ensureCoordinatorsRecovered(tb, client)
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
				err := arangodb.CreateDocuments(ctx, col, coordinatorKillSlowQueryDocCount, func(index int) any {
					return map[string]any{"value": index}
				})
				require.NoError(tb, err)

				query := coordinatorSlowReadQuery(col.Name())
				readCtx, cancelRead := context.WithCancel(ctx)
				defer cancelRead()

				// Channels coordinate the main test goroutine and the background cursor reader.
				// cursorOpen      — reader calls close(cursorOpen) immediately after db.Query succeeds
				// killNow         — reader sends this after coordinatorKillCursorAfterDocs ReadDocument calls
				// iterationFailed — reader sends this when the read loop exits (error or finished too early)
				iterationFailed := make(chan error, 1)
				docsRead := atomic.Int32{}
				killNow := make(chan struct{}, 1)
				cursorOpen := make(chan struct{})

				go func() {
					cursor, err := db.Query(readCtx, query, &arangodb.QueryOptions{
						BatchSize: 1,
					})
					if err != nil {
						iterationFailed <- err
						return
					}
					defer cursor.Close()
					// Query() returned a cursor handle; iteration has not started yet.
					close(cursorOpen)

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

						if docsRead.Add(1) == coordinatorKillCursorAfterDocs {
							killNow <- struct{}{} // signals: read 30 docs; main should kill coordinator now
						}
					}
				}()

				// Phase 1: wait until db.Query succeeds and the reader closes cursorOpen.
				select {
				case <-cursorOpen:
				case err := <-iterationFailed:
					require.Fail(tb, "cursor failed before coordinator kill: %v", err)
				case <-time.After(coordinatorKillCursorOpenTimeout):
					require.Fail(tb, "cursor did not open before timeout; possible hang")
				}

				// Phase 2: wait until 30 documents are read, then kill the serving coordinator.
				select {
				case <-killNow:
					killCoordinatorForClient(tb, client)
				case err := <-iterationFailed:
					require.Fail(tb, "cursor finished before kill threshold: %v", err)
				case <-time.After(5 * time.Minute):
					require.Fail(tb, "cursor iteration did not reach kill threshold before timeout")
				}

				// Phase 3: the in-flight cursor must fail cleanly (not hang); same client recovers later.
				select {
				case err := <-iterationFailed:
					require.Error(tb, err)
					tb.Logf("cursor iteration failed as expected after coordinator kill: %v", err)
				case <-time.After(coordinatorKillOperationTimeout):
					require.Fail(tb, "expected cursor iteration to fail after coordinator kill; possible hang")
				}

				cancelRead()
				ensureCoordinatorsRecovered(tb, client)
			})
		})
	})
}
