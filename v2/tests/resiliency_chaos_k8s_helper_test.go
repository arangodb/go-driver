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
// Kubernetes chaos injection and ingress readiness checks for resiliency tests.
//
// ArangoDB HTTP traffic is handled elsewhere (driver client). This file uses
// kubectl only to delete coordinator pods, list pod state, and read Service
// Endpoints (in-cluster: arangodb-driver-tests; ingress: arangodb-driver-tests-ea).

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

	"github.com/arangodb/go-driver/v2/arangodb"
)

// resiliencyChaosPodRemovalTimeout is how long deleteCoordinatorPod waits for a
// killed coordinator to leave Service Endpoints before failover probes run.
const resiliencyChaosPodRemovalTimeout = 60 * time.Second

// k8sChaos injects coordinator failures on kube-arangodb via kubectl.
type k8sChaos struct {
	t          testing.TB
	namespace  string
	deployment string
}

// newK8sChaos builds a kubectl-backed chaos backend from K8S_NAMESPACE and
// K8S_DEPLOYMENT (defaults: default / arangodb-driver-tests). Skips the test
// when kubectl is not on PATH.
func newK8sChaos(t testing.TB) *k8sChaos {
	t.Helper()

	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("kubectl is not available in the test environment")
	}

	namespace := strings.TrimSpace(os.Getenv("K8S_NAMESPACE"))
	if namespace == "" {
		namespace = "default"
	}

	deployment := strings.TrimSpace(os.Getenv("K8S_DEPLOYMENT"))
	if deployment == "" {
		deployment = "arangodb-driver-tests"
	}

	return &k8sChaos{
		t:          t,
		namespace:  namespace,
		deployment: deployment,
	}
}

// killRandomCoordinator selects a coordinator from cluster health at random and
// deletes its pod. Implements chaosBackend.
func (c *k8sChaos) killRandomCoordinator(ctx context.Context, client arangodb.Client) (CoordinatorTarget, error) {
	targets, err := c.listCoordinatorTargets(ctx, client)
	if err != nil {
		return CoordinatorTarget{}, err
	}

	target := targets[rand.Intn(len(targets))]
	return c.deleteCoordinatorPod(target)
}

// killCoordinatorByServerID deletes the coordinator pod matching serverID (e.g.
// CRDN-abc from GET /_admin/status). Implements chaosBackend.
func (c *k8sChaos) killCoordinatorByServerID(ctx context.Context, client arangodb.Client, serverID string) (CoordinatorTarget, error) {
	target, err := c.findCoordinatorTarget(ctx, client, serverID)
	if err != nil {
		return CoordinatorTarget{}, err
	}
	return c.deleteCoordinatorPod(target)
}

// deleteCoordinatorPod force-deletes a coordinator pod and waits until its IP
// is removed from the in-cluster Service Endpoints. Uses --wait=false on delete
// to avoid blocking on kube-arangodb Terminating pods; endpoint removal is
// polled separately via waitUntilCoordinatorUnavailable.
func (c *k8sChaos) deleteCoordinatorPod(target CoordinatorTarget) (CoordinatorTarget, error) {
	c.t.Logf("Deleting coordinator pod %s (server %s, namespace %s)", target.ResourceName, target.ServerID, c.namespace)

	podIP, _ := c.coordinatorPodIP(target.ResourceName)
	if podIP != "" {
		c.t.Logf("Coordinator pod %s has IP %s before delete", target.ResourceName, podIP)
	}

	// --wait=false: do not block until the pod object is gone. kube-arangodb may
	// recreate the same coordinator quickly; default kubectl delete can hang for
	// minutes while the operator reconciles Terminating pods.
	out, err := exec.Command(
		"kubectl", "-n", c.namespace, "delete", "pod", target.ResourceName,
		"--grace-period=0", "--force", "--wait=false",
	).CombinedOutput()
	if err != nil {
		return CoordinatorTarget{}, fmt.Errorf("kubectl delete pod %s failed: %w: %s", target.ResourceName, err, strings.TrimSpace(string(out)))
	}

	c.waitUntilCoordinatorUnavailable(target.ResourceName, podIP, resiliencyChaosPodRemovalTimeout)
	return target, nil
}

