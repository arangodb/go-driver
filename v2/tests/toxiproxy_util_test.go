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

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

// toxiproxyConnectionFactory builds a driver connection routed through Toxiproxy.
type toxiproxyConnectionFactory func(testing.TB) connection.Connection

// toxiproxyProtocolConfig overrides default HTTP/1 and HTTP/2 connection factories.
// A zero value uses connectionJsonHttp and connectionJsonHttp2.
type toxiproxyProtocolConfig struct {
	http1     toxiproxyConnectionFactory
	http2     toxiproxyConnectionFactory
	skipHTTP2 string
}

// http1Factory returns the configured HTTP/1 factory or the default JSON-over-HTTP/1 connection.
func (c toxiproxyProtocolConfig) http1Factory() toxiproxyConnectionFactory {
	if c.http1 != nil {
		return c.http1
	}
	return connectionJsonHttp
}

// http2Factory returns the configured HTTP/2 factory or the default JSON-over-HTTP/2 connection.
func (c toxiproxyProtocolConfig) http2Factory() toxiproxyConnectionFactory {
	if c.http2 != nil {
		return c.http2
	}
	return connectionJsonHttp2
}

// newToxiproxyClient creates a driver client and waits until ArangoDB is reachable through the proxy.
func newToxiproxyClient(t testing.TB, connFactory toxiproxyConnectionFactory) arangodb.Client {
	timeout := 1 * time.Minute
	if isK8S() {
		timeout = 3 * time.Minute
	}
	client := waitForConnectionTimeout(t, arangodb.NewClient(connFactory(t)), timeout)
	requireDriverRoutesThroughToxiproxy(t, client)
	return client
}

// runToxiproxyWithHTTPProtocols runs the given test body for HTTP/1 and HTTP/2.
// Pass an optional toxiproxyProtocolConfig to override connection factories or skip HTTP/2.
func runToxiproxyWithHTTPProtocols(t *testing.T, run func(t *testing.T, connFactory toxiproxyConnectionFactory), cfg ...toxiproxyProtocolConfig) {
	requireToxiproxyAvailable(t)

	var protocols toxiproxyProtocolConfig
	if len(cfg) > 0 {
		protocols = cfg[0]
	}

	probe := newToxiproxyClient(t, connectionJsonHttp)
	version, err := probe.Version(context.Background())
	require.NoError(t, err)

	t.Run("HTTP/1", func(t *testing.T) {
		run(t, protocols.http1Factory())
	})

	t.Run("HTTP/2", func(t *testing.T) {
		if protocols.skipHTTP2 != "" {
			t.Skip(protocols.skipHTTP2)
		}
		if version.Version.CompareTo("3.7.1") < 1 {
			t.Skip("HTTP/2 requires ArangoDB 3.7.1 or newer")
		}
		run(t, protocols.http2Factory())
	})
}

// toxiproxyBandwidthOperationTimeout returns the context budget for throttled insert/query workloads.
func toxiproxyBandwidthOperationTimeout() time.Duration {
	if isK8S() {
		return 10 * time.Minute
	}
	return 5 * time.Minute
}

const (
	toxiproxyHighLatencyMs                    = int64(2000)
	toxiproxyExtremeLatencyMs                 = int64(30000)
	toxiproxyContextTimeoutLatencyMs          = int64(20000)
	toxiproxyContextTimeoutDeadline           = 2 * time.Second
	toxiproxyServerTimeoutResponseLatencyMs   = int64(20000)
	toxiproxyServerTimeoutDeadline            = 2 * time.Second
	toxiproxyServerTimeoutMaxWait             = 5 * time.Second
	toxiproxyPartialPacketLossToxicity        = float32(0.3)
	toxiproxyPartialPacketLossAttempts        = 20
	toxiproxyFullPacketLossCallTimeout        = 10 * time.Second
	toxiproxyBandwidthLimitKBs                = int64(20)
	toxiproxyBandwidthUploadDocCount          = 30
	toxiproxyBandwidthUploadPayloadBytes      = 4000
	toxiproxyBandwidthDownloadDocCount        = 80
	toxiproxyBandwidthDownloadPayloadBytes    = 4000
	toxiproxyBandwidthMinSlowdownFactor       = 2.0
	toxiproxyStreamingDocCount                = 100
	toxiproxyStreamingDocsBeforeDisconnect    = 5
	toxiproxyStreamingSlowQueryDocCount       = 200
	toxiproxyStreamingSlowQueryBurnIterations = 80
	toxiproxyStreamingInterruptTimeout        = 30 * time.Second
	toxiproxyStreamingCursorOpenTimeout       = 2 * time.Minute
)

// connectionToxiproxyHttpServerTimeout builds an HTTP/1 connection with a short response-header
// timeout so delayed server responses surface as a driver timeout instead of hanging.
func connectionToxiproxyHttpServerTimeout(t testing.TB) connection.Connection {
	transport := testHTTPTransport()
	transport.ResponseHeaderTimeout = toxiproxyServerTimeoutDeadline

	h := connection.HttpConfiguration{
		Endpoint:    getRandomEndpointsManager(t),
		ContentType: connection.ApplicationJSON,
		Transport:   wrapTransportWithIngressHost(transport),
	}

	c := connection.NewHttpConnection(h)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
		c = createAuthenticationFromEnv(t, c)
	})
	return c
}

// connectionToxiproxyHttpNoKeepAlive builds HTTP/1 without keep-alive so each request opens
// a new TCP connection (needed for per-link packet-loss toxics to affect individual calls).
func connectionToxiproxyHttpNoKeepAlive(t testing.TB) connection.Connection {
	transport := testHTTPTransport()
	transport.DisableKeepAlives = true

	h := connection.HttpConfiguration{
		Endpoint:    getRandomEndpointsManager(t),
		ContentType: connection.ApplicationJSON,
		Transport:   wrapTransportWithIngressHost(transport),
	}

	c := connection.NewHttpConnection(h)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
		c = createAuthenticationFromEnv(t, c)
	})
	return c
}

// toxiproxyRecoveryTimeout returns the post-fault recovery wait budget for toxiproxy tests.
func toxiproxyRecoveryTimeout() time.Duration {
	if isK8S() {
		return 3 * time.Minute
	}
	return 1 * time.Minute
}

// measureVersionCall times a successful client.Version call.
func measureVersionCall(t testing.TB, client arangodb.Client, ctx context.Context) time.Duration {
	t.Helper()

	start := time.Now()
	_, err := client.Version(ctx)
	require.NoError(t, err)
	return time.Since(start)
}
