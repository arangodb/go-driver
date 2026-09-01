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

// TestToxiproxy_HighLatency validates that Version() succeeds through added latency
// when the request context timeout exceeds the injected delay.
func TestToxiproxy_HighLatency(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testHighLatency)
}

func testHighLatency(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	client := newToxiproxyClient(t, connFactory)

	var baseline time.Duration
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		baseline = measureVersionCall(t, client, ctx)
	}()

	addUpstreamLatency(t, proxy, "latency_up", toxiproxyHighLatencyMs)
	t.Cleanup(func() {
		_ = proxy.RemoveToxic("latency_up")
	})

	var slow time.Duration
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		slow = measureVersionCall(t, client, ctx)
	}()

	require.Greater(t, slow, baseline+time.Second,
		"expected roughly 2s injected latency to be visible (baseline %v, with latency %v)", baseline, slow)
}

// TestToxiproxy_ExtremeLatency validates that Version() fails when injected latency
// exceeds the request context deadline.
func TestToxiproxy_ExtremeLatency(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testExtremeLatency)
}

func testExtremeLatency(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	client := newToxiproxyClient(t, connFactory)
	waitForSuccessfulVersion(t, client, 1*time.Minute)

	addUpstreamLatency(t, proxy, "latency_up", toxiproxyExtremeLatencyMs)
	t.Cleanup(func() {
		_ = proxy.RemoveToxic("latency_up")
	})

	require.NotPanics(t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.Version(ctx)
		require.Error(t, err)
		t.Logf("error message: %s", err.Error())
		require.True(t, isContextDeadlineExceeded(err),
			"expected context deadline exceeded with 30s latency and 10s timeout, got: %v", err)
	})

	require.NoError(t, proxy.RemoveToxic("latency_up"))
	waitForSuccessfulVersion(t, client, 1*time.Minute)
}

// TestToxiproxy_LatencyRemoved validates that request duration returns to normal
// after a latency toxic is removed.
func TestToxiproxy_LatencyRemoved(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testLatencyRemoved)
}

func testLatencyRemoved(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	client := newToxiproxyClient(t, connFactory)

	var baseline time.Duration
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		baseline = measureVersionCall(t, client, ctx)
	}()

	addUpstreamLatency(t, proxy, "latency_up", toxiproxyHighLatencyMs)

	var withLatency time.Duration
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		withLatency = measureVersionCall(t, client, ctx)
	}()
	require.Greater(t, withLatency, baseline+time.Second)

	require.NoError(t, proxy.RemoveToxic("latency_up"))

	var afterRemoval time.Duration
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		afterRemoval = measureVersionCall(t, client, ctx)
	}()
	t.Logf("afterRemoval: %v", afterRemoval)
	t.Logf("withLatency*2/3: %v", withLatency*2/3)
	require.Less(t, afterRemoval, withLatency*2/3,
		"expected faster requests after latency removal (with latency %v, after %v)", withLatency, afterRemoval)
}
