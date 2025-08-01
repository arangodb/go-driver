//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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
	"crypto/tls"
	"math/rand"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

type Wrapper func(t *testing.T, client arangodb.Client)
type WrapperConnection func(t *testing.T, conn connection.Connection)
type WrapperConnectionFactory func(t *testing.T, connFactory ConnectionFactory)
type ConnectionFactory func(t *testing.T) connection.Connection

type WrapperB func(t *testing.B, client arangodb.Client)

// WrapOptions describes testing options for a wrapper.
type WrapOptions struct {
	// Parallel describes if internal tests should be launched in parallel.
	// If it is nil then by default, it is true.
	Parallel *bool

	// Async describes if the client should be created with async mode (controlled within the context).
	Async *bool
}

func WrapConnectionFactory(t *testing.T, w WrapperConnectionFactory, wo ...WrapOptions) {
	c := newClient(t, connectionJsonHttp(t))

	version, err := c.Version(context.Background())
	require.NoError(t, err)

	if getTestMode() == string(testModeResilientSingle) {
		// AF mode is not anymore supported
		skipFromVersion(c, context.Background(), "3.12", t)
	}

	parallel := true
	async := false

	if len(wo) > 0 {
		if wo[0].Parallel != nil {
			parallel = *wo[0].Parallel
		}
		if wo[0].Async != nil {
			async = *wo[0].Async
		}
	}

	if parallel {
		t.Parallel()
	}

	t.Run("HTTP JSON", func(t *testing.T) {
		if parallel {
			t.Parallel()
		}

		w(t, func(t *testing.T) connection.Connection {
			conn := connectionJsonHttp(t)
			if async {
				conn = connection.NewConnectionAsyncWrapper(conn)
			}

			waitForConnection(t, arangodb.NewClient(conn))
			applyCompression(conn)
			return conn
		})
	})

	t.Run("HTTP2 JSON", func(t *testing.T) {
		if version.Version.CompareTo("3.7.1") < 1 {
			t.Skipf("Not supported")
		}
		if parallel {
			t.Parallel()
		}

		w(t, func(t *testing.T) connection.Connection {
			conn := connectionJsonHttp2(t)
			if async {
				conn = connection.NewConnectionAsyncWrapper(conn)
			}

			waitForConnection(t, arangodb.NewClient(conn))
			applyCompression(conn)
			return conn
		})
	})
}

func applyCompression(conn connection.Connection) {
	if os.Getenv("ENABLE_DATABASE_EXTRA_FEATURES") == "true" {
		cmp := conn.GetConfiguration()
		cmp.Compression = &connection.CompressionConfig{
			CompressionType:            connection.RequestCompressionTypeDeflate,
			RequestCompressionLevel:    9,
			ResponseCompressionEnabled: true,
			RequestCompressionEnabled:  true,
		}
		conn.SetConfiguration(cmp)
	}
}

func WrapConnection(t *testing.T, w WrapperConnection, wo ...WrapOptions) {
	WrapConnectionFactory(t, func(t *testing.T, connFactory ConnectionFactory) {
		w(t, connFactory(t))
	}, wo...)
}

func Wrap(t *testing.T, w Wrapper, wo ...WrapOptions) {
	WrapConnection(t, func(t *testing.T, conn connection.Connection) {
		w(t, arangodb.NewClient(conn))
	}, wo...)
}

func WrapB(t *testing.B, w WrapperB) {
	// HTTP

	c := newClient(t, connectionJsonHttp(t))

	version, err := c.Version(context.Background())
	require.NoError(t, err)

	t.Run("HTTP JSON", func(t *testing.B) {
		w(t, newClient(t, connectionJsonHttp(t)))
	})

	t.Run("HTTP2 JSON", func(t *testing.B) {
		if version.Version.CompareTo("3.7.1") < 1 {
			t.Skipf("Not supported")
		}
		w(t, newClient(t, connectionJsonHttp2(t)))
	})
}