// coordinatorPodIP returns the podIP of a coordinator pod before delete, used to
// detect when that IP leaves Service Endpoints after chaos injection.
func (c *k8sChaos) coordinatorPodIP(podName string) (string, error) {
	out, err := exec.Command(
		"kubectl", "-n", c.namespace, "get", "pod", podName,
		"-o", "jsonpath={.status.podIP}",
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl get pod %s IP failed: %w: %s", podName, err, strings.TrimSpace(string(out)))
	}

	ip := strings.TrimSpace(string(out))
	if ip == "" {
		return "", fmt.Errorf("pod %q has no podIP", podName)
	}
	return ip, nil
}

// coordinatorServiceEndpointIPs lists backend pod IPs registered on the
// in-cluster coordinator Service (K8S_DEPLOYMENT, not -ea). Requires RBAC get/list
// on endpoints for the in-cluster resiliency ServiceAccount.
func (c *k8sChaos) coordinatorServiceEndpointIPs() ([]string, error) {
	out, err := exec.Command(
		"kubectl", "-n", c.namespace, "get", "endpoints", c.deployment,
		"-o", "jsonpath={.subsets[*].addresses[*].ip}",
	).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kubectl get endpoints %s failed: %w: %s", c.deployment, err, strings.TrimSpace(string(out)))
	}

	var ips []string
	for _, part := range strings.Fields(string(out)) {
		if part != "" {
			ips = append(ips, part)
		}
	}
	return ips, nil
}

// coordinatorPodPhase returns the Kubernetes phase of podName. found is false when
// the pod object no longer exists (NotFound).
func (c *k8sChaos) coordinatorPodPhase(podName string) (phase string, found bool, err error) {
	out, err := exec.Command(
		"kubectl", "-n", c.namespace, "get", "pod", podName,
		"-o", "jsonpath={.status.phase}",
	).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "NotFound") {
			return "", false, nil
		}
		return "", false, fmt.Errorf("kubectl get pod %s phase failed: %w: %s", podName, err, msg)
	}

	return strings.TrimSpace(string(out)), true, nil
}

// waitUntilCoordinatorUnavailable blocks until the deleted coordinator is no longer
// reachable through the in-cluster Service (IP removed from Endpoints). With
// --wait=false on delete, probing immediately would still hit the live pod.
func (c *k8sChaos) waitUntilCoordinatorUnavailable(podName, podIP string, timeout time.Duration) {
	tt, ok := c.t.(*testing.T)
	if !ok {
		c.t.Fatal("waitUntilCoordinatorUnavailable requires *testing.T")
	}

	NewTimeout(func() error {
		if podIP != "" {
			ips, err := c.coordinatorServiceEndpointIPs()
			if err != nil {
				c.t.Logf("Waiting for coordinator pod %s to leave service endpoints: %v", podName, err)
				return nil
			}

			for _, ip := range ips {
				if ip == podIP {
					c.t.Logf("Waiting for coordinator pod %s IP %s to leave service %s endpoints (still %v)", podName, podIP, c.deployment, ips)
					return nil
				}
			}

			c.t.Logf("Coordinator pod %s IP %s removed from service %s endpoints", podName, podIP, c.deployment)
			return Interrupt{}
		}

		phase, found, err := c.coordinatorPodPhase(podName)
		if err != nil {
			c.t.Logf("Waiting for coordinator pod %s removal: %v", podName, err)
			return nil
		}
		if !found || phase != "Running" {
			c.t.Logf("Coordinator pod %s is no longer running (found=%v phase=%q)", podName, found, phase)
			return Interrupt{}
		}

		c.t.Logf("Waiting for coordinator pod %s to leave Running (phase=%s)", podName, phase)
		return nil
	}).TimeoutT(tt, timeout, 500*time.Millisecond)
}

