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
// Docker chaos helpers (arangodb-starter). Kills coordinator containers via
// /var/run/docker.sock; does not send ArangoDB HTTP requests.

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

const dockerSocketPath = "/var/run/docker.sock"

type dockerChaos struct {
	t             testing.TB
	testContainer string
	client        *dockerClient
}

type dockerClient struct {
	httpClient *http.Client
}

type dockerContainer struct {
	Names []string `json:"Names"`
}

func newDockerChaos(t testing.TB) *dockerChaos {
	t.Helper()

	if _, err := os.Stat(dockerSocketPath); os.IsNotExist(err) {
		t.Skip("docker socket is not mounted at /var/run/docker.sock")
	}

	testContainer := strings.TrimSpace(os.Getenv("TESTCONTAINER"))
	if testContainer == "" {
		testContainer = "go-driver-test"
	}

	return &dockerChaos{
		t:             t,
		testContainer: testContainer,
		client:        newDockerClient(),
	}
}

func newDockerClient() *dockerClient {
	return &dockerClient{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", dockerSocketPath)
				},
			},
		},
	}
}

func (c *dockerChaos) killRandomCoordinator(ctx context.Context, client arangodb.Client) (CoordinatorTarget, error) {
	targets, err := c.listCoordinators(ctx, client)
	if err != nil {
		return CoordinatorTarget{}, err
	}

	target := targets[rand.Intn(len(targets))]
	return c.killCoordinatorTarget(target)
}

func (c *dockerChaos) killCoordinatorByServerID(ctx context.Context, client arangodb.Client, serverID string) (CoordinatorTarget, error) {
	targets, err := c.listCoordinators(ctx, client)
	if err != nil {
		return CoordinatorTarget{}, err
	}

	for _, target := range targets {
		if target.ServerID == serverID {
			return c.killCoordinatorTarget(target)
		}
	}

	return CoordinatorTarget{}, fmt.Errorf("coordinator %q not found in cluster health", serverID)
}

func (c *dockerChaos) killCoordinatorTarget(target CoordinatorTarget) (CoordinatorTarget, error) {
	c.t.Logf("Killing coordinator container %s (server %s, endpoint %s)", target.ResourceName, target.ServerID, target.Endpoint)

	if err := c.client.killContainer(target.ResourceName); err != nil {
		return CoordinatorTarget{}, err
	}

	return target, nil
}

func (c *dockerChaos) waitForInfrastructureRecovery(_ time.Duration) {}

func (c *dockerChaos) listCoordinators(ctx context.Context, client arangodb.Client) ([]CoordinatorTarget, error) {
	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster health: %w", err)
	}

	containers, err := c.listStarterContainers()
	if err != nil {
		return nil, err
	}

	var targets []CoordinatorTarget
	for id, server := range health.Health {
		if server.Role != arangodb.ServerRoleCoordinator {
			continue
		}

		port, err := endpointPort(server.Endpoint)
		if err != nil {
			c.t.Logf("Skipping coordinator endpoint %q: %v", server.Endpoint, err)
			continue
		}

		containerName, ok := findContainerByPort(containers, port)
		if !ok {
			return nil, fmt.Errorf("no docker container found for coordinator endpoint %q (port %d)", server.Endpoint, port)
		}

		targets = append(targets, CoordinatorTarget{
			ServerID:     string(id),
			Endpoint:     connection.FixupEndpointURLScheme(server.Endpoint),
			ResourceName: containerName,
		})
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no coordinators found in cluster health")
	}

	return targets, nil
}

func (c *dockerChaos) listStarterContainers() ([]string, error) {
	prefix := c.testContainer + "-s-"

	resp, err := c.client.request(http.MethodGet, "/containers/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker list containers failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var containers []dockerContainer
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("decode docker containers response: %w", err)
	}

	var names []string
	for _, container := range containers {
		for _, name := range container.Names {
			name = strings.TrimPrefix(name, "/")
			if strings.HasPrefix(name, prefix) {
				names = append(names, name)
			}
		}
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("no arangodb-starter containers found with prefix %q", prefix)
	}

	return names, nil
}

func (d *dockerClient) killContainer(name string) error {
	path := "/containers/" + url.PathEscape(name) + "/kill"
	resp, err := d.request(http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("docker kill %s failed with status %d: %s", name, resp.StatusCode, strings.TrimSpace(string(body)))
}

func (d *dockerClient) request(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://docker"+path, body)
	if err != nil {
		return nil, err
	}
	return d.httpClient.Do(req)
}
