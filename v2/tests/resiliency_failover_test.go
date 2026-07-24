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

const resiliencyFailoverResponseTimeout = 90 * time.Second

// TestResiliency_1_IngressCoordinatorFailover kills a coordinator while the driver
// reaches the cluster through ingress, verifies requests become available again, and
// checks kube-arangodb restores the deployment. Coordinator routing change is logged
// but not strictly asserted because ingress/LB behavior is less predictable.
func TestResiliency_1_IngressCoordinatorFailover(t *testing.T) {
	requireResiliencyK8sCoordinatorMode(t)
	waitForMinimumIngressBackends(t, 2, coordinatorRecoveryTimeout)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		t.Run("shared HTTP/1 connection", func(t *testing.T) {
			runIngressCoordinatorFailover(ctx, t, connectionJsonHttp, connectionJsonHttpFresh)
		})
		t.Run("shared HTTP/2 connection", func(t *testing.T) {
			runIngressCoordinatorFailover(ctx, t, connectionJsonHttp2, connectionJsonHttp2)
		})
	})
}

func runIngressCoordinatorFailover(
	ctx context.Context,
	t *testing.T,
	newClientConn resiliencyConnectionFactory,
	freshProbeConn resiliencyConnectionFactory,
) {
	t.Helper()

	// Ensure pods from a prior protocol's kill/recover are ready before Health()-based kill.
	if isK8S() {
		waitForCoordinatorsReady(t, expectedCoordinatorCount())
	}

	client := newResiliencyClient(t, newClientConn(t))
	requireMinimumCoordinators(t, client, 2)

	expectedCoordinators := expectedCoordinatorCount()
	if expectedCoordinators < 2 {
		expectedCoordinators = countHealthyCoordinators(ctx, client)
	}
	t.Logf("Cluster has %d coordinators (health reports %d)",
		expectedCoordinators, countHealthyCoordinators(ctx, client))

	baselineCoordinatorID := waitForCoordinatorResponse(t, resiliencyFailoverResponseTimeout, freshProbeConn)
	t.Logf("Baseline coordinator before kill: %s", baselineCoordinatorID)

	target := killRandomCoordinator(t, client)
	t.Logf("Killed coordinator pod %s (server %s)", target.ResourceName, target.ServerID)

	newCoordinatorID := waitForCoordinatorResponse(t, resiliencyFailoverResponseTimeout, freshProbeConn)
	t.Logf("Coordinator after kill: %s", newCoordinatorID)
	if newCoordinatorID == baselineCoordinatorID {
		t.Logf("Coordinator ID unchanged after kill; valid when ingress/LB keeps routing to a surviving coordinator")
	} else {
		t.Logf("Coordinator ID changed from %s to %s after kill", baselineCoordinatorID, newCoordinatorID)
	}

	ensureCoordinatorsRecovered(t, client)
	// Do not use a single-shot countHealthyCoordinators here: Health() can briefly fail or
	// return an empty map after pod recreate while Version() already works (expected 3, got 0).
	waitForHealthyCoordinatorCount(t, client, expectedCoordinators, coordinatorRecoveryTimeout)

	finalCoordinatorID := waitForCoordinatorResponse(t, resiliencyFailoverResponseTimeout, freshProbeConn)
	t.Logf("Coordinator after operator recovery: %s", finalCoordinatorID)
}

// waitForCoordinatorResponse retries GET /_admin/status through fresh clients (one per
// attempt) until a coordinator responds. Fresh connections avoid stale TCP sessions to a
// killed backend. freshProbeConn should match the protocol under test (HTTP/1 or HTTP/2).
func waitForCoordinatorResponse(
	t testing.TB,
	timeout time.Duration,
	freshProbeConn resiliencyConnectionFactory,
	excludeCoordinatorIDs ...string,
) string {
	t.Helper()

	var coordinatorID string
	err := NewTimeout(func() error {
		return withContext(5*time.Second, func(reqCtx context.Context) error {
			perRequestClient := newResiliencyClient(t, freshProbeConn(t))
			id, err := respondingCoordinatorID(reqCtx, perRequestClient)
			if err != nil {
				if isResiliencyTransientError(err) {
					t.Logf("Waiting for coordinator response: %v", err)
					return nil
				}
				t.Logf("Waiting for coordinator response: %v", err)
				return nil
			}

			for _, exclude := range excludeCoordinatorIDs {
				if id == exclude {
					t.Logf("Waiting for coordinator response: still routed to excluded coordinator %s", exclude)
					return nil
				}
			}

			coordinatorID = id
			return Interrupt{}
		})
	}).Timeout(timeout, 500*time.Millisecond)

	require.NoError(t, err, "coordinator did not respond within %s", timeout)
	require.NotEmpty(t, coordinatorID)
	return coordinatorID
}
