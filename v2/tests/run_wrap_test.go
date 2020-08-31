//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package tests

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

type Wrapper func(t *testing.T, client arangodb.Client)
type WrapperB func(t *testing.B, client arangodb.Client)

func Wrap(t *testing.T, w Wrapper) {
	c := newClient(t, connectionJsonHttp(t))

	version, err := c.Version(context.Background())
	require.NoError(t, err)
	// HTTP

	t.Parallel()

	t.Run("HTTP JSON", func(t *testing.T) {
		t.Parallel()
		w(t, newClient(t, connectionJsonHttp(t)))
	})

	t.Run("HTTP VPACK", func(t *testing.T) {
		t.Parallel()
		w(t, newClient(t, connectionVPACKHttp(t)))
	})

	t.Run("HTTP2 JSON", func(t *testing.T) {
		if version.Version.CompareTo("3.7.1") < 1 {
			t.Skipf("Not supported")
		}
		t.Parallel()
		w(t, newClient(t, connectionJsonHttp2(t)))
	})

	t.Run("HTTP2 VPACK", func(t *testing.T) {
		if version.Version.CompareTo("3.7.1") < 1 {
			t.Skipf("Not supported")
		}
		t.Parallel()
		w(t, newClient(t, connectionVPACKHttp2(t)))
	})
}

func WrapB(t *testing.B, w WrapperB) {
	// HTTP

	c := newClient(t, connectionJsonHttp(t))

	version, err := c.Version(context.Background())
	require.NoError(t, err)

	t.Run("HTTP JSON", func(t *testing.B) {
		w(t, newClient(t, connectionJsonHttp(t)))
	})

	t.Run("HTTP VPACK", func(t *testing.B) {
		w(t, newClient(t, connectionVPACKHttp(t)))
	})

	t.Run("HTTP2 VPACK", func(t *testing.B) {
		if version.Version.CompareTo("3.7.1") < 1 {
			t.Skipf("Not supported")
		}
		w(t, newClient(t, connectionVPACKHttp2(t)))
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

func waitForConnection(t testing.TB, client arangodb.Client) arangodb.Client {
	NewTimeout(func() error {
		return withContext(time.Second, func(ctx context.Context) error {

			resp, err := client.Get(ctx, nil, "_admin", "server", "availability")
			if err != nil {
				log.Warn().Err(err).Msgf("Unable to get cluster health")
				return nil
			}

			if resp.Code() != http.StatusOK {
				return nil
			}

			return Interrupt{}
		})
	}).TimeoutT(t, time.Minute, 2*time.Second)

	return client
}

func connectionJsonHttp(t testing.TB) connection.Connection {
	h := connection.HttpConfiguration{
		Endpoint:    connection.NewEndpoints(getEndpointsFromEnv(t)...),
		ContentType: connection.ApplicationJSON,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	c := connection.NewHttpConnection(h)

	withContext(2*time.Minute, func(ctx context.Context) error {
		c = createAuthenticationFromEnv(t, ctx, c)
		return nil
	})
	return c
}

func connectionVPACKHttp(t testing.TB) connection.Connection {
	h := connection.HttpConfiguration{
		Endpoint:    connection.NewEndpoints(getEndpointsFromEnv(t)...),
		ContentType: connection.ApplicationVPack,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	c := connection.NewHttpConnection(h)

	withContext(2*time.Minute, func(ctx context.Context) error {
		c = createAuthenticationFromEnv(t, ctx, c)
		return nil
	})
	return c
}

func connectionJsonHttp2(t testing.TB) connection.Connection {
	h := connection.Http2Configuration{
		Endpoint:    connection.NewEndpoints(getEndpointsFromEnv(t)...),
		ContentType: connection.ApplicationJSON,
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			AllowHTTP:       true,

			DialTLS: connection.NewHTTP2DialForEndpoint(connection.NewEndpoints(getEndpointsFromEnv(t)...)),
		},
	}

	c := connection.NewHttp2Connection(h)

	withContext(2*time.Minute, func(ctx context.Context) error {
		c = createAuthenticationFromEnv(t, ctx, c)
		return nil
	})
	return c
}

func connectionVPACKHttp2(t testing.TB) connection.Connection {
	h := connection.Http2Configuration{
		Endpoint:    connection.NewEndpoints(getEndpointsFromEnv(t)...),
		ContentType: connection.ApplicationVPack,
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			AllowHTTP:       true,

			DialTLS: connection.NewHTTP2DialForEndpoint(connection.NewEndpoints(getEndpointsFromEnv(t)...)),
		},
	}

	c := connection.NewHttp2Connection(h)

	withContext(2*time.Minute, func(ctx context.Context) error {
		c = createAuthenticationFromEnv(t, ctx, c)
		return nil
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
