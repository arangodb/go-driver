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
	"testing"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/stretchr/testify/require"
)

const (
	defaultToxiproxyAdminURL  = "http://127.0.0.1:8474"
	defaultToxiproxyProxyName = "arangodb"
)

type toxiproxyEnv struct {
	client    *toxiproxy.Client
	proxyName string
}

func toxiproxyAdminURL() string {
	if v := os.Getenv("TEST_TOXIPROXY_ADMIN"); v != "" {
		return v
	}
	return defaultToxiproxyAdminURL
}

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

func (e *toxiproxyEnv) proxy(t testing.TB) *toxiproxy.Proxy {
	t.Helper()

	proxy, err := e.client.Proxy(e.proxyName)
	require.NoError(t, err)
	return proxy
}

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
