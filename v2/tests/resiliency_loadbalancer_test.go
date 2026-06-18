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

const resiliencyLoadBalancerRequests = 3

// TestResiliency_0_LoadBalancerCoordinatorDistribution checks how ingress/LB
// routes driver HTTP traffic across coordinators.
//
// Request flow (all HTTP from this test):
//
//	go-driver client
//	  → https://arangodb.local  (TEST_ENDPOINTS_OVERRIDE, one ingress URL)
//	  → nginx ingress  (may round-robin independently of downstream affinity)
//	  → kube-arangodb external-access Service (often sessionAffinity: ClientIP)
//	  → one coordinator pod
//	  → GET /_admin/status  (via client.GetServerStatus)
//	  → response.serverInfo.serverId  (e.g. CRDN-ppwdeatg)
//
// Session affinity is NOT a guarantee of fixed routing:
//   - ClientIP on the coordinator Service keys off the source IP seen at that
//     Service (often the ingress pod IP, not the end-user app IP).
//   - nginx ingress may open multiple upstream connections or balance per request.
//   - HTTP/2 may multiplex streams; HTTP/1 fresh clients still share client IP.
//   - Only 3 samples — you may see 1, 2, or 3 distinct serverIds.
//
// Typical (not guaranteed) expectations:
//   - HTTP/2 + one shared client: often 1 coordinator (connection-level stickiness).
//   - HTTP/2 + new client per request: new TCP per client; compare with shared HTTP/2.
//   - HTTP/1 + new client per request: often >1 coordinator when ingress has 2+ backends.
//   - Any subtest can produce the opposite pattern; this test logs and observes.
//
// Three requests per subtest (resiliencyLoadBalancerRequests = 3).
// serverId tells which coordinator answered; driver endpoint URL stays the ingress
// address on k8s and does not prove TCP connection reuse by itself.
func TestResiliency_0_LoadBalancerCoordinatorDistribution(t *testing.T) {
	requireResiliencyEnabled(t)
	requireClusterMode(t)
	requireK8SIngress(t)

	conn := connectionJsonHttp2(t)
	client := newResiliencyClient(t, conn)
	requireMinimumCoordinators(t, client, resiliencyLoadBalancerRequests)
	waitForMinimumIngressBackends(t, 2, resiliencyClusterRecoveryTimeout)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		expectedCoordinators := coordinatorCount(ctx, t, client)
		t.Logf("Cluster has %d coordinators", expectedCoordinators)

		// Subtest 1: ONE shared HTTP/2 client, THREE calls to GET /_admin/status.
		// Goal: model a long-lived client (typical app). Often same coordinator;
		// not required — ingress/driver may still spread across coordinators.
		t.Run("shared HTTP/2 connection", func(t *testing.T) {
			ids := make([]string, 0, resiliencyLoadBalancerRequests)
			for i := 0; i < resiliencyLoadBalancerRequests; i++ {
				id, err := respondingCoordinatorID(ctx, client)
				require.NoError(t, err)
				ids = append(ids, id)
				t.Logf("Request %d via shared HTTP/2 client -> coordinator %s", i+1, id)
			}

			unique := uniqueStrings(ids)
			logCoordinatorDistribution(t, "shared HTTP/2 connection", ids, unique)

			if len(unique) == 1 {
				t.Logf("All %d requests hit the same coordinator (common with connection-level LB)", len(ids))
			} else {
				t.Logf("Requests spread across %d coordinators on a shared HTTP/2 connection", len(unique))
			}
		})

		// Subtest 2: ONE shared HTTP/1 client (keep-alive enabled), THREE calls.
		// Compare with subtest 1 (HTTP/2) and subtests 3–4 (new client per request).
		t.Run("shared HTTP/1 with same client connection", func(t *testing.T) {
			ids := make([]string, 0, resiliencyLoadBalancerRequests)
			http1Client := newResiliencyClient(t, connectionJsonHttp(t))
			for i := 0; i < resiliencyLoadBalancerRequests; i++ {
				id, err := respondingCoordinatorID(ctx, http1Client)
				require.NoError(t, err)
				ids = append(ids, id)
				t.Logf("Request %d via shared HTTP/1 client -> coordinator %s", i+1, id)
			}

			unique := uniqueStrings(ids)
			logCoordinatorDistribution(t, "shared HTTP/1 connection", ids, unique)

			if len(unique) == 1 {
				t.Logf("All %d requests hit the same coordinator (common with connection-level LB)", len(ids))
			} else {
				t.Logf("Requests spread across %d coordinators on a shared HTTP/1 connection", len(unique))
			}
		})

		// Subtest 3: THREE separate HTTP/1 clients (DisableKeepAlives — new TCP each time),
		// each sends one GET /_admin/status through the same ingress URL.
		// Goal: observe coordinator selection for fresh HTTP/1 connections.
		// Depending on ingress/load-balancer behavior, requests may hit the same
		// coordinator or multiple coordinators.
		t.Run("fresh HTTP/1 connection per request", func(t *testing.T) {
			ids := collectCoordinatorIDsViaFreshHTTP1(ctx, t, resiliencyLoadBalancerRequests)
			unique := uniqueStrings(ids)

			logCoordinatorDistribution(t, "fresh HTTP/1 connection per request", ids, unique)

			switch len(unique) {
			case 1:
				t.Logf(
					"All %d requests hit coordinator %s. This is valid and may indicate ingress stickiness or load-balancer routing decisions.",
					len(ids),
					unique[0],
				)
			default:
				t.Logf(
					"Requests were distributed across %d coordinators.",
					len(unique),
				)
			}

			if len(unique) == resiliencyLoadBalancerRequests &&
				expectedCoordinators >= resiliencyLoadBalancerRequests {
				t.Logf(
					"Each of the %d requests reached a distinct coordinator.",
					resiliencyLoadBalancerRequests,
				)
			}
		})

		// Subtest 4: THREE separate HTTP/2 clients (new Transport each time → new TCP per client).
		// Compare with subtest 1 (one shared HTTP/2 client) to see if connection reuse changes routing.
		t.Run("fresh HTTP/2 connection per request", func(t *testing.T) {
			ids := collectCoordinatorIDsViaFreshHTTP2(ctx, t, resiliencyLoadBalancerRequests)
			unique := uniqueStrings(ids)

			logCoordinatorDistribution(t, "fresh HTTP/2 connection per request", ids, unique)

			switch len(unique) {
			case 1:
				t.Logf(
					"All %d requests hit coordinator %s.",
					len(ids),
					unique[0],
				)
			default:
				t.Logf(
					"Requests were distributed across %d coordinators.",
					len(unique),
				)
			}

			if len(unique) == resiliencyLoadBalancerRequests &&
				expectedCoordinators >= resiliencyLoadBalancerRequests {
				t.Logf(
					"Each of the %d requests reached a distinct coordinator.",
					resiliencyLoadBalancerRequests,
				)
			}
		})
	})
}

