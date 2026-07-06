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

	"github.com/stretchr/testify/require"
)

// TestToxiproxy_ContextTimeout validates that Version() fails with context deadline
// exceeded when the caller timeout is shorter than injected network latency.
func TestToxiproxy_ContextTimeout(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testContextTimeout)
}

// testContextTimeout injects upstream latency and expects the caller context deadline
// to expire before the full round-trip delay elapses.
func testContextTimeout(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	client := newToxiproxyClient(t, connFactory)
	waitForSuccessfulVersion(t, client, 1*time.Minute)

	addUpstreamLatency(t, proxy, "latency_up", toxiproxyContextTimeoutLatencyMs)
	t.Cleanup(func() {
		_ = proxy.RemoveToxic("latency_up")
	})

	require.NotPanics(t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), toxiproxyContextTimeoutDeadline)
		defer cancel()

		start := time.Now()
		_, err := client.Version(ctx)
		elapsed := time.Since(start)

		require.Error(t, err)
		require.True(t, isContextDeadlineExceeded(err),
			"expected context deadline exceeded with 20s latency and 2s timeout, got: %v", err)
		require.Less(t, elapsed, 5*time.Second,
			"expected failure near the 2s context deadline, not after full 20s latency (elapsed %v)", elapsed)
	})
}

// TestToxiproxy_ServerTimeout validates that Version() reports a driver timeout without
// hanging when the server response is delayed beyond the transport response-header deadline.
func TestToxiproxy_ServerTimeout(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testServerTimeout, toxiproxyProtocolConfig{
		http1: connectionToxiproxyHttpServerTimeout,
		skipHTTP2: "HTTP/2 transport has no ResponseHeaderTimeout; server response delay covered on HTTP/1",
	})
}

// testServerTimeout injects downstream response delay and expects the HTTP transport
// response-header deadline to fire without waiting for the full toxic latency.
func testServerTimeout(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	client := newToxiproxyClient(t, connFactory)
	waitForSuccessfulVersion(t, client, 1*time.Minute)

	addDownstreamLatency(t, proxy, "latency_down", toxiproxyServerTimeoutResponseLatencyMs)
	t.Cleanup(func() {
		_ = proxy.RemoveToxic("latency_down")
	})

	require.NotPanics(t, func() {
		start := time.Now()
		_, err := client.Version(context.Background())
		elapsed := time.Since(start)

		require.Error(t, err)
		require.True(t, isDriverTimeoutError(err),
			"expected driver timeout on delayed server response, got: %v", err)
		require.Less(t, elapsed, toxiproxyServerTimeoutMaxWait,
			"expected timeout without hanging for 20s response delay (elapsed %v)", elapsed)
	})
}