// findCoordinatorTarget resolves serverID from cluster health to a CoordinatorTarget
// with the matching running pod name.
func (c *k8sChaos) findCoordinatorTarget(ctx context.Context, client arangodb.Client, serverID string) (CoordinatorTarget, error) {
	targets, err := c.listCoordinatorTargets(ctx, client)
	if err != nil {
		return CoordinatorTarget{}, err
	}

	for _, target := range targets {
		if target.ServerID == serverID {
			return target, nil
		}
	}

	return CoordinatorTarget{}, fmt.Errorf("coordinator %q not found in cluster health", serverID)
}

// listCoordinatorTargets maps every coordinator in client.Health() to its kube-arangodb
// pod name via findCoordinatorPod.
func (c *k8sChaos) listCoordinatorTargets(ctx context.Context, client arangodb.Client) ([]CoordinatorTarget, error) {
	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("cluster health: %w", err)
	}

	var targets []CoordinatorTarget
	for id, server := range health.Health {
		if server.Role != arangodb.ServerRoleCoordinator {
			continue
		}

		serverID := string(id)
		podName, err := c.findCoordinatorPod(serverID, server.Endpoint)
		if err != nil {
			return nil, err
		}

		targets = append(targets, CoordinatorTarget{
			ServerID:     serverID,
			Endpoint:     server.Endpoint,
			ResourceName: podName,
		})
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no coordinators found in cluster health")
	}

	return targets, nil
}

// findCoordinatorPod resolves a coordinator server ID to its pod name.
// kube-arangodb health endpoints are often internal *.svc DNS names, not pod IPs,
// so pod lookup prefers the server ID embedded in the pod name (e.g. CRDN-zz2qsc4m
// → arangodb-driver-tests-crdn-zz2qsc4m-a9f77a).
func (c *k8sChaos) findCoordinatorPod(serverID, endpoint string) (string, error) {
	if podName, err := c.findCoordinatorPodByServerID(serverID); err == nil {
		return podName, nil
	}

	host, err := endpointHost(endpoint)
	if err != nil {
		return "", fmt.Errorf("coordinator %q: %w", serverID, err)
	}

	if podName, err := c.findCoordinatorPodByIP(host); err == nil {
		return podName, nil
	}

	return "", fmt.Errorf("no running coordinator pod found for server %q (endpoint host %q)", serverID, host)
}

// findCoordinatorPodByServerID matches a running coordinator pod whose name contains
// serverID (case-insensitive). Preferred lookup when health endpoints are internal
// *.svc DNS names rather than pod IPs.
func (c *k8sChaos) findCoordinatorPodByServerID(serverID string) (string, error) {
	marker := strings.ToLower(serverID)

	pods, err := c.listRunningCoordinatorPods()
	if err != nil {
		return "", err
	}

	for _, pod := range pods {
		if strings.Contains(strings.ToLower(pod), marker) {
			return pod, nil
		}
	}

	return "", fmt.Errorf("no running coordinator pod matching server ID %q", serverID)
}

// findCoordinatorPodByIP finds a running coordinator pod by podIP. Used as a
// fallback when server ID substring matching fails but health reports a routable IP.
func (c *k8sChaos) findCoordinatorPodByIP(ip string) (string, error) {
	out, err := exec.Command(
		"kubectl", "-n", c.namespace, "get", "pods",
		"-l", fmt.Sprintf("arango_deployment=%s", c.deployment),
		"--field-selector=status.phase=Running",
		"-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\t\"}{.status.podIP}{\"\\n\"}{end}",
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl get pods failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		if !isCoordinatorPod(parts[0]) {
			continue
		}
		if parts[1] == ip {
			return parts[0], nil
		}
	}

	return "", fmt.Errorf("no running coordinator pod with IP %q for deployment %q", ip, c.deployment)
}

