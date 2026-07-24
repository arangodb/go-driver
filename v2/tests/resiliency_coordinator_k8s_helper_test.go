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
	"fmt"
	"math/rand"
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
	coordinatorRecoveryTimeout   = 5 * time.Minute
)

// CoordinatorTarget identifies a coordinator instance targeted by chaos injection.
type CoordinatorTarget struct {
	ServerID     string
	Endpoint     string
	ResourceName string
}

// killRandomCoordinator deletes one coordinator pod selected at random from cluster health.
// Only Health entries that map to a live pod are eligible: after a prior kill/recover,
// Health can briefly still list replaced server IDs with no matching pod (CircleCI flake).
func killRandomCoordinator(t testing.TB, client arangodb.Client) CoordinatorTarget {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var target CoordinatorTarget
	err := NewTimeout(func() error {
		health, err := client.Health(ctx)
		if err != nil {
			t.Logf("killRandomCoordinator: Health() not ready: %v", err)
			return nil
		}

		pods, err := tryListCoordinatorPods()
		if err != nil {
			t.Logf("killRandomCoordinator: listing pods: %v", err)
			return nil
		}
		var targets []CoordinatorTarget
		for id, server := range health.Health {
			if server.Role != arangodb.ServerRoleCoordinator {
				continue
			}
			serverID := string(id)
			pod, ok := matchCoordinatorPod(pods, serverID)
			if !ok {
				t.Logf("killRandomCoordinator: skipping %s (no live pod among %v)", serverID, pods)
				continue
			}
			targets = append(targets, CoordinatorTarget{
				ServerID:     serverID,
				Endpoint:     server.Endpoint,
				ResourceName: pod,
			})
		}
		if len(targets) == 0 {
			t.Logf("killRandomCoordinator: no pod-backed coordinators yet (health entries may be stale)")
			return nil
		}

		target = targets[rand.Intn(len(targets))]
		return Interrupt{}
	}).Timeout(coordinatorRecoveryTimeout, time.Second)
	require.NoError(t, err, "no pod-backed coordinator available to kill")
	require.NotEmpty(t, target.ResourceName)

	t.Logf("Killing coordinator pod %s (server %s)", target.ResourceName, target.ServerID)
	deleteCoordinatorPod(t, target.ResourceName)
	return target
}

func coordinatorPodForServerID(t testing.TB, serverID string) string {
	t.Helper()

	pods := listCoordinatorPods(t)
	pod, ok := matchCoordinatorPod(pods, serverID)
	require.True(t, ok,
		"expected exactly one coordinator pod for server ID %q (token %q) among %v",
		serverID, coordinatorPodToken(serverID), pods)
	return pod
}

// matchCoordinatorPod returns the unique live pod name for serverID, if any.
func matchCoordinatorPod(pods []string, serverID string) (string, bool) {
	podToken := coordinatorPodToken(serverID)
	var matches []string
	for _, pod := range pods {
		if strings.Contains(strings.ToLower(pod), podToken) {
			matches = append(matches, pod)
		}
	}
	if len(matches) != 1 {
		return "", false
	}
	return matches[0], true
}

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

	pods, err := tryListCoordinatorPods()
	require.NoError(t, err)
	require.NotEmpty(t, pods, "no coordinator pods found with selector %q", coordinatorLabelSelector())
	return pods
}

