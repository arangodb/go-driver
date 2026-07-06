//go:build toxiproxy

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

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/stretchr/testify/require"
)

// TestToxiproxy_PartialPacketLoss validates intermittent failures under ~30% upstream
// connection loss without panicking, then recovery after toxic removal.
//
// Toxiproxy 2.9 does not ship a packet_loss toxic; reset_peer with toxicity 0.3 applies
// the fault to ~30% of new TCP links. DisableKeepAlives forces a fresh link per request.
func TestToxiproxy_PartialPacketLoss(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testPartialPacketLoss, toxiproxyProtocolConfig{
		http1:     connectionToxiproxyHttpNoKeepAlive,
		skipHTTP2: "HTTP/2 multiplexes one TCP connection; per-request packet loss covered on HTTP/1",
	})
}

// testPartialPacketLoss runs multiple Version() calls while ~30% of new links are reset.
func testPartialPacketLoss(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	recoveryTimeout := toxiproxyRecoveryTimeout()

	client := newToxiproxyClient(t, connFactory)
	waitForSuccessfulVersion(t, client, recoveryTimeout)

	addUpstreamResetPeer(t, proxy, "reset_peer_partial", toxiproxyPartialPacketLossToxicity, 0)
	t.Cleanup(func() {
		_ = proxy.RemoveToxic("reset_peer_partial")
	})

	var failures, successes int
	for i := 0; i < toxiproxyPartialPacketLossAttempts; i++ {
		require.NotPanics(t, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			_, err := versionCallWithFreshConnection(t, connFactory, ctx)
			switch {
			case err != nil:
				t.Logf("error message: %s", err.Error())
				t.Logf("isIntermittentNetworkError: %t", isIntermittentNetworkError(err))
				failures++
				require.True(t, isIntermittentNetworkError(err),
					"expected intermittent transport failure, got: %v", err)
			default:
				successes++
			}
		})
	}

	require.Greater(t, failures, 0, "expected some failures under 30%% packet loss (%d/%d succeeded)",
		successes, toxiproxyPartialPacketLossAttempts)
	require.Greater(t, successes, 0, "expected some successes under 30%% packet loss (%d/%d failed)",
		failures, toxiproxyPartialPacketLossAttempts)

	require.NoError(t, proxy.RemoveToxic("reset_peer_partial"))
	waitForSuccessfulVersion(t, client, recoveryTimeout)
}

// TestToxiproxy_FullPacketLoss validates complete upstream data loss (network outage):
// requests time out, then succeed again after the toxic is removed.
func TestToxiproxy_FullPacketLoss(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testFullPacketLoss)
}

// testFullPacketLoss blocks all upstream data (timeout toxic, toxicity 1.0) until removal.
func testFullPacketLoss(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	recoveryTimeout := toxiproxyRecoveryTimeout()

	client := newToxiproxyClient(t, connFactory)
	waitForSuccessfulVersion(t, client, recoveryTimeout)

	addUpstreamTimeout(t, proxy, "timeout_full", 1.0, 0)
	t.Cleanup(func() {
		_ = proxy.RemoveToxic("timeout_full")
	})

	require.NotPanics(t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), toxiproxyFullPacketLossCallTimeout)
		defer cancel()

		start := time.Now()
		_, err := client.Version(ctx)
		elapsed := time.Since(start)

		require.Error(t, err)
		t.Logf("error message: %s", err.Error())
		require.True(t, isDriverTimeoutError(err) || isIntermittentNetworkError(err),
			"expected timeout or transport failure under 100%% packet loss, got: %v", err)
		require.Less(t, elapsed, toxiproxyFullPacketLossCallTimeout+3*time.Second,
			"expected call to return on context/transport timeout, not hang (elapsed %v)", elapsed)
	})

	require.NoError(t, proxy.RemoveToxic("timeout_full"))
	waitForSuccessfulVersion(t, client, recoveryTimeout)
}

// versionCallWithFreshConnection issues Version() on a new client/connection (no keep-alive pooling).
func versionCallWithFreshConnection(t testing.TB, connFactory toxiproxyConnectionFactory, ctx context.Context) (arangodb.VersionInfo, error) {
	t.Helper()
	return arangodb.NewClient(connFactory(t)).Version(ctx)
}
