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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

// insertWorkloadStats tracks insert outcomes across a coordinator failure timeline.
type insertWorkloadStats struct {
	mu sync.Mutex

	successes        int
	failures         int
	failuresBefore   int
	failuresDuring   int
	failuresAfter    int
	successesBefore  int
	successesAfter   int
	successesPending int
	restartStarted   bool
	recoveryReady    bool
}

func (s *insertWorkloadStats) recordSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.successes++
	switch {
	case !s.restartStarted:
		s.successesBefore++
	case s.recoveryReady:
		s.successesAfter++
	default:
		s.successesPending++
	}
}

func (s *insertWorkloadStats) recordFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failures++
	switch {
	case !s.restartStarted:
		s.failuresBefore++
	case s.recoveryReady:
		s.failuresAfter++
	default:
		s.failuresDuring++
	}
}

func (s *insertWorkloadStats) markRestartStarted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restartStarted = true
}

func (s *insertWorkloadStats) markRecoveryReady() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recoveryReady = true
	if s.successesPending > 0 {
		s.successesAfter += s.successesPending
		s.successesPending = 0
	}
}

func (s *insertWorkloadStats) snapshot() (successesBefore, successesAfter, failures int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.successesBefore, s.successesAfter, s.failures
}

func (s *insertWorkloadStats) failureSnapshot() (failuresBefore, failuresDuring, failuresAfter int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failuresBefore, s.failuresDuring, s.failuresAfter
}

func (s *insertWorkloadStats) totalAttempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.successes + s.failures
}

const insertWorkloadInterval = 50 * time.Millisecond

// runInsertWorkload issues continuous document inserts until ctx is cancelled.
func runInsertWorkload(ctx context.Context, col arangodb.Collection, stats *insertWorkloadStats) {
	counter := 0
	for {
		if err := ctx.Err(); err != nil {
			return
		}

		counter++
		reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err := col.CreateDocument(reqCtx, map[string]any{
			"value": counter,
		})
		cancel()

		if ctx.Err() != nil {
			return
		}

		if err == nil {
			stats.recordSuccess()
		} else {
			stats.recordFailure()
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(insertWorkloadInterval):
		}
	}
}

func waitForInsertSuccesses(
	t testing.TB,
	stats *insertWorkloadStats,
	beforeRestart bool,
	minSuccesses int,
	timeout time.Duration,
) {
	t.Helper()

	NewTimeout(func() error {
		successesBefore, successesAfter, _ := stats.snapshot()
		count := successesBefore
		if !beforeRestart {
			count = successesAfter
		}
		if count >= minSuccesses {
			return Interrupt{}
		}
		return nil
	}).TimeoutT(t, timeout, 250*time.Millisecond)
}

func assertInsertWorkloadRecovered(t testing.TB, stats *insertWorkloadStats) {
	t.Helper()

	successesBefore, successesAfter, failures := stats.snapshot()
	failuresBefore, failuresDuring, failuresAfter := stats.failureSnapshot()
	require.Greater(t, stats.totalAttempts(), 0, "insert workload did not issue any requests")
	require.GreaterOrEqual(t, successesBefore, 1, "expected at least one successful insert before coordinator failure")
	require.GreaterOrEqual(t, successesAfter, 1, "expected at least one successful insert after coordinator recovery")
	require.GreaterOrEqual(t, successesBefore, failuresBefore,
		"unexpected failures before coordinator failure")
	require.GreaterOrEqual(t, successesAfter, failuresAfter,
		"unexpected failures after coordinator recovery; possible pathological failure rate")
	t.Logf("insert workload summary: successesBefore=%d successesAfter=%d failures=%d (before=%d during=%d after=%d) totalAttempts=%d",
		successesBefore, successesAfter, failures, failuresBefore, failuresDuring, failuresAfter, stats.totalAttempts())
}
