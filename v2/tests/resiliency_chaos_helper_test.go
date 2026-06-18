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
// Resiliency chaos helpers (docker + k8s). File suffix _test.go is required so
// gopls loads this file together with util_test.go, utils_retry_test.go, etc.
//
// This file does NOT send HTTP requests to ArangoDB. It only:
//   - kills coordinators (docker or kubectl)
//   - waits for cluster health via client.Health() (cluster-wide API, any coordinator)

package tests

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

const resiliencyClusterRecoveryTimeout = 3 * time.Minute

type chaosBackend interface {
	killRandomCoordinator(ctx context.Context, client arangodb.Client) (CoordinatorTarget, error)
	waitForInfrastructureRecovery(timeout time.Duration)
}

// ChaosController injects infrastructure faults during resiliency tests.
// It uses docker (arangodb-starter) or kubectl (kube-arangodb on Kubernetes).
type ChaosController struct {
	t       testing.TB
	backend chaosBackend
}

// CoordinatorTarget identifies a coordinator instance targeted by chaos injection.
type CoordinatorTarget struct {
	Endpoint     string
	ResourceName string // docker container name or Kubernetes pod name
}

// NewChaosController creates a controller for the active test environment.
func NewChaosController(t testing.TB) *ChaosController {
	t.Helper()

	if isK8S() {
		return &ChaosController{t: t, backend: newK8sChaos(t)}
	}
	return &ChaosController{t: t, backend: newDockerChaos(t)}
}

// KillRandomCoordinator kills one coordinator. The platform is expected to restart it.
func (c *ChaosController) KillRandomCoordinator(ctx context.Context, client arangodb.Client) (CoordinatorTarget, error) {
	return c.backend.killRandomCoordinator(ctx, client)
}

// WaitForClusterRecovery waits until the cluster is healthy again after chaos injection.
func (c *ChaosController) WaitForClusterRecovery(client arangodb.Client, timeout time.Duration) {
	tt, ok := c.t.(*testing.T)
	if !ok {
		c.t.Fatal("WaitForClusterRecovery requires *testing.T")
	}

	if isK8S() {
		WaitForHealthyCluster(tt, client, timeout, false)
		c.backend.waitForInfrastructureRecovery(timeout)
		c.t.Logf("Cluster recovery complete")
		return
	}

	NewTimeout(func() error {
		return withContext(3*time.Second, func(ctx context.Context) error {
			health, err := client.Health(ctx)
			if err != nil {
				c.t.Logf("Waiting for cluster recovery: health request failed: %v", err)
				return nil
			}

			for id, server := range health.Health {
				if server.Status != arangodb.ServerStatusGood {
					c.t.Logf("Waiting for cluster recovery: server %s status=%s", id, server.Status)
					return nil
				}

				if server.Role != arangodb.ServerRoleCoordinator {
					continue
				}

				ep := normalizeLocalhostEndpoint(connection.FixupEndpointURLScheme(server.Endpoint))
				if err := client.CheckAvailability(ctx, ep); err != nil {
					c.t.Logf("Waiting for cluster recovery: coordinator %s not available yet: %v", ep, err)
					return nil
				}
			}

			return Interrupt{}
		})
	}).TimeoutT(tt, timeout, 500*time.Millisecond)

	c.t.Logf("Cluster recovery complete")
}

func endpointPort(endpoint string) (int, error) {
	fixed := connection.FixupEndpointURLScheme(endpoint)
	u, err := url.Parse(fixed)
	if err != nil {
		return 0, fmt.Errorf("parse endpoint: %w", err)
	}

	portStr := u.Port()
	if portStr == "" {
		return 0, fmt.Errorf("endpoint has no port: %q", endpoint)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: %w", portStr, err)
	}

	return port, nil
}

func findContainerByPort(containers []string, port int) (string, bool) {
	suffix := fmt.Sprintf("-%d", port)
	for _, name := range containers {
		if strings.HasSuffix(name, suffix) {
			return name, true
		}
	}
	return "", false
}

func requireResiliencyEnabled(t testing.TB) {
	enabled := os.Getenv("TEST_ENABLE_RESILIENCY")
	if enabled != "on" && enabled != "1" {
		t.Skip("TEST_ENABLE_RESILIENCY is not set")
	}
}

func requireMinimumCoordinators(t testing.TB, client arangodb.Client, min int) {
	t.Helper()

	timeout := defaultTestTimeout
	if isK8S() {
		timeout = resiliencyClusterRecoveryTimeout
	}

	err := NewTimeout(func() error {
		return withContext(3*time.Second, func(ctx context.Context) error {
			coordinators := countHealthyCoordinators(ctx, client)
			if coordinators >= min {
				t.Logf("Cluster has %d coordinator(s)", coordinators)
				return Interrupt{}
			}

			if isK8S() {
				t.Logf("Waiting for coordinators in cluster health: got %d, want at least %d", coordinators, min)
			}
			return nil
		})
	}).Timeout(timeout, time.Second)

	if err != nil {
		hint := ""
		if isK8S() {
			hint = fmt.Sprintf(" (deploy with K8S_COORDINATORS_COUNT=%d via: K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency)", min)
		}
		t.Fatalf("resiliency tests require at least %d coordinators%s: %v", min, hint, err)
	}
}

func countHealthyCoordinators(ctx context.Context, client arangodb.Client) int {
	health, err := client.Health(ctx)
	if err != nil {
		return 0
	}

	coordinators := 0
	for _, server := range health.Health {
		if server.Role == arangodb.ServerRoleCoordinator {
			coordinators++
		}
	}
	return coordinators
}
