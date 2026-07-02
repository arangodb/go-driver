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

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/stretchr/testify/require"
)

// TestToxiproxy_AbruptTCPConnectionClose validates abrupt TCP connection loss through Toxiproxy.
//
// Topology:
//
//	Driver → Proxy "arangodb" (listen 127.0.0.1:17001) → ArangoDB / Ingress (upstream)
//
// Proxy vs toxic:
//   - Proxy: permanent TCP tunnel (listen → upstream). Always present during the test.
//   - Toxic: temporary fault rule on that tunnel. "reset_peer" sends TCP RST to active connections.
//
// Flow:
//  1. Connect and call Version() successfully — establishes a keep-alive connection in the HTTP pool.
//  2. Add reset_peer toxic (upstream, timeout 0) — next traffic through the proxy is reset immediately.
//  3. Call Version() again — must return a transport error (not panic), e.g.:
//     HTTP/1: read tcp ...: read: connection reset by peer
//     HTTP/2: unexpected EOF (HTTP/2 framing sees an abrupt close as incomplete data)
//     The 15s context deadline is only a safety bound; reset_peer fails fast (milliseconds).
//  4. RemoveToxic("reset_peer") — deletes the fault rule only; the proxy itself stays up.
//  5. Call Version() again — driver opens a fresh connection; request must succeed.
//
// What we assert on the failing request:
//   - require.Error: Version() must fail (no silent success).
//   - isConnectionError: failure must be transport-level (TCP reset, broken pipe, etc.),
//     not an ArangoDB HTTP error (401, 503) or an application-level response.
func TestToxiproxy_AbruptTCPConnectionClose(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testAbruptTCPConnectionClose)
}

func testAbruptTCPConnectionClose(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	recoveryTimeout := 1 * time.Minute
	if isK8S() {
		recoveryTimeout = 3 * time.Minute
	}

	client := newToxiproxyClient(t, connFactory)
	waitForSuccessfulVersion(t, client, recoveryTimeout)

	_, err := proxy.AddToxic("reset_peer", "reset_peer", "upstream", 1.0, toxiproxy.Attributes{
		"timeout": int64(0),
	})
	require.NoError(t, err)

	// reset_peer kills the pooled TCP connection; Version() must fail with a transport error.
	require.NotPanics(t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		_, err := client.Version(ctx)
		require.Error(t, err)
		require.True(t, isConnectionError(err),
			"expected transport-level connection failure (e.g. connection reset by peer), got: %v", err)
	})

	// Remove the toxic (not the proxy) so traffic flows normally again.
	require.NoError(t, proxy.RemoveToxic("reset_peer"))
	waitForSuccessfulVersion(t, client, recoveryTimeout)
}

// TestToxiproxy_NetworkDisconnect validates complete network outage via proxy disable/enable.
//
// Flow:
//  1. Connect and call Version() successfully.
//  2. proxy.Disable() — simulates a full network disconnect (no traffic passes, active conns dropped).
//  3. Call Version() while disabled — must fail with a transport error (e.g. connection refused).
//  4. proxy.Enable() — network path restored.
//  5. Call Version() again — driver must work on a fresh connection.
func TestToxiproxy_NetworkDisconnect(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testNetworkDisconnect)
}

func testNetworkDisconnect(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	recoveryTimeout := 1 * time.Minute
	if isK8S() {
		recoveryTimeout = 3 * time.Minute
	}

	client := newToxiproxyClient(t, connFactory)
	waitForSuccessfulVersion(t, client, recoveryTimeout)

	require.NoError(t, proxy.Disable())

	require.NotPanics(t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		_, err := client.Version(ctx)
		require.Error(t, err)
		t.Logf("expected connection error, got: %v", err)
		require.True(t, isConnectionError(err),
			"expected transport-level connection failure while proxy disabled, got: %v", err)
	})

	require.NoError(t, proxy.Enable())
	waitForSuccessfulVersion(t, client, recoveryTimeout)
}

// TestToxiproxy_ConnectionResetByPeer simulates a TCP RST from the remote peer via reset_peer.
//
// Flow:
//  1. Connect and call Version() successfully.
//  2. Add reset_peer on the downstream stream (server → client) with timeout 0 — injects RST.
//  3. Call Version() — must fail with connection reset (HTTP/1) or unexpected EOF (HTTP/2).
//  4. RemoveToxic — RST injection stops.
//  5. Call Version() again — driver recovers on a fresh connection.
func TestToxiproxy_ConnectionResetByPeer(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testConnectionResetByPeer)
}

func testConnectionResetByPeer(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	recoveryTimeout := 1 * time.Minute
	if isK8S() {
		recoveryTimeout = 3 * time.Minute
	}

	client := newToxiproxyClient(t, connFactory)
	waitForSuccessfulVersion(t, client, recoveryTimeout)

	// Downstream reset models the remote peer aborting the connection during the response path.
	_, err := proxy.AddToxic("reset_peer_down", "reset_peer", "downstream", 1.0, toxiproxy.Attributes{
		"timeout": int64(0),
	})
	require.NoError(t, err)

	require.NotPanics(t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		_, err := client.Version(ctx)
		require.Error(t, err)
		t.Logf("expected connection error, got: %v", err)
		require.True(t, isResetOrEOFError(err),
			"expected connection reset or unexpected EOF after RST, got: %v", err)
	})

	require.NoError(t, proxy.RemoveToxic("reset_peer_down"))
	waitForSuccessfulVersion(t, client, recoveryTimeout)
}
