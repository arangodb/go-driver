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

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

// TestResiliency_0_LoadBalancerCoordinatorDistributionInCluster checks ClientIP
// session affinity when the driver reaches coordinators through the internal
// Kubernetes Service (kube-proxy), not through nginx ingress.
//
// Request flow:
//
//	go-driver (in-cluster Job pod)
//	  → http://<deployment>.<namespace>.svc.cluster.local:8529
//	  → Service ClusterIP
//	  → kube-proxy (sessionAffinity: ClientIP)
//	  → one coordinator pod
//	  → GET /_admin/status
//
// With ClientIP enabled on the coordinator Service, every request from this pod
// should hit the same coordinator — including fresh HTTP/1 clients (same source
// pod IP). See v2/tests/k8s-resiliency-access-modes.md.
//
// Run via: make run-k8s-v2-resiliency-incluster
func TestResiliency_0_LoadBalancerCoordinatorDistributionInCluster(t *testing.T) {
	requireResiliencyEnabled(t)
	requireClusterMode(t)
	requireK8SInCluster(t)

	conn := connectionJsonHttp2(t)
	client := newResiliencyClient(t, conn)
	requireMinimumCoordinators(t, client, resiliencyLoadBalancerRequests)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		expectedCoordinators := coordinatorCount(ctx, t, client)
		t.Logf("Cluster has %d coordinators", expectedCoordinators)
		t.Logf("In-cluster service endpoint: %v", client.Connection().GetEndpoint().List())

		t.Run("shared HTTP/2 connection", func(t *testing.T) {
			assertAllRequestsSameCoordinator(ctx, t, client, "shared HTTP/2 connection")
		})

		t.Run("shared HTTP/1 with same client connection", func(t *testing.T) {
			http1Client := newResiliencyClient(t, connectionJsonHttp(t))
			assertAllRequestsSameCoordinator(ctx, t, http1Client, "shared HTTP/1 connection")
		})

		t.Run("fresh HTTP/1 connection per request", func(t *testing.T) {
			ids := collectCoordinatorIDsViaFreshHTTP1(ctx, t, resiliencyLoadBalancerRequests)
			assertSingleCoordinator(t, "fresh HTTP/1 connection per request", ids)
		})

		t.Run("fresh HTTP/2 connection per request", func(t *testing.T) {
			ids := collectCoordinatorIDsViaFreshHTTP2(ctx, t, resiliencyLoadBalancerRequests)
			assertSingleCoordinator(t, "fresh HTTP/2 connection per request", ids)
		})
	})
}

func requireK8SInCluster(t testing.TB) {
	t.Helper()
	if !isK8SInCluster() {
		t.Skip("in-cluster load balancer test requires TEST_MODE_K8S=k8s-incluster")
	}
}

func assertAllRequestsSameCoordinator(ctx context.Context, t *testing.T, client arangodb.Client, mode string) {
	t.Helper()

	ids := make([]string, 0, resiliencyLoadBalancerRequests)
	for i := 0; i < resiliencyLoadBalancerRequests; i++ {
		id, err := respondingCoordinatorID(ctx, client)
		require.NoError(t, err)
		ids = append(ids, id)
		t.Logf("Request %d via %s -> coordinator %s", i+1, mode, id)
	}
	assertSingleCoordinator(t, mode, ids)
}

func assertSingleCoordinator(t testing.TB, mode string, ids []string) {
	t.Helper()

	unique := uniqueStrings(ids)
	logCoordinatorDistribution(t, mode, ids, unique)
	require.Len(t, unique, 1,
		"%s: expected ClientIP session affinity to route all requests to one coordinator", mode)
	t.Logf("All %d requests hit coordinator %s (ClientIP session affinity)", len(ids), unique[0])
}
