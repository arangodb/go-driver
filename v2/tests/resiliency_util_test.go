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
// Kubernetes resiliency tests use a longer connection timeout after coordinator chaos.
func newResiliencyClient(t testing.TB, conn connection.Connection) arangodb.Client {
	client := arangodb.NewClient(conn)
	if isK8S() {
		return waitForConnectionTimeout(t, client, 3*time.Minute)
	}
	return waitForConnectionTimeout(t, client, 1*time.Minute)
}

// prepareResiliencyClient waits for coordinator recovery and cluster stability before a subtest.
// On Kubernetes this verifies both read (Version) and write (CreateDatabase) paths through ingress.
func prepareResiliencyClient(t testing.TB, connFactory resiliencyConnectionFactory) arangodb.Client {
	if isK8S() && expectedCoordinatorCount() >= minCoordinatorResiliencyPods {
		ensureCoordinatorsRecovered(t, nil)
	}

	client := newResiliencyClient(t, connFactory(t))
	if isK8S() {
		waitForClusterStable(t, client, 3*time.Minute)
		waitForClusterWritable(t, client, 3*time.Minute)
		return client
	}

	waitForSuccessfulVersion(t, client, 2*time.Minute)
	return client
}

// runResiliencyWithHTTPProtocols runs the given test body for both HTTP/1 and HTTP/2 connections.
func runResiliencyWithHTTPProtocols(t *testing.T, run func(t *testing.T, connFactory resiliencyConnectionFactory)) {
	probe := prepareResiliencyClient(t, connectionJsonHttp)
	version, err := probe.Version(context.Background())
	require.NoError(t, err)

	t.Run("HTTP/1", func(t *testing.T) {
		run(t, connectionJsonHttp)
	})

	t.Run("HTTP/2", func(t *testing.T) {
		if version.Version.CompareTo("3.7.1") < 1 {
			t.Skip("HTTP/2 requires ArangoDB 3.7.1 or newer")
		}
		if isK8S() && expectedCoordinatorCount() >= minCoordinatorResiliencyPods {
			prepareResiliencyClient(t, connectionJsonHttp2)
		}
		run(t, connectionJsonHttp2)
	})
}

// waitForClusterStable requires consecutive successful Version calls before proceeding.
func waitForClusterStable(t testing.TB, client arangodb.Client, timeout time.Duration) {
	t.Helper()

	const requiredStreak = 5
	streak := 0

	NewTimeout(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.Version(ctx)
		if err != nil {
			streak = 0
			return nil
		}

		streak++
		if streak >= requiredStreak {
			return Interrupt{}
		}
		return nil
	}).TimeoutT(t, timeout, 250*time.Millisecond)
}

// waitForClusterWritable retries a create/delete database probe until ingress serves writes.
func waitForClusterWritable(t testing.TB, client arangodb.Client, timeout time.Duration) {
	t.Helper()

	NewTimeout(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		name := GenerateUUID("resiliency-probe-db")
		db, err := client.CreateDatabase(ctx, name, nil)
		if err != nil {
			return nil
		}

		removeCtx, removeCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer removeCancel()
		if err := db.Remove(removeCtx); err != nil {
			return nil
		}

		return Interrupt{}
	}).TimeoutT(t, timeout, 500*time.Millisecond)
}