func newClient(t testing.TB, connection connection.Connection) arangodb.Client {
	return waitForConnection(t, arangodb.NewClient(connection))
}

type clusterEndpointsResponse struct {
	Endpoints []clusterEndpoint `json:"endpoints,omitempty"`
}

type clusterEndpoint struct {
	Endpoint string `json:"endpoint,omitempty"`
}

func waitForConnection(t testing.TB, client arangodb.Client) arangodb.Client {
	// For Active Failover, we need to track the leader endpoint
	var nextEndpoint = -1

	NewTimeout(func() error {
		return withContext(time.Second, func(ctx context.Context) error {
			if getTestMode() != string(testModeSingle) {
				cer := clusterEndpointsResponse{}
				resp, err := client.Get(ctx, &cer, "_api", "cluster", "endpoints")
				if err != nil {
					log.Warn().Err(err).Msgf("Unable to get cluster endpoints")
					return nil
				}

				if resp.Code() != http.StatusOK {
					return nil
				}

				if len(cer.Endpoints) == 0 {
					t.Fatal("No endpoints found")
				}

				nextEndpoint++
				if nextEndpoint >= len(cer.Endpoints) {
					nextEndpoint = 0
				}

				// pick the first one endpoint which is always the leader in AF mode
				// also for Cluster mode we only need one endpoint to avoid the problem with the data propagation in tests
				endpoint := connection.NewRoundRobinEndpoints([]string{connection.FixupEndpointURLScheme(cer.Endpoints[nextEndpoint].Endpoint)})
				err = client.Connection().SetEndpoint(endpoint)
				if err != nil {
					log.Warn().Err(err).Msgf("Unable to set endpoints")
					return nil
				}
			}
			current := time.Now()
			resp, err := client.Get(ctx, nil, "_admin", "server", "availability")
			if err != nil {
				log.Warn().Err(err).Msgf("Unable to get SUT health, deadline: %s", time.Since(current))
				return nil
			}

			if resp.Code() != http.StatusOK {
				return nil
			}

			t.Logf("Found endpoints: %v", client.Connection().GetEndpoint().List())

			return Interrupt{}
		})
	}).TimeoutT(t, 1*time.Minute, 1*time.Second)

	return client
}

// getRandomEndpointsManager returns random endpoints manager
func getRandomEndpointsManager(t testing.TB) connection.Endpoint {
	eps := getEndpointsFromEnv(t)
	rand.New(rand.NewSource(time.Now().UnixNano()))
	if rand.Intn(2) == 1 {
		t.Log("Using MaglevHashEndpoints")
		ep, err := connection.NewMaglevHashEndpoints(eps, connection.RequestDBNameValueExtractor)
		require.NoError(t, err)
		return ep
	}
	t.Log("Using RoundRobinEndpoints")
	return connection.NewRoundRobinEndpoints(eps)
}

func connectionJsonHttp(t testing.TB) connection.Connection {
	h := connection.HttpConfiguration{
		Endpoint:    getRandomEndpointsManager(t),
		ContentType: connection.ApplicationJSON,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 90 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	c := connection.NewHttpConnection(h)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
		c = createAuthenticationFromEnv(t, c)
	})
	return c
}

func connectionJsonHttp2(t testing.TB) connection.Connection {
	endpoints := getRandomEndpointsManager(t)
	h := connection.Http2Configuration{
		Endpoint:    endpoints,
		ContentType: connection.ApplicationJSON,
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			AllowHTTP:       true,

			DialTLSContext: connection.NewHTTP2DialForEndpoint(endpoints),
		},
	}

	c := connection.NewHttp2Connection(h)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
		c = createAuthenticationFromEnv(t, c)
	})
	return c
}

func withContext(timeout time.Duration, f func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return f(ctx)
}

func withContextT(t testing.TB, timeout time.Duration, f func(ctx context.Context, t testing.TB)) {
	require.NoError(t, withContext(timeout, func(ctx context.Context) error {
		f(ctx, t)
		return nil
	}))
}