// waitForInfrastructureRecovery polls until at least expectedCoordinatorCount()
// coordinator pods are Running. Implements chaosBackend; complements
// WaitForHealthyCluster (ArangoDB API) with a kubectl pod-count check.
func (c *k8sChaos) waitForInfrastructureRecovery(timeout time.Duration) {
	tt, ok := c.t.(*testing.T)
	if !ok {
		c.t.Fatal("waitForInfrastructureRecovery requires *testing.T")
	}

	expected := expectedCoordinatorCount()

	NewTimeout(func() error {
		pods, err := c.listRunningCoordinatorPods()
		if err != nil {
			c.t.Logf("Waiting for coordinator pods: %v", err)
			return nil
		}
		if len(pods) < expected {
			c.t.Logf("Waiting for coordinator pods: got %d, want %d", len(pods), expected)
			return nil
		}
		return Interrupt{}
	}).TimeoutT(tt, timeout, time.Second)
}

// listRunningCoordinatorPods returns Running pod names for the deployment that
// contain "-crdn-" (kube-arangodb coordinator naming convention).
func (c *k8sChaos) listRunningCoordinatorPods() ([]string, error) {
	out, err := exec.Command(
		"kubectl", "-n", c.namespace, "get", "pods",
		"-l", fmt.Sprintf("arango_deployment=%s", c.deployment),
		"--field-selector=status.phase=Running",
		"-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}",
	).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kubectl get pods failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	var pods []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isCoordinatorPod(line) {
			pods = append(pods, line)
		}
	}

	if len(pods) == 0 {
		return nil, fmt.Errorf("no running coordinator pods found for deployment %q in namespace %q", c.deployment, c.namespace)
	}

	return pods, nil
}

// isCoordinatorPod reports whether name follows kube-arangodb coordinator pod naming
// (contains "-crdn-").
func isCoordinatorPod(name string) bool {
	return strings.Contains(name, "-crdn-")
}

// expectedCoordinatorCount reads K8S_COORDINATORS_COUNT from the environment,
// defaulting to 3 when unset or invalid.
func expectedCoordinatorCount() int {
	if v := strings.TrimSpace(os.Getenv("K8S_COORDINATORS_COUNT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 3
}

// k8sDeploymentEnv returns K8S_NAMESPACE and K8S_DEPLOYMENT with the same
// defaults as newK8sChaos.
func k8sDeploymentEnv() (namespace, deployment string) {
	namespace = strings.TrimSpace(os.Getenv("K8S_NAMESPACE"))
	if namespace == "" {
		namespace = "default"
	}
	deployment = strings.TrimSpace(os.Getenv("K8S_DEPLOYMENT"))
	if deployment == "" {
		deployment = "arangodb-driver-tests"
	}
	return namespace, deployment
}

// ingressCoordinatorBackendCount counts coordinator backend addresses on the
// external-access Service (<deployment>-ea). Used before ingress resiliency tests.
func ingressCoordinatorBackendCount() (int, error) {
	namespace, deployment := k8sDeploymentEnv()
	out, err := exec.Command(
		"kubectl", "-n", namespace, "get", "endpoints", deployment+"-ea",
		"-o", "jsonpath={.subsets[*].addresses[*].ip}",
	).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("kubectl get endpoints %s-ea failed: %w: %s", deployment, err, strings.TrimSpace(string(out)))
	}

	count := 0
	for _, part := range strings.Fields(string(out)) {
		if part != "" {
			count++
		}
	}
	return count, nil
}

// waitForMinimumIngressBackends blocks until the ingress Service has at least min
// coordinator backends in Endpoints. No-op when not running against Kubernetes.
func waitForMinimumIngressBackends(t testing.TB, min int, timeout time.Duration) {
	t.Helper()
	if !isK8S() {
		return
	}

	tt, ok := t.(*testing.T)
	if !ok {
		return
	}

	NewTimeout(func() error {
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
	}).TimeoutT(tt, timeout, time.Second)
}
