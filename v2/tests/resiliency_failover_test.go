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

	"github.com/arangodb/go-driver/v2/connection"
)

// resiliencyConnectionFactory builds a driver connection for resiliency scenarios.
type resiliencyConnectionFactory func(testing.TB) connection.Connection

const resiliencyFailoverResponseTimeout = 90 * time.Second

// TestResiliency_1_InClusterCoordinatorFailover kills the coordinator that ClientIP
// session affinity selected for this client pod, then verifies traffic fails over to
// another coordinator and kube-arangodb restores the deployment.
//
// Run via: make run-k8s-v2-resiliency-incluster
func TestResiliency_1_InClusterCoordinatorFailover(t *testing.T) {
	requireResiliencyEnabled(t)
	requireClusterMode(t)
	requireK8SInCluster(t)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		t.Run("shared HTTP/1 connection", func(t *testing.T) {
			runInClusterCoordinatorFailover(ctx, t, connectionJsonHttp, connectionJsonHttpFresh)
		})
		t.Run("shared HTTP/2 connection", func(t *testing.T) {
			runInClusterCoordinatorFailover(ctx, t, connectionJsonHttp2, connectionJsonHttp2)
		})
	})
}

func runInClusterCoordinatorFailover(
	ctx context.Context,
	t *testing.T,
	newClientConn resiliencyConnectionFactory,
	freshProbeConn resiliencyConnectionFactory,
) {
	t.Helper()

	client := newResiliencyClient(t, newClientConn(t))
	requireMinimumCoordinators(t, client, 2)

	expectedCoordinators := coordinatorCount(ctx, t, client)
	t.Logf("Cluster has %d coordinators", expectedCoordinators)

	stickyCoordinatorID := establishStickyCoordinatorID(ctx, t)
	t.Logf("Sticky coordinator before kill: %s", stickyCoordinatorID)

	chaos := NewChaosController(t)
	target, err := chaos.KillCoordinatorByServerID(ctx, client, stickyCoordinatorID)
	require.NoError(t, err)
	t.Logf("Killed coordinator pod %s (server %s)", target.ResourceName, target.ServerID)

	newCoordinatorID := waitForCoordinatorResponse(ctx, t, resiliencyFailoverResponseTimeout, freshProbeConn, stickyCoordinatorID)
	require.NotEqual(t, stickyCoordinatorID, newCoordinatorID,
		"after killing sticky coordinator, requests should fail over to a different coordinator")
	t.Logf("Failover coordinator after kill: %s", newCoordinatorID)

	chaos.WaitForClusterRecovery(client, resiliencyClusterRecoveryTimeout)
	require.Equal(t, expectedCoordinators, coordinatorCount(ctx, t, client),
		"operator should restore the original coordinator count")

	postRecoveryID := establishStickyCoordinatorID(ctx, t)
	t.Logf("Sticky coordinator after recovery: %s", postRecoveryID)
}

// TestResiliency_1_IngressCoordinatorFailover kills a coordinator while the driver
// reaches the cluster through ingress, verifies requests become available again, and
// checks kube-arangodb restores the deployment. Coordinator routing change is logged
// but not strictly asserted because ingress/LB behavior is less predictable.
//
// Run via: make run-k8s-v2-resiliency
func TestResiliency_1_IngressCoordinatorFailover(t *testing.T) {
	requireResiliencyEnabled(t)
	requireClusterMode(t)
	requireK8SIngress(t)

	waitForMinimumIngressBackends(t, 2, resiliencyClusterRecoveryTimeout)

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

	client := newResiliencyClient(t, newClientConn(t))
	requireMinimumCoordinators(t, client, 2)

	expectedCoordinators := coordinatorCount(ctx, t, client)
	t.Logf("Cluster has %d coordinators", expectedCoordinators)

	baselineCoordinatorID := waitForCoordinatorResponse(ctx, t, resiliencyFailoverResponseTimeout, freshProbeConn)
	t.Logf("Baseline coordinator before kill: %s", baselineCoordinatorID)

	chaos := NewChaosController(t)
	target, err := chaos.KillRandomCoordinator(ctx, client)
	require.NoError(t, err)
	t.Logf("Killed coordinator pod %s (server %s)", target.ResourceName, target.ServerID)

	newCoordinatorID := waitForCoordinatorResponse(ctx, t, resiliencyFailoverResponseTimeout, freshProbeConn)
	t.Logf("Coordinator after kill: %s", newCoordinatorID)
	if newCoordinatorID == baselineCoordinatorID {
		t.Logf("Coordinator ID unchanged after kill; valid when ingress/LB keeps routing to a surviving coordinator")
	} else {
		t.Logf("Coordinator ID changed from %s to %s after kill", baselineCoordinatorID, newCoordinatorID)
	}

	chaos.WaitForClusterRecovery(client, resiliencyClusterRecoveryTimeout)
	require.Equal(t, expectedCoordinators, coordinatorCount(ctx, t, client),
		"operator should restore the original coordinator count")

	finalCoordinatorID := waitForCoordinatorResponse(ctx, t, resiliencyFailoverResponseTimeout, freshProbeConn)
	t.Logf("Coordinator after operator recovery: %s", finalCoordinatorID)
}

func establishStickyCoordinatorID(ctx context.Context, t testing.TB) string {
	t.Helper()

	ids := collectCoordinatorIDsViaFreshHTTP1(ctx, t, resiliencyLoadBalancerRequests)
	assertSingleCoordinator(t, "establish sticky coordinator", ids)
	return ids[0]
}

// waitForCoordinatorResponse retries GET /_admin/status through fresh clients (one per
// attempt) until a coordinator responds. Fresh connections avoid stale TCP sessions to a
// killed backend. freshProbeConn should match the protocol under test (HTTP/1 or HTTP/2).
// Optional excludeCoordinatorIDs keeps retrying while routing still hits a coordinator that
// must no longer serve traffic (e.g. sticky target before kube-proxy/endpoints update).
func waitForCoordinatorResponse(
	ctx context.Context,
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
				if isRetryableConnectionError(err) {
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
