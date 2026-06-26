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
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	defaultIngressRestartNamespace  = "ingress-nginx"
	defaultIngressRestartDeployment = "ingress-nginx-controller"
)

// requireKubectl skips the test when kubectl is not available on the host PATH.
func requireKubectl(t testing.TB) {
	t.Helper()
	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("kubectl not found in PATH; resiliency tests need kubectl in the test container")
	}
}

// ingressRestartNamespace returns the namespace of the ingress-nginx controller deployment.
func ingressRestartNamespace() string {
	if v := os.Getenv("K8S_INGRESS_RESTART_NAMESPACE"); v != "" {
		return v
	}
	return defaultIngressRestartNamespace
}

// ingressRestartDeployment returns the name of the ingress-nginx controller deployment.
func ingressRestartDeployment() string {
	if v := os.Getenv("K8S_INGRESS_RESTART_DEPLOYMENT"); v != "" {
		return v
	}
	return defaultIngressRestartDeployment
}

// restartIngressController triggers a rolling restart of the ingress-nginx controller.
func restartIngressController(t testing.TB) {
	t.Helper()
	requireKubectl(t)

	namespace := ingressRestartNamespace()
	deployment := ingressRestartDeployment()

	t.Logf("Restarting ingress deployment %s/%s", namespace, deployment)
	cmd := exec.Command(
		"kubectl", "rollout", "restart",
		"deployment", deployment,
		"-n", namespace,
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "kubectl rollout restart failed: %s", string(output))
}

// waitForIngressControllerReady blocks until the ingress-nginx controller rollout completes.
func waitForIngressControllerReady(t testing.TB) {
	t.Helper()
	requireKubectl(t)

	namespace := ingressRestartNamespace()
	deployment := ingressRestartDeployment()
	timeout := 6 * time.Minute

	t.Logf("Waiting for ingress deployment %s/%s to become ready", namespace, deployment)
	NewTimeout(func() error {
		cmd := exec.Command(
			"kubectl", "-n", namespace,
			"rollout", "status", "deployment", deployment,
			"--timeout=30s",
		)
		if err := cmd.Run(); err != nil {
			return nil
		}
		return Interrupt{}
	}).TimeoutT(t, timeout, 5*time.Second)
}