func tryListCoordinatorPods() ([]string, error) {
	cmd := exec.Command(
		"kubectl", "-n", k8sNamespace(),
		"get", "pods", "-l", coordinatorLabelSelector(),
		"-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kubectl get coordinator pods failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.Fields(string(output)), nil
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
	return coordinatorPodForServerID(t, serverID)
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

// killAllCoordinators force-deletes every coordinator pod. Used during active cursor reads
// because ingress load balancing can pin the cursor to a different coordinator than ServerID().
func killAllCoordinators(t testing.TB) {
	t.Helper()

	pods := listCoordinatorPods(t)
	t.Logf("Killing %d coordinator pods during active cursor in %s/%s", len(pods), k8sNamespace(), k8sDeployment())
	for _, pod := range pods {
		deleteCoordinatorPod(t, pod)
	}
}

// ensureCoordinatorsRecovered waits until coordinators, the ArangoDeployment, ingress backends,
// and driver requests through ingress are healthy again.
func ensureCoordinatorsRecovered(t testing.TB, client arangodb.Client) {
	t.Helper()
	waitForCoordinatorsReady(t, expectedCoordinatorCount())
	waitForArangoDeploymentReady(t)
	waitForExternalAccessEndpoints(t)
	if client != nil {
		waitForIngressRecovered(t, client, coordinatorRecoveryTimeout)
		waitForClusterStable(t, client, time.Minute)
	}
}

func waitForArangoDeploymentReady(t testing.TB) {
	t.Helper()
	requireKubectl(t)

	namespace := k8sNamespace()
	deployment := k8sDeployment()
	timeout := 10 * time.Minute

	t.Logf("Waiting for ArangoDeployment %s/%s to become ready", namespace, deployment)
	err := NewTimeout(func() error {
		cmd := exec.Command(
			"kubectl", "-n", namespace,
			"get", "arangodeployment", deployment,
			"-o", "jsonpath={.status.conditions[?(@.type==\"Ready\")].status}",
		)
		output, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Logf("kubectl get arangodeployment failed: %v: %s", runErr, string(output))
			return nil
		}
		if strings.TrimSpace(string(output)) == "True" {
			return Interrupt{}
		}
		return nil
	}).Timeout(timeout, 5*time.Second)
	if err != nil {
		require.NoError(t, err, "ArangoDeployment %s/%s did not become ready within %s", namespace, deployment, timeout)
	}
}

func waitForExternalAccessEndpoints(t testing.TB) {
	t.Helper()
	requireKubectl(t)

	namespace := k8sNamespace()
	service := k8sDeployment() + "-ea"
	timeout := 10 * time.Minute

	t.Logf("Waiting for service/%s to have ready endpoints in %s", service, namespace)
	err := NewTimeout(func() error {
		cmd := exec.Command(
			"kubectl", "-n", namespace,
			"get", "endpoints", service,
			"-o", "jsonpath={.subsets[*].addresses[*].ip}",
		)
		output, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Logf("kubectl get endpoints failed: %v: %s", runErr, string(output))
			return nil
		}
		if len(strings.Fields(string(output))) > 0 {
			return Interrupt{}
		}
		return nil
	}).Timeout(timeout, 5*time.Second)
	if err != nil {
		require.NoError(t, err, "service/%s did not get ready endpoints within %s", service, timeout)
	}
}

// waitForIngressRecovered retries client.Version until ingress and coordinators accept traffic.
func waitForIngressRecovered(t testing.TB, client arangodb.Client, timeout time.Duration) {
	t.Helper()

	t.Logf("Waiting for ingress to serve Version() after coordinator recovery (budget: %s)", timeout)

	var lastErr error
	attempts := 0
	err := NewTimeout(func() error {
		attempts++
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.Version(ctx)
		if err == nil {
			t.Logf("ingress recovered after %d Version() attempt(s)", attempts)
			return Interrupt{}
		}

		lastErr = err
		if attempts == 1 || attempts%10 == 0 {
			t.Logf("ingress not ready yet (attempt %d): %v", attempts, err)
		}
		return nil
	}).Timeout(timeout, 500*time.Millisecond)
	if err != nil {
		require.NoError(t, err, "ingress did not recover within %s (last Version error: %v)", timeout, lastErr)
	}
}

func waitForCoordinatorsReady(t testing.TB, expectedCount int) {
	t.Helper()
	requireKubectl(t)

	namespace := k8sNamespace()
	selector := coordinatorLabelSelector()
	timeout := 10 * time.Minute

	t.Logf("Waiting for %d coordinator pods to become ready in %s", expectedCount, namespace)
	var lastLog time.Time
	err := NewTimeout(func() error {
		readyCount, countErr := readyCoordinatorPodCount(namespace, selector)
		if countErr != nil {
			t.Logf("kubectl get coordinator pods failed: %v", countErr)
			return nil
		}
		if readyCount >= expectedCount {
			t.Logf("coordinator pods ready: %d/%d", readyCount, expectedCount)
			return Interrupt{}
		}
		if lastLog.IsZero() || time.Since(lastLog) >= 15*time.Second {
			t.Logf("coordinator pods ready: %d/%d", readyCount, expectedCount)
			lastLog = time.Now()
		}
		return nil
	}).Timeout(timeout, 5*time.Second)
	if err != nil {
		require.NoError(t, err, "expected %d ready coordinator pods within %s", expectedCount, timeout)
	}
}

