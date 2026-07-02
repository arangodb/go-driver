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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestResiliency_IngressRestartWhileIdle validates that a connected client (no concurrent
// request loop) can execute requests again after the ingress-nginx controller is restarted.
func TestResiliency_IngressRestartWhileIdle(t *testing.T) {
	requireResiliencyK8sIngressMode(t)
	requireKubectl(t)

	runResiliencyWithHTTPProtocols(t, testIngressRestartWhileIdle)
}

// testIngressRestartWhileIdle connects through ingress without an active workload loop,
// restarts the controller, and verifies the same client can query again after recovery.
func testIngressRestartWhileIdle(t *testing.T, connFactory resiliencyConnectionFactory) {
	client := newResiliencyClient(t, connFactory(t))

	waitForSuccessfulVersion(t, client, 2*time.Minute)

	restartIngressController(t)
	waitForIngressControllerReady(t)

	waitForSuccessfulVersion(t, client, 2*time.Minute)
}

// TestResiliency_IngressRestartDuringActiveWorkload validates that a client issuing
// continuous requests through ingress survives an ingress-nginx controller restart.
// Temporary request failures during the restart window are expected.
func TestResiliency_IngressRestartDuringActiveWorkload(t *testing.T) {
	requireResiliencyK8sIngressMode(t)
	requireKubectl(t)

	runResiliencyWithHTTPProtocols(t, testIngressRestartDuringActiveWorkload)
}

// testIngressRestartDuringActiveWorkload keeps requests running while ingress is restarted and recovered.
func testIngressRestartDuringActiveWorkload(t *testing.T, connFactory resiliencyConnectionFactory) {
	client := newResiliencyClient(t, connFactory(t))

	waitForSuccessfulVersion(t, client, 2*time.Minute)

	stats := &versionWorkloadStats{}
	workloadCtx, stopWorkload := context.WithCancel(context.Background())
	workloadDone := make(chan struct{})

	go func() {
		defer close(workloadDone)
		runVersionWorkload(workloadCtx, client, stats)
	}()

	waitForWorkloadSuccesses(t, stats, true, 1, 2*time.Minute)

	stats.markRestartStarted()
	restartIngressController(t)
	waitForIngressControllerReady(t)
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
