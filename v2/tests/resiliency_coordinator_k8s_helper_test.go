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
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

const (
	defaultK8sNamespace          = "default"
	defaultK8sDeployment         = "arangodb-driver-tests"
	minCoordinatorResiliencyPods = 3
)

// requireResiliencyK8sCoordinatorMode skips unless the test runs in k8s cluster mode with kubectl and
// at least three coordinators configured for coordinator failure scenarios.
func requireResiliencyK8sCoordinatorMode(t testing.TB) {
	requireResiliencyK8sIngressMode(t)
	requireKubectl(t)

	count := expectedCoordinatorCount()
	if count < minCoordinatorResiliencyPods {
		t.Skipf("coordinator resiliency tests require at least %d coordinators, got %d",
			minCoordinatorResiliencyPods, count)
	}
}

func k8sNamespace() string {
	if v := strings.TrimSpace(os.Getenv("K8S_NAMESPACE")); v != "" {
		return v
	}
	return defaultK8sNamespace
}

func k8sDeployment() string {
	if v := strings.TrimSpace(os.Getenv("K8S_DEPLOYMENT")); v != "" {
		return v
	}
	return defaultK8sDeployment
}

func expectedCoordinatorCount() int {
	if v := strings.TrimSpace(os.Getenv("K8S_COORDINATORS_COUNT")); v != "" {
		count, err := strconv.Atoi(v)
		if err == nil && count > 0 {
			return count
		}
	}
	return 1
}

func coordinatorLabelSelector() string {
	return "arango_deployment=" + k8sDeployment() + ",role=coordinator"
}

func listCoordinatorPods(t testing.TB) []string {
	t.Helper()
	requireKubectl(t)

	cmd := exec.Command(
		"kubectl", "-n", k8sNamespace(),
		"get", "pods", "-l", coordinatorLabelSelector(),
		"-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}",
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "kubectl get coordinator pods failed: %s", string(output))

	pods := strings.Fields(string(output))
	require.NotEmpty(t, pods, "no coordinator pods found with selector %q", coordinatorLabelSelector())
	return pods
}

func deleteCoordinatorPod(t testing.TB, pod string) {
	t.Helper()
	requireKubectl(t)

	t.Logf("Deleting coordinator pod %s/%s", k8sNamespace(), pod)
	cmd := exec.Command(
		"kubectl", "-n", k8sNamespace(),
		"delete", "pod", pod,
		"--grace-period=0", "--force", "--wait=false",
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "kubectl delete pod failed: %s", string(output))
}

func restartAllCoordinators(t testing.TB) {
	t.Helper()

	pods := listCoordinatorPods(t)
	t.Logf("Restarting %d coordinator pods in %s/%s", len(pods), k8sNamespace(), k8sDeployment())
	for _, pod := range pods {
		deleteCoordinatorPod(t, pod)
	}
	waitForCoordinatorsReady(t, len(pods))
}

func coordinatorPodForClient(t testing.TB, client arangodb.Client) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	serverID, err := client.ServerID(ctx)
	require.NoError(t, err)

	podToken := coordinatorPodToken(serverID)
	pods := listCoordinatorPods(t)
	var matches []string
	for _, pod := range pods {
		if strings.Contains(strings.ToLower(pod), podToken) {
			matches = append(matches, pod)
		}
	}

	require.Len(t, matches, 1,
		"expected exactly one coordinator pod for server ID %q (token %q), got %v among %v",
		serverID, podToken, matches, pods)
	return matches[0]
}

// coordinatorPodToken returns the kube-arangodb pod name fragment for a coordinator server ID.
// Server IDs use "CRDN-<id>" while pod names use "crdn-<id>-<suffix>".
func coordinatorPodToken(serverID string) string {
	token := strings.TrimPrefix(serverID, "CRDN-")
	token = strings.TrimPrefix(token, "crdn-")
	return strings.ToLower(token)
}

func killCoordinatorForClient(t testing.TB, client arangodb.Client) {
	t.Helper()
	deleteCoordinatorPod(t, coordinatorPodForClient(t, client))
}

// ensureCoordinatorsRecovered waits until all expected coordinator pods are ready again.
// When client is non-nil, also verifies the cluster serves requests via client.Version.
// Call after coordinator chaos and before the next resiliency subtest or client connection.
func ensureCoordinatorsRecovered(t testing.TB, client arangodb.Client) {
	t.Helper()
	waitForCoordinatorsReady(t, expectedCoordinatorCount())
	if client != nil {
		waitForSuccessfulVersion(t, client, 30*time.Second)
	}
}

func waitForCoordinatorsReady(t testing.TB, expectedCount int) {
	t.Helper()
	requireKubectl(t)

	namespace := k8sNamespace()
	selector := coordinatorLabelSelector()
	timeout := 10 * time.Minute

	t.Logf("Waiting for %d coordinator pods to become ready in %s", expectedCount, namespace)
	NewTimeout(func() error {
		cmd := exec.Command(
			"kubectl", "-n", namespace,
			"wait", "--for=condition=ready", "pod",
			"-l", selector,
			"--timeout=30s",
		)
		if err := cmd.Run(); err != nil {
			return nil
		}

		cmd = exec.Command(
			"kubectl", "-n", namespace,
			"get", "pods", "-l", selector,
			"-o", "jsonpath={.items[*].metadata.name}",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("kubectl get pods failed: %v: %s", err, string(output))
			return nil
		}

		if len(strings.Fields(string(output))) >= expectedCount {
			return Interrupt{}
		}
		return nil
	}).TimeoutT(t, timeout, 5*time.Second)
}