func collectCoordinatorIDsViaFreshHTTP1(ctx context.Context, t testing.TB, count int) []string {
	t.Helper()

	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		perRequestClient := newResiliencyClient(t, connectionJsonHttpFresh(t))
		id, err := respondingCoordinatorID(ctx, perRequestClient)
		require.NoError(t, err)
		ids = append(ids, id)
		t.Logf("Request %d via new HTTP/1 client -> coordinator %s", i+1, id)
	}
	return ids
}

// collectCoordinatorIDsViaFreshHTTP2 creates a new HTTP/2 transport and
// client per request to compare routing behavior against a shared HTTP/2 client.
func collectCoordinatorIDsViaFreshHTTP2(ctx context.Context, t testing.TB, count int) []string {
	t.Helper()

	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		perRequestClient := newResiliencyClient(t, connectionJsonHttp2(t))
		id, err := respondingCoordinatorID(ctx, perRequestClient)
		require.NoError(t, err)
		ids = append(ids, id)
		t.Logf("Request %d via new HTTP/2 client -> coordinator %s", i+1, id)
	}
	return ids
}

func requireK8SIngress(t testing.TB) {
	t.Helper()
	if !isK8SIngress() {
		if isK8SInCluster() {
			t.Skip("ingress load balancer test skipped in in-cluster mode (TEST_MODE_K8S=k8s-incluster)")
		}
		t.Skip("ingress load balancer test requires TEST_MODE_K8S=k8s (kind + kube-arangodb ingress)")
	}
}

func coordinatorCount(ctx context.Context, t testing.TB, client arangodb.Client) int {
	t.Helper()

	health, err := client.Health(ctx)
	require.NoError(t, err)

	count := 0
	for _, server := range health.Health {
		if server.Role == arangodb.ServerRoleCoordinator {
			count++
		}
	}
	return count
}

// respondingCoordinatorID sends GET /_admin/status through the client connection
// and returns the coordinator serverId from the JSON response.
func respondingCoordinatorID(ctx context.Context, client arangodb.Client) (string, error) {
	status, err := client.GetServerStatus(ctx, "")
	if err != nil {
		return "", err
	}
	if status.ServerInfo.ServerId == nil || *status.ServerInfo.ServerId == "" {
		return "", errMissingCoordinatorServerID
	}
	return *status.ServerInfo.ServerId, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		unique = append(unique, v)
	}
	return unique
}

func logCoordinatorDistribution(t testing.TB, mode string, ids, unique []string) {
	t.Helper()
	t.Logf("%s: server IDs %v (%d unique coordinator(s))", mode, ids, len(unique))
}