func ingressCoordinatorBackendCount() (int, error) {
	out, err := exec.Command(
		"kubectl", "-n", k8sNamespace(),
		"get", "endpoints", k8sDeployment()+"-ea",
		"-o", "jsonpath={.subsets[*].addresses[*].ip}",
	).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("kubectl get endpoints %s-ea failed: %w: %s",
			k8sDeployment(), err, strings.TrimSpace(string(out)))
	}

	count := 0
	for _, part := range strings.Fields(string(out)) {
		if part != "" {
			count++
		}
	}
	return count, nil
}

// waitForMinimumIngressBackends blocks until the ingress external-access service has at
// least min coordinator backends registered in Endpoints.
func waitForMinimumIngressBackends(t testing.TB, min int, timeout time.Duration) {
	t.Helper()
	if !isK8S() {
		return
	}

	t.Logf("Waiting for at least %d ingress coordinator backend(s)", min)
	err := NewTimeout(func() error {
		count, err := ingressCoordinatorBackendCount()
		if err != nil {
			t.Logf("Waiting for ingress coordinator backends: %v", err)
			return nil
		}
		if count >= min {
			t.Logf("Ingress service has %d coordinator backend(s)", count)
			return Interrupt{}
		}
		t.Logf("Waiting for ingress coordinator backends: got %d, want at least %d", count, min)
		return nil
	}).Timeout(timeout, time.Second)
	if err != nil {
		require.NoError(t, err, "ingress did not reach %d coordinator backend(s) within %s", min, timeout)
	}
}

// requireMinimumCoordinators blocks until cluster health reports at least min coordinators.
func requireMinimumCoordinators(t testing.TB, client arangodb.Client, min int) {
	t.Helper()

	timeout := defaultTestTimeout
	if isK8S() {
		timeout = coordinatorRecoveryTimeout
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

// waitForHealthyCoordinatorCount retries until cluster health reports at least expected
// coordinators that each map to a live pod. A bare Health() count can include replaced
// server IDs after kill/recover (no pod yet / ghost entry), which then breaks the next kill.
func waitForHealthyCoordinatorCount(t testing.TB, client arangodb.Client, expected int, timeout time.Duration) {
	t.Helper()

	err := NewTimeout(func() error {
		return withContext(10*time.Second, func(ctx context.Context) error {
			health, err := client.Health(ctx)
			if err != nil {
				t.Logf("Waiting for cluster health coordinator count: Health() error: %v", err)
				return nil
			}

			pods, err := tryListCoordinatorPods()
			if err != nil {
				t.Logf("Waiting for cluster health coordinator count: list pods: %v", err)
				return nil
			}

			matched := 0
			for id, server := range health.Health {
				if server.Role != arangodb.ServerRoleCoordinator {
					continue
				}
				if _, ok := matchCoordinatorPod(pods, string(id)); ok {
					matched++
				}
			}
			if matched >= expected && len(pods) >= expected {
				t.Logf("Cluster health reports %d pod-backed coordinator(s) (want >= %d; pods=%d)",
					matched, expected, len(pods))
				return Interrupt{}
			}
			t.Logf("Waiting for pod-backed coordinators: matched=%d pods=%d want>=%d", matched, len(pods), expected)
			return nil
		})
	}).Timeout(timeout, time.Second)

	require.NoError(t, err, "cluster health did not report at least %d pod-backed coordinators within %s", expected, timeout)
}

func readyCoordinatorPodCount(namespace, selector string) (int, error) {
	cmd := exec.Command(
		"kubectl", "-n", namespace,
		"get", "pods", "-l", selector,
		"--no-headers",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	ready := 0
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		parts := strings.Split(fields[1], "/")
		if len(parts) != 2 {
			continue
		}
		if parts[0] == parts[1] && parts[0] != "0" {
			ready++
		}
	}
	return ready, nil
}
