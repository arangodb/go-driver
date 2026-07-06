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
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/connection"
)

const (
	defaultToxiproxyAdminURL  = "http://127.0.0.1:8474"
	defaultToxiproxyProxyName = "arangodb"
)

type toxiproxyEnv struct {
	client    *toxiproxy.Client
	proxyName string
}

// toxiproxyAdminURL returns the Toxiproxy admin API base URL from TEST_TOXIPROXY_ADMIN or the default.
func toxiproxyAdminURL() string {
	if v := os.Getenv("TEST_TOXIPROXY_ADMIN"); v != "" {
		return v
	}
	return defaultToxiproxyAdminURL
}

// toxiproxyProxyName returns the proxy name from TEST_TOXIPROXY_PROXY or the default.
func toxiproxyProxyName() string {
	if v := os.Getenv("TEST_TOXIPROXY_PROXY"); v != "" {
		return v
	}
	return defaultToxiproxyProxyName
}

// requireToxiproxyAvailable skips the test when the Toxiproxy admin API is not reachable.
func requireToxiproxyAvailable(t testing.TB) {
	t.Helper()

	adminURL := toxiproxyAdminURL()
	parsed, err := url.Parse(adminURL)
	require.NoError(t, err)

	hostPort := parsed.Host
	if hostPort == "" {
		hostPort = parsed.Path
	}

	conn, err := net.DialTimeout("tcp", hostPort, 2*time.Second)
	if err != nil {
		t.Skipf("Toxiproxy admin API not reachable at %s: %v", adminURL, err)
	}
	_ = conn.Close()
}

// newToxiproxyEnv connects to the Toxiproxy admin API and clears any existing toxics on the proxy.
func newToxiproxyEnv(t testing.TB) *toxiproxyEnv {
	t.Helper()

	requireToxiproxyAvailable(t)

	client := toxiproxy.NewClient(toxiproxyAdminURL())
	proxyName := toxiproxyProxyName()

	proxy, err := client.Proxy(proxyName)
	require.NoError(t, err, "proxy %q must exist; start Toxiproxy via make run-v2-tests-toxiproxy", proxyName)

	clearProxyToxics(t, proxy)

	return &toxiproxyEnv{
		client:    client,
		proxyName: proxy.Name,
	}
}

// proxy returns the named Toxiproxy proxy handle for toxic configuration.
func (e *toxiproxyEnv) proxy(t testing.TB) *toxiproxy.Proxy {
	t.Helper()

	proxy, err := e.client.Proxy(e.proxyName)
	require.NoError(t, err)
	return proxy
}

// clearProxyToxics removes all toxics from the proxy so each subtest starts from a clean state.
func clearProxyToxics(t testing.TB, proxy *toxiproxy.Proxy) {
	t.Helper()

	toxics, err := proxy.Toxics()
	require.NoError(t, err)

	for _, toxic := range toxics {
		require.NoError(t, proxy.RemoveToxic(toxic.Name))
	}
}

// ensureProxyEnabled re-enables the proxy after a disconnect scenario.
func ensureProxyEnabled(t testing.TB, proxy *toxiproxy.Proxy) {
	t.Helper()
	require.NoError(t, proxy.Enable())
}

// addUpstreamLatency adds a latency toxic on the client→server stream (latency in milliseconds).
func addUpstreamLatency(t testing.TB, proxy *toxiproxy.Proxy, name string, latencyMs int64) {
	t.Helper()

	_, err := proxy.AddToxic(name, "latency", "upstream", 1.0, toxiproxy.Attributes{
		"latency": latencyMs,
	})
	require.NoError(t, err)
}

// addDownstreamLatency adds a latency toxic on the server→client stream (delays the response).
func addDownstreamLatency(t testing.TB, proxy *toxiproxy.Proxy, name string, latencyMs int64) {
	t.Helper()

	_, err := proxy.AddToxic(name, "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": latencyMs,
	})
	require.NoError(t, err)
}

// addUpstreamResetPeer adds reset_peer on the client→server stream with configurable toxicity.
// Lower toxicity simulates intermittent packet loss (only some TCP links are affected).
func addUpstreamResetPeer(t testing.TB, proxy *toxiproxy.Proxy, name string, toxicity float32, timeoutMs int64) {
	t.Helper()

	_, err := proxy.AddToxic(name, "reset_peer", "upstream", toxicity, toxiproxy.Attributes{
		"timeout": timeoutMs,
	})
	require.NoError(t, err)
}

// addUpstreamTimeout adds a timeout toxic on the client→server stream that stops all data flow.
// With timeout 0 and toxicity 1.0 this models a complete packet-loss / network outage.
func addUpstreamTimeout(t testing.TB, proxy *toxiproxy.Proxy, name string, toxicity float32, timeoutMs int64) {
	t.Helper()

	_, err := proxy.AddToxic(name, "timeout", "upstream", toxicity, toxiproxy.Attributes{
		"timeout": timeoutMs,
	})
	require.NoError(t, err)
}

// addUpstreamBandwidth limits upload throughput on the client→server stream (rate in KB/s).
func addUpstreamBandwidth(t testing.TB, proxy *toxiproxy.Proxy, name string, rateKBs int64) {
	t.Helper()

	_, err := proxy.AddToxic(name, "bandwidth", "upstream", 1.0, toxiproxy.Attributes{
		"rate": rateKBs,
	})
	require.NoError(t, err)
}

// addDownstreamBandwidth limits download throughput on the server→client stream (rate in KB/s).
func addDownstreamBandwidth(t testing.TB, proxy *toxiproxy.Proxy, name string, rateKBs int64) {
	t.Helper()

	_, err := proxy.AddToxic(name, "bandwidth", "downstream", 1.0, toxiproxy.Attributes{
		"rate": rateKBs,
	})
	require.NoError(t, err)
}

// requireDriverRoutesThroughToxiproxy fails when cluster endpoint discovery rewrote the
// driver away from the toxiproxy listen URL (injected toxics would not apply).
func requireDriverRoutesThroughToxiproxy(t testing.TB, client interface {
	Connection() connection.Connection
}) {
	t.Helper()

	expected := getEndpointsFromEnv(t)
	actual := client.Connection().GetEndpoint().List()
	require.Len(t, actual, len(expected),
		"driver endpoint count changed after connect (expected %v, got %v)", expected, actual)

	for i, ep := range actual {
		want := connection.FixupEndpointURLScheme(expected[i])
		got := connection.FixupEndpointURLScheme(ep)
		require.Equal(t, want, got,
			"driver must keep routing through toxiproxy %s, not %s; for Kubernetes use: K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy",
			want, got)
	}

	listenPort := os.Getenv("TOXIPROXY_LISTEN_PORT")
	if listenPort == "" {
		listenPort = "17001"
	}
	for _, ep := range actual {
		require.True(t, strings.Contains(ep, ":"+listenPort),
			"toxiproxy tests require proxy listen port %s, got %q", listenPort, ep)
	}
}
