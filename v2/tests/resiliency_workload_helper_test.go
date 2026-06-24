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

// versionWorkloadStats tracks request outcomes across the ingress restart timeline.
type versionWorkloadStats struct {
	mu sync.Mutex

	successes        int
	failures         int
	successesBefore  int
	successesAfter   int
	successesPending int
	restartStarted   bool
	ingressReady     bool
}

// recordSuccess increments success counters for the current restart phase.
func (s *versionWorkloadStats) recordSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.successes++
	switch {
	case !s.restartStarted:
		s.successesBefore++
	case s.ingressReady:
		s.successesAfter++
	default:
		// Success while restart is in progress; credited to "after" on markIngressReady().
		s.successesPending++
	}
}

// recordFailure increments the failure counter; failures during restart are expected.
func (s *versionWorkloadStats) recordFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures++
}

// markRestartStarted records that the ingress restart has begun.
func (s *versionWorkloadStats) markRestartStarted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restartStarted = true
}

// markIngressReady records that the ingress controller rollout has completed and
// credits any successes observed during the restart window toward post-recovery.
func (s *versionWorkloadStats) markIngressReady() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ingressReady = true
	if s.successesPending > 0 {
		s.successesAfter += s.successesPending
		s.successesPending = 0
	}
}

// snapshot returns a thread-safe copy of the before/after success and failure counts.
func (s *versionWorkloadStats) snapshot() (successesBefore, successesAfter, failures int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.successesBefore, s.successesAfter, s.failures
}

// totalAttempts returns the number of workload requests issued (successes and failures).
func (s *versionWorkloadStats) totalAttempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.successes + s.failures
}

const versionWorkloadInterval = 100 * time.Millisecond

// runVersionWorkload issues continuous client.Version requests until ctx is cancelled.
func runVersionWorkload(ctx context.Context, client arangodb.Client, stats *versionWorkloadStats) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}

		reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err := client.Version(reqCtx)
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
		case <-time.After(versionWorkloadInterval):
		}
	}
}

// waitForWorkloadSuccesses blocks until the workload reaches minSuccesses before or after restart.
func waitForWorkloadSuccesses(
	t testing.TB,
	stats *versionWorkloadStats,
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

// assertWorkloadRecovered verifies that requests succeeded both before and after ingress recovery.
func assertWorkloadRecovered(t testing.TB, stats *versionWorkloadStats) {
	t.Helper()

	successesBefore, successesAfter, failures := stats.snapshot()
	require.Greater(t, stats.totalAttempts(), 0, "workload did not issue any requests")
	require.GreaterOrEqual(t, successesBefore, 1, "expected at least one successful request before ingress restart")
	require.GreaterOrEqual(t, successesAfter, 1, "expected at least one successful request after ingress recovery")
	t.Logf("workload summary: successesBefore=%d successesAfter=%d failures=%d totalAttempts=%d",
		successesBefore, successesAfter, failures, stats.totalAttempts())
}
