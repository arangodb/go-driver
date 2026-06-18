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
// Resiliency client helpers — how the go-driver reaches ArangoDB:
//
//	Docker path (arangodb-starter):
//	  go-driver → each coordinator URL directly (round-robin after discovery)
//	  e.g. http://127.0.0.1:7001, :7002, :7003
//
//	K8s path (kube-arangodb + ingress):
//	  go-driver → single ingress URL (TEST_ENDPOINTS_OVERRIDE)
//	  e.g. https://arangodb.local
//	    → nginx ingress / LoadBalancer (ClientIP session affinity)
//	    → one of N coordinator pods (CRDN-xxx)
//
// Load-balancer test uses GetServerStatus (GET /_admin/status) to read which
// coordinator (serverInfo.serverId) answered each HTTP call.

package tests

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

const resiliencyRetryCount = 15

var errMissingCoordinatorServerID = &missingCoordinatorServerIDError{}

type missingCoordinatorServerIDError struct{}

func (e *missingCoordinatorServerIDError) Error() string {
	return "server status response missing coordinator serverId"
}

// newResiliencyClient creates a client connected to all coordinator endpoints with retry support.
func newResiliencyClient(t testing.TB, conn connection.Connection) arangodb.Client {
	t.Helper()

	conn = connection.RetryOn503(conn, resiliencyRetryCount)
	client := arangodb.NewClient(conn)
	return waitForResiliencyConnection(t, client)
}

func waitForResiliencyConnection(t testing.TB, client arangodb.Client) arangodb.Client {
	t.Helper()

	if isK8S() {
		return waitForK8sResiliencyConnection(t, client)
	}

	NewTimeout(func() error {
		return withContext(time.Second, func(ctx context.Context) error {
			eps, err := getCoordinatorEndpoints(ctx, client)
			if err != nil {
				log.Warn().Err(err).Msg("Unable to get coordinator endpoints")
				return nil
			}

			err = client.Connection().SetEndpoint(connection.NewRoundRobinEndpoints(eps))
			if err != nil {
				log.Warn().Err(err).Msg("Unable to set coordinator endpoints")
				return nil
			}

			resp, err := client.Get(ctx, nil, "_admin", "server", "availability")
			if err != nil {
				log.Warn().Err(err).Msg("Unable to get server availability")
				return nil
			}
			if resp.Code() != http.StatusOK {
				return nil
			}

			t.Logf("Resiliency client endpoints: %v", client.Connection().GetEndpoint().List())
			return Interrupt{}
		})
	}).TimeoutT(t, time.Minute, time.Second)

	return client
}

// waitForK8sResiliencyConnection waits until GET /_admin/server/availability
// succeeds through the ingress URL (single endpoint from TEST_ENDPOINTS_OVERRIDE).
func waitForK8sResiliencyConnection(t testing.TB, client arangodb.Client) arangodb.Client {
	t.Helper()

	// Traffic goes through the ingress endpoint (TEST_ENDPOINTS / TEST_ENDPOINTS_OVERRIDE).
	NewTimeout(func() error {
		return withContext(time.Second, func(ctx context.Context) error {
			resp, err := client.Get(ctx, nil, "_admin", "server", "availability")
			if err != nil {
				log.Warn().Err(err).Msg("Unable to get server availability through ingress")
				return nil
			}
			if resp.Code() != http.StatusOK {
				return nil
			}

			if isK8SInCluster() {
				t.Logf("Resiliency client endpoints (k8s in-cluster service): %v", client.Connection().GetEndpoint().List())
				return Interrupt{}
			}
			t.Logf("Resiliency client endpoints (k8s ingress): %v", client.Connection().GetEndpoint().List())
			return Interrupt{}
		})
	}).TimeoutT(t, time.Minute, time.Second)

	return client
}

func getCoordinatorEndpoints(ctx context.Context, client arangodb.Client) ([]string, error) {
	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("cluster health: %w", err)
	}

	eps := make([]string, 0)
	for _, server := range health.Health {
		if server.Role != arangodb.ServerRoleCoordinator {
			continue
		}
		eps = append(eps, normalizeLocalhostEndpoint(connection.FixupEndpointURLScheme(server.Endpoint)))
	}

	if len(eps) == 0 {
		return nil, fmt.Errorf("no coordinator endpoints found in cluster health")
	}

	return eps, nil
}

// normalizeLocalhostEndpoint rewrites localhost to 127.0.0.1 to avoid IPv6 resolution issues in CI.
func normalizeLocalhostEndpoint(ep string) string {
	u, err := url.Parse(ep)
	if err != nil || u.Hostname() != "localhost" {
		return ep
	}
	u.Host = net.JoinHostPort("127.0.0.1", u.Port())
	return u.String()
}

// retryReadDocument retries a document read while coordinators may be unavailable.
func retryReadDocument(t testing.TB, col arangodb.Collection, key string, doc interface{}, timeout time.Duration) {
	t.Helper()

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		err := NewTimeout(func() error {
			_, readErr := col.ReadDocument(ctx, key, doc)
			if readErr == nil {
				return Interrupt{}
			}

			if ok, arangoErr := shared.IsArangoError(readErr); ok {
				switch arangoErr.Code {
				case http.StatusServiceUnavailable, http.StatusRequestTimeout, http.StatusInternalServerError:
					return nil
				}
			}

			if isRetryableConnectionError(readErr) {
				return nil
			}

			return readErr
		}).Timeout(timeout, 500*time.Millisecond)

		require.NoError(t, err, "read document %s", key)
	})
}

// connectionJsonHttpFresh builds an HTTP/1 client that closes the connection
// after each request (DisableKeepAlives). Used to force a new TCP connection
// to ingress for each of the 3 load-balancer probe requests.
func connectionJsonHttpFresh(t testing.TB) connection.Connection {
	endpoints := connection.NewRoundRobinEndpoints(getEndpointsFromEnv(t))
	h := connection.HttpConfiguration{
		Endpoint:    endpoints,
		ContentType: connection.ApplicationJSON,
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives:     true,
			MaxIdleConns:          0,
			IdleConnTimeout:       0,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).DialContext,
		},
	}

	c := connection.NewHttpConnection(h)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
		c = createAuthenticationFromEnv(t, c)
	})
	return c
}

func isRetryableConnectionError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	return stringsContainsAny(msg,
		"connection refused",
		"connection reset",
		"EOF",
		"broken pipe",
		"i/o timeout",
		"no such host",
		"use of closed network connection",
	)
}

func stringsContainsAny(s string, parts ...string) bool {
	lower := strings.ToLower(s)
	for _, part := range parts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}
