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
// Kubernetes chaos + ingress checks. HTTP to ArangoDB still goes through the
// ingress URL on the driver client; kubectl here only manages coordinator pods
// and inspects Service endpoints (arangodb-driver-tests-ea).

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

type k8sChaos struct {
	t          testing.TB
	namespace  string
	deployment string
}

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

func (c *k8sChaos) killRandomCoordinator(ctx context.Context, client arangodb.Client) (CoordinatorTarget, error) {
	_ = ctx
	_ = client

	pods, err := c.listRunningCoordinatorPods()
	if err != nil {
		return CoordinatorTarget{}, err
	}

	pod := pods[rand.Intn(len(pods))]
	c.t.Logf("Deleting coordinator pod %s (namespace %s)", pod, c.namespace)

	out, err := exec.Command(
		"kubectl", "-n", c.namespace, "delete", "pod", pod,
		"--grace-period=0", "--force",
	).CombinedOutput()
	if err != nil {
		return CoordinatorTarget{}, fmt.Errorf("kubectl delete pod %s failed: %w: %s", pod, err, strings.TrimSpace(string(out)))
	}

	return CoordinatorTarget{ResourceName: pod}, nil
}

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

func isCoordinatorPod(name string) bool {
	// kube-arangodb coordinator pods contain "-crdn-" in their name.
	return strings.Contains(name, "-crdn-")
}

func expectedCoordinatorCount() int {
	if v := strings.TrimSpace(os.Getenv("K8S_COORDINATORS_COUNT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 3
}

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
