//go:build resiliency

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

// resiliencyConnectionFactory builds a driver connection for a resiliency scenario.
type resiliencyConnectionFactory func(testing.TB) connection.Connection

// requireResiliencyK8sIngressMode skips the test unless it runs in Kubernetes cluster mode through ingress.
func requireResiliencyK8sIngressMode(t testing.TB) {
	require.True(t, isK8S(), "resiliency ingress tests require TEST_MODE_K8S=k8s")
	requireClusterMode(t)
}

// newResiliencyClient creates a driver client and waits until the cluster is reachable.
func newResiliencyClient(t testing.TB, conn connection.Connection) arangodb.Client {
	return newClient(t, conn)
}

// runResiliencyWithHTTPProtocols runs the given test body for both HTTP/1 and HTTP/2 connections.
func runResiliencyWithHTTPProtocols(t *testing.T, run func(t *testing.T, connFactory resiliencyConnectionFactory)) {
	probe := newResiliencyClient(t, connectionJsonHttp(t))
	version, err := probe.Version(context.Background())
	require.NoError(t, err)

	t.Run("HTTP/1", func(t *testing.T) {
		run(t, connectionJsonHttp)
	})

	t.Run("HTTP/2", func(t *testing.T) {
		if version.Version.CompareTo("3.7.1") < 1 {
			t.Skip("HTTP/2 requires ArangoDB 3.7.1 or newer")
		}
		run(t, connectionJsonHttp2)
	})
}

// waitForSuccessfulVersion retries client.Version until it succeeds or the timeout expires.
func waitForSuccessfulVersion(t testing.TB, client arangodb.Client, timeout time.Duration) {
	t.Helper()

	NewTimeout(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.Version(ctx)
		if err == nil {
			return Interrupt{}
		}
		return nil
	}).TimeoutT(t, timeout, 500*time.Millisecond)
}
