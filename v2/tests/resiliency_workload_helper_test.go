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
	failuresBefore   int
	failuresDuring   int
	failuresAfter    int
	successesBefore  int
	successesAfter   int
	successesPending int
	restartStarted   bool
	ingressReady     bool
	duringErrors     []error
}

const maxWorkloadDuringErrors = 32

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
		// Success during the fault window is tracked separately and must not count as recovery.
		s.successesPending++
	}
}

// recordFailure increments the failure counter for the current restart phase.
// Failures after pods/ingress are marked ready but before the first post-recovery
// success are treated as settling (during), not pathological after-recovery failures —
// coordinator soft-shutdown often keeps Version() failing briefly after Ready=true.
func (s *versionWorkloadStats) recordFailure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failures++
	switch {
	case !s.restartStarted:
		s.failuresBefore++
	case s.ingressReady && s.successesAfter > 0:
		s.failuresAfter++
	default:
		s.failuresDuring++
		s.captureDuringError(err)
	}
}

func (s *versionWorkloadStats) captureDuringError(err error) {
	if err == nil || len(s.duringErrors) >= maxWorkloadDuringErrors {
		return
	}
	s.duringErrors = append(s.duringErrors, err)
}

// markRestartStarted records that the ingress restart has begun.
func (s *versionWorkloadStats) markRestartStarted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restartStarted = true
}

// markIngressReady records that the ingress controller rollout has completed.
// Successes observed during the fault window stay in successesPending and do not
// count toward successesAfter — recovery requires new successes after ready.
func (s *versionWorkloadStats) markIngressReady() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ingressReady = true
}

// snapshot returns a thread-safe copy of the before/after success and failure counts.
func (s *versionWorkloadStats) snapshot() (successesBefore, successesAfter, failures int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.successesBefore, s.successesAfter, s.failures
}

func (s *versionWorkloadStats) failureSnapshot() (failuresBefore, failuresDuring, failuresAfter int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failuresBefore, s.failuresDuring, s.failuresAfter
}

func (s *versionWorkloadStats) duringFailureSnapshot() []error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]error, len(s.duringErrors))
	copy(out, s.duringErrors)
	return out
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
			stats.recordFailure(err)
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
	failuresBefore, failuresDuring, failuresAfter := stats.failureSnapshot()
	require.Greater(t, stats.totalAttempts(), 0, "workload did not issue any requests")
	require.GreaterOrEqual(t, successesBefore, 1, "expected at least one successful request before ingress restart")
	require.GreaterOrEqual(t, successesAfter, 1, "expected at least one successful request after ingress recovery")
	require.GreaterOrEqual(t, successesBefore, failuresBefore,
		"unexpected failures before ingress restart")
	require.GreaterOrEqual(t, successesAfter, failuresAfter,
		"unexpected failures after ingress recovery; possible pathological failure rate")
	t.Logf("workload summary: successesBefore=%d successesAfter=%d failures=%d (before=%d during=%d after=%d) totalAttempts=%d",
		successesBefore, successesAfter, failures, failuresBefore, failuresDuring, failuresAfter, stats.totalAttempts())
	assertResiliencyDuringErrors(t, stats.duringFailureSnapshot())
}
