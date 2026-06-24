#!/usr/bin/env bash
#
# Shared Kubernetes runner for go-driver integration tests.
#
# Sets up kube-arangodb + ArangoDeployment + Ingress, then runs a caller-supplied
# test command against the external ingress endpoint.
#
# Commands:
#   run        Deploy cluster, run tests inside Docker (normal k8s integration tests)
#   run-host   Deploy cluster, run tests on the host (needs kubectl from tests, e.g. resiliency)
#   start      Deploy cluster only (no test command)
#   setup-kind Create local kind cluster with ingress-nginx
#   cleanup    Remove ArangoDeployment, Ingress, and secrets
#
# Test env wiring:
#   K8S_TEST_*_ENV variables (lines below) name the env vars passed to Make/go test.
#   Example: K8S_TEST_ENDPOINTS_ENV defaults to TEST_ENDPOINTS_OVERRIDE.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# --- Kubernetes / ArangoDB deployment settings ---
KUBE_ARANGODB_VERSION="${KUBE_ARANGODB_VERSION:-1.2.43}"
K8S_NAMESPACE="${K8S_NAMESPACE:-default}"
K8S_DEPLOYMENT="${K8S_DEPLOYMENT:-arangodb-driver-tests}"
K8S_MODE="${K8S_MODE:-Cluster}"
K8S_ENVIRONMENT="${K8S_ENVIRONMENT:-Development}"
K8S_EXTERNAL_ACCESS="${K8S_EXTERNAL_ACCESS:-NodePort}"
K8S_PORT="${K8S_PORT:-8529}"
K8S_INGRESS_NAME="${K8S_INGRESS_NAME:-${K8S_DEPLOYMENT}-ingress}"
K8S_INGRESS_HOST="${K8S_INGRESS_HOST:-arangodb.local}"
K8S_INGRESS_ADDRESS="${K8S_INGRESS_ADDRESS:-}"
K8S_INGRESS_TLS="${K8S_INGRESS_TLS:-true}"
K8S_INGRESS_TLS_SECRET="${K8S_INGRESS_TLS_SECRET:-${K8S_DEPLOYMENT}-ingress-tls}"
K8S_INGRESS_CLASS="${K8S_INGRESS_CLASS:-nginx}"
K8S_INGRESS_PORT="${K8S_INGRESS_PORT:-}"
K8S_WAIT_TIMEOUT="${K8S_WAIT_TIMEOUT:-15m}"
K8S_STUCK_INIT_TIMEOUT="${K8S_STUCK_INIT_TIMEOUT:-5m}"
K8S_KEEP_DEPLOYMENT="${K8S_KEEP_DEPLOYMENT:-false}"
K8S_DELETE_NAMESPACE="${K8S_DELETE_NAMESPACE:-false}"
K8S_INSTALL_OPERATOR="${K8S_INSTALL_OPERATOR:-true}"
K8S_AUTHENTICATION="${K8S_AUTHENTICATION:-true}"
K8S_TEST_AUTHENTICATION="${K8S_TEST_AUTHENTICATION:-basic}"
K8S_TLS="${K8S_TLS:-false}"
K8S_TEST_WORKDIR="${K8S_TEST_WORKDIR:-${ROOT_DIR}}"

# --- Names of env vars passed to the driver test command (values built at runtime) ---
# Example: K8S_TEST_ENDPOINTS_ENV=TEST_ENDPOINTS_OVERRIDE prints
#   TEST_ENDPOINTS_OVERRIDE=http://arangodb.local
K8S_TEST_ENDPOINTS_ENV="${K8S_TEST_ENDPOINTS_ENV:-TEST_ENDPOINTS_OVERRIDE}"
K8S_TEST_AUTHENTICATION_ENV="${K8S_TEST_AUTHENTICATION_ENV:-TEST_AUTHENTICATION_OVERRIDE}"
K8S_TEST_LEGACY_AUTHENTICATION_ENV="${K8S_TEST_LEGACY_AUTHENTICATION_ENV:-TEST_AUTHENTICATION}"
K8S_TEST_MODE_ENV="${K8S_TEST_MODE_ENV:-TEST_MODE_K8S}"
K8S_TEST_NOT_WAIT_UNTIL_READY_ENV="${K8S_TEST_NOT_WAIT_UNTIL_READY_ENV:-TEST_NOT_WAIT_UNTIL_READY}"
K8S_TEST_NET_ENV="${K8S_TEST_NET_ENV:-TEST_NET_OVERRIDE}"

# --- kind cluster settings (local development) ---
K8S_KIND_CLUSTER_NAME="${K8S_KIND_CLUSTER_NAME:-arangodb-driver-tests}"
K8S_KIND_NODE_IMAGE="${K8S_KIND_NODE_IMAGE:-kindest/node:v1.31.12}"
K8S_KIND_RETAIN="${K8S_KIND_RETAIN:-false}"
K8S_DELETE_KIND_CLUSTER="${K8S_DELETE_KIND_CLUSTER:-false}"
K8S_INGRESS_NGINX_VERSION="${K8S_INGRESS_NGINX_VERSION:-controller-v1.12.1}"
K8S_INGRESS_NGINX_MANIFEST="${K8S_INGRESS_NGINX_MANIFEST:-https://raw.githubusercontent.com/kubernetes/ingress-nginx/${K8S_INGRESS_NGINX_VERSION}/deploy/static/provider/kind/deploy.yaml}"

ARANGODB="${ARANGODB:-arangodb/enterprise-preview:latest}"
KUBE_ARANGODB_IMAGE="${KUBE_ARANGODB_IMAGE:-arangodb/kube-arangodb:${KUBE_ARANGODB_VERSION}}"
ARANGO_ROOT_PASSWORD="${ARANGO_ROOT_PASSWORD:-rootpw}"

# usage prints command-line help.
usage() {
	cat <<EOF
Usage:
  $0 run <test-command> [args...]
  $0 run-host <test-command> [args...]
  $0 start
  $0 setup-kind
  $0 endpoint
  $0 ingress-address
  $0 cleanup
  $0 cleanup-kind

Environment:
  KUBE_ARANGODB_VERSION  kube-arangodb release to install (default: ${KUBE_ARANGODB_VERSION})
  KUBE_ARANGODB_IMAGE    kube-arangodb operator image (default: ${KUBE_ARANGODB_IMAGE})
  K8S_NAMESPACE          namespace for the ArangoDeployment (default: ${K8S_NAMESPACE})
  K8S_DEPLOYMENT         ArangoDeployment name (default: ${K8S_DEPLOYMENT})
  K8S_MODE               ArangoDeployment mode: Cluster or Single (default: ${K8S_MODE})
  K8S_AUTHENTICATION     enable ArangoDB authentication in Kubernetes (default: ${K8S_AUTHENTICATION})
  K8S_TEST_AUTHENTICATION driver auth mode: basic, jwt, or none (default: ${K8S_TEST_AUTHENTICATION})
  K8S_TLS                enable TLS in the ArangoDeployment (default: ${K8S_TLS})
  ARANGODB               ArangoDB image used by kube-arangodb (default: ${ARANGODB})
  ARANGO_ROOT_PASSWORD   root password configured for driver tests (default: ${ARANGO_ROOT_PASSWORD})
  ARANGO_LICENSE_KEY     optional Enterprise license key, stored in a Kubernetes secret
  K8S_INGRESS_HOST       host name used by ingress mode (default: ${K8S_INGRESS_HOST})
  K8S_INGRESS_ADDRESS    ingress IP for Docker host mapping; uses Ingress status when empty
  K8S_STUCK_INIT_TIMEOUT delete pods stuck in init-lifecycle longer than this (default: ${K8S_STUCK_INIT_TIMEOUT})
  K8S_KEEP_DEPLOYMENT    keep deployment after "run" (default: ${K8S_KEEP_DEPLOYMENT})
  K8S_DELETE_NAMESPACE   delete K8S_NAMESPACE during cleanup (default: ${K8S_DELETE_NAMESPACE})
  K8S_TEST_WORKDIR       working directory for the test command (default: ${K8S_TEST_WORKDIR})
  K8S_KIND_CLUSTER_NAME  kind cluster name for "setup-kind" (default: ${K8S_KIND_CLUSTER_NAME})
  K8S_KIND_NODE_IMAGE    kind node image for "setup-kind" (default: ${K8S_KIND_NODE_IMAGE})
  K8S_KIND_RETAIN        retain kind nodes after setup failure (default: ${K8S_KIND_RETAIN})
  K8S_DELETE_KIND_CLUSTER delete the kind cluster after "run" (default: ${K8S_DELETE_KIND_CLUSTER})
  K8S_INGRESS_NGINX_VERSION ingress-nginx release for "setup-kind" (default: ${K8S_INGRESS_NGINX_VERSION})
EOF
}

# require_tool exits if the named binary is not on PATH.
require_tool() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "ERROR: required tool '$1' was not found in PATH" >&2
		exit 1
	fi
}

# setup_kind creates (or reuses) a kind cluster and installs ingress-nginx.
# Maps host ports 80/443 to the ingress controller for local testing.
setup_kind() {
	local clusters
	local -a kind_args

	require_tool kind
	require_tool kubectl

	kind_args=(create cluster --name "${K8S_KIND_CLUSTER_NAME}" --image "${K8S_KIND_NODE_IMAGE}")
	if [ "${K8S_KIND_RETAIN}" = "true" ]; then
		kind_args+=(--retain)
	fi

	clusters="$(kind get clusters 2>/dev/null || true)"
	if [[ $'\n'"${clusters}"$'\n' == *$'\n'"${K8S_KIND_CLUSTER_NAME}"$'\n'* ]]; then
		echo "kind cluster ${K8S_KIND_CLUSTER_NAME} already exists."
		kubectl config use-context "kind-${K8S_KIND_CLUSTER_NAME}"
	else
		echo "Creating kind cluster ${K8S_KIND_CLUSTER_NAME}..."
		cat <<EOF | kind "${kind_args[@]}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
  - |-
    [plugins."io.containerd.grpc.v1.cri".containerd]
      snapshotter = "native"
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
    extraPortMappings:
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP
EOF
	fi

	kubectl apply -f "${K8S_INGRESS_NGINX_MANIFEST}"
	kubectl -n ingress-nginx rollout status deployment/ingress-nginx-controller --timeout=6m
	echo "kind is ready. Run tests with K8S_INGRESS_ADDRESS=127.0.0.1."
}

# cleanup_kind deletes the kind cluster named K8S_KIND_CLUSTER_NAME.
cleanup_kind() {
	require_tool kind
	echo "Deleting kind cluster ${K8S_KIND_CLUSTER_NAME}..."
	kind delete cluster --name "${K8S_KIND_CLUSTER_NAME}"
}

# dump_operator_diagnostics prints operator deployment state on rollout failures.
dump_operator_diagnostics() {
	echo "=== kube-arangodb operator diagnostics ===" >&2
	kubectl -n default get deployment arango-deployment-operator -o wide || true
	kubectl -n default describe deployment arango-deployment-operator || true
	kubectl -n default get replicasets,pods -o wide || true
	kubectl -n default get events --sort-by=.lastTimestamp | tail -n 30 || true

	for pod in $(kubectl -n default get pods -l app.kubernetes.io/name=kube-arangodb,release=deployment -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null); do
		echo "=== describe pod/${pod} ===" >&2
		kubectl -n default describe pod "${pod}" || true
		echo "=== logs pod/${pod} ===" >&2
		kubectl -n default logs "${pod}" --all-containers=true --tail=100 || true
		echo "=== previous logs pod/${pod} ===" >&2
		kubectl -n default logs "${pod}" --all-containers=true --previous --tail=100 || true
	done
}

# wait_for_operator_rollout blocks until arango-deployment-operator is ready.
wait_for_operator_rollout() {
	echo "Waiting for kube-arangodb operator deployment to become ready..."
	local deadline=$((SECONDS + $(duration_to_seconds "${K8S_WAIT_TIMEOUT}")))
	while [ "${SECONDS}" -lt "${deadline}" ]; do
		if kubectl -n default rollout status deployment/arango-deployment-operator --timeout=30s; then
			return
		fi
		dump_operator_diagnostics
		sleep 10
	done

	echo "ERROR: kube-arangodb operator did not become ready before timeout" >&2
	dump_operator_diagnostics
	exit 1
}

# install_operator applies kube-arangodb CRDs and the operator deployment.
install_operator() {
	if [ "${K8S_INSTALL_OPERATOR}" != "true" ]; then
		return
	fi
	if [ "${K8S_NAMESPACE}" != "default" ]; then
		echo "ERROR: K8S_INSTALL_OPERATOR=true installs kube-arangodb in the default namespace." >&2
		echo "Use K8S_NAMESPACE=default, or preinstall an operator for ${K8S_NAMESPACE} and set K8S_INSTALL_OPERATOR=false." >&2
		exit 1
	fi

	echo "Installing kube-arangodb ${KUBE_ARANGODB_VERSION} operator..."
	kubectl apply -f "https://raw.githubusercontent.com/arangodb/kube-arangodb/${KUBE_ARANGODB_VERSION}/manifests/arango-crd.yaml"
	kubectl apply -f "https://raw.githubusercontent.com/arangodb/kube-arangodb/${KUBE_ARANGODB_VERSION}/manifests/arango-deployment.yaml"
	kubectl -n default set image deployment/arango-deployment-operator operator="${KUBE_ARANGODB_IMAGE}"
	wait_for_operator_rollout
}

# apply_deployment creates secrets and the ArangoDeployment custom resource.
apply_deployment() {
	echo "Creating ArangoDeployment ${K8S_NAMESPACE}/${K8S_DEPLOYMENT} (${K8S_MODE})..."
	if [ "${K8S_NAMESPACE}" != "default" ]; then
		kubectl create namespace "${K8S_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
	fi
	kubectl -n "${K8S_NAMESPACE}" create secret generic "${K8S_DEPLOYMENT}-root-password" \
		--from-literal=username=root \
		--from-literal=password="${ARANGO_ROOT_PASSWORD}" \
		--dry-run=client -o yaml | kubectl apply -f -
	# Let kube-arangodb generate fresh JWT secrets for each test deployment.
	# This avoids stale operator-managed secrets from previous runs.
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-jwt" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-jwt-folder" --ignore-not-found=true
	if [ -n "${ARANGO_LICENSE_KEY:-}" ]; then
		kubectl -n "${K8S_NAMESPACE}" create secret generic "${K8S_DEPLOYMENT}-license" \
			--from-literal=token-v2="${ARANGO_LICENSE_KEY}" \
			--dry-run=client -o yaml | kubectl apply -f -
	fi

	cat <<EOF | kubectl apply -f -
apiVersion: database.arangodb.com/v1
kind: ArangoDeployment
metadata:
  name: ${K8S_DEPLOYMENT}
  namespace: ${K8S_NAMESPACE}
spec:
  mode: ${K8S_MODE}
  environment: ${K8S_ENVIRONMENT}
  image: ${ARANGODB}
  imageDiscoveryMode: direct
$(render_auth_spec)
$(render_tls_spec)
  externalAccess:
    type: ${K8S_EXTERNAL_ACCESS}
  bootstrap:
    passwordSecretNames:
      root: ${K8S_DEPLOYMENT}-root-password
$(if [ -n "${ARANGO_LICENSE_KEY:-}" ]; then cat <<EOF_LICENSE
  license:
    secretName: ${K8S_DEPLOYMENT}-license
EOF_LICENSE
fi)
$(if [ "${K8S_MODE}" = "Cluster" ]; then cat <<EOF_CLUSTER
  agents:
    count: 1
$(render_arangod_args "    ")
  dbservers:
    count: 3
$(render_arangod_args "    ")
  coordinators:
    count: 1
$(render_arangod_args "    ")
EOF_CLUSTER
else cat <<EOF_SINGLE
  single:
$(render_arangod_args "    ")
EOF_SINGLE
fi)
EOF
}

# render_auth_spec emits the auth section for the ArangoDeployment manifest.
render_auth_spec() {
	if [ "${K8S_AUTHENTICATION}" = "true" ]; then
		cat <<EOF
  auth:
    jwtSecretName: ${K8S_DEPLOYMENT}-jwt
EOF
	else
		cat <<EOF
  auth:
    jwtSecretName: None
EOF
	fi
}

# render_tls_spec emits the TLS section for the ArangoDeployment manifest.
render_tls_spec() {
	if [ "${K8S_TLS}" = "true" ]; then
		cat <<EOF
  tls:
    caSecretName: ${K8S_DEPLOYMENT}-ca
EOF
	else
		cat <<EOF
  tls:
    caSecretName: None
EOF
	fi
}

# render_arangod_args emits common arangod CLI args for all pod roles.
render_arangod_args() {
	local indent="$1"

	cat <<EOF
${indent}args:
${indent}  - --javascript.startup-options-allowlist=.*
EOF

	if [ "${ENABLE_DATABASE_EXTRA_FEATURES:-}" = "true" ]; then
		cat <<EOF
${indent}  - --database.extended-names-databases=true
${indent}  - --http.compress-response-threshold=1
${indent}  - --http.handle-content-encoding-for-unauthenticated-requests=true
EOF
	fi

	if [ "${ENABLE_VECTOR_INDEX:-}" = "true" ]; then
		cat <<EOF
${indent}  - --vector-index=true
${indent}  - --experimental-vector-index=true
EOF
	fi
}

# wait_for_jwt_secret_folder waits until kube-arangodb populates the JWT secret folder.
wait_for_jwt_secret_folder() {
	if [ "${K8S_AUTHENTICATION}" != "true" ]; then
		echo "Skipping JWT folder wait because Kubernetes authentication is disabled."
		return
	fi

	echo "Waiting for kube-arangodb to populate ${K8S_DEPLOYMENT}-jwt-folder..."
	local deadline=$((SECONDS + $(duration_to_seconds "${K8S_WAIT_TIMEOUT}")))
	local jwt_data
	while [ "${SECONDS}" -lt "${deadline}" ]; do
		jwt_data="$(kubectl -n "${K8S_NAMESPACE}" get secret "${K8S_DEPLOYMENT}-jwt-folder" -o jsonpath='{.data}' 2>/dev/null || true)"
		if [ -n "${jwt_data}" ] && [ "${jwt_data}" != "map[]" ]; then
			delete_failed_pods
			return
		fi
		sleep 2
	done

	echo "ERROR: ${K8S_DEPLOYMENT}-jwt-folder was not populated before timeout" >&2
	dump_diagnostics
	exit 1
}

# wait_for_deployment blocks until the ArangoDeployment is Ready and the -ea service has endpoints.
wait_for_deployment() {
	echo "Waiting for ArangoDeployment ${K8S_NAMESPACE}/${K8S_DEPLOYMENT} to become ready..."
	local deadline=$((SECONDS + $(duration_to_seconds "${K8S_WAIT_TIMEOUT}")))
	local ready_status
	while [ "${SECONDS}" -lt "${deadline}" ]; do
		ready_status="$(kubectl -n "${K8S_NAMESPACE}" get arangodeployment "${K8S_DEPLOYMENT}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)"
		if [ "${ready_status}" = "True" ]; then
			break
		fi
		kubectl -n "${K8S_NAMESPACE}" get pods -l "arango_deployment=${K8S_DEPLOYMENT}" || true
		delete_stuck_init_pods
		if has_failed_pods; then
			echo "ERROR: one or more ArangoDB pods failed while waiting for deployment readiness" >&2
			dump_diagnostics
			exit 1
		fi
		sleep 10
	done
	if [ "${ready_status}" != "True" ]; then
		echo "ERROR: ArangoDeployment ${K8S_NAMESPACE}/${K8S_DEPLOYMENT} did not become ready before timeout" >&2
		dump_diagnostics
		exit 1
	fi
	if ! kubectl -n "${K8S_NAMESPACE}" wait "service/${K8S_DEPLOYMENT}-ea" \
		--for=jsonpath='{.spec.ports[0].port}'="${K8S_PORT}" \
		--timeout=2m; then
		dump_diagnostics
		exit 1
	fi

	echo "Waiting for service/${K8S_DEPLOYMENT}-ea to have ready endpoints..."
	local deadline=$((SECONDS + $(duration_to_seconds "${K8S_WAIT_TIMEOUT}")))
	while [ "${SECONDS}" -lt "${deadline}" ]; do
		if [ -n "$(kubectl -n "${K8S_NAMESPACE}" get endpoints "${K8S_DEPLOYMENT}-ea" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null)" ]; then
			return
		fi
		kubectl -n "${K8S_NAMESPACE}" get pods -l "arango_deployment=${K8S_DEPLOYMENT}" || true
		delete_stuck_init_pods
		if has_failed_pods; then
			echo "ERROR: one or more ArangoDB pods failed while waiting for ready endpoints" >&2
			dump_diagnostics
			exit 1
		fi
		sleep 10
	done

	echo "ERROR: service/${K8S_DEPLOYMENT}-ea did not get ready endpoints before timeout" >&2
	dump_diagnostics
	exit 1
}

# has_failed_pods returns 0 when any ArangoDB pod is in a permanent failure state.
has_failed_pods() {
	local pod_states
	pod_states="$(kubectl -n "${K8S_NAMESPACE}" get pods -l "arango_deployment=${K8S_DEPLOYMENT}" -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.phase}{" "}{range .status.initContainerStatuses[*]}{.state.waiting.reason}{","}{.state.terminated.reason}{","}{end}{range .status.containerStatuses[*]}{.state.waiting.reason}{","}{.state.terminated.reason}{","}{end}{"\n"}{end}' 2>/dev/null || true)"

	case "${pod_states}" in
		*Error*|*CrashLoopBackOff*|*ImagePullBackOff*|*ErrImagePull*|*CreateContainerConfigError*|*InvalidImageName*)
			echo "${pod_states}" >&2
			return 0
			;;
	esac

	return 1
}

# delete_failed_pods removes pods in Failed phase so the operator can recreate them.
delete_failed_pods() {
	local failed_pods
	failed_pods="$(kubectl -n "${K8S_NAMESPACE}" get pods -l "arango_deployment=${K8S_DEPLOYMENT}" --field-selector=status.phase=Failed -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)"
	if [ -n "${failed_pods}" ]; then
		echo "Deleting pods that failed before the JWT folder was populated..."
		for pod in ${failed_pods}; do
			kubectl -n "${K8S_NAMESPACE}" delete pod "${pod}" --ignore-not-found=true
		done
	fi
}

# delete_stuck_init_pods removes pods stuck in init-lifecycle longer than K8S_STUCK_INIT_TIMEOUT.
delete_stuck_init_pods() {
	local timeout_seconds now pod started_at started_at_seconds age
	timeout_seconds="$(duration_to_seconds "${K8S_STUCK_INIT_TIMEOUT}")"
	now="$(date +%s)"

	while read -r pod started_at; do
		if [ -z "${pod}" ] || [ -z "${started_at}" ]; then
			continue
		fi

		started_at_seconds="$(date -d "${started_at}" +%s 2>/dev/null || true)"
		if [ -z "${started_at_seconds}" ]; then
			continue
		fi

		age=$((now - started_at_seconds))
		if [ "${age}" -gt "${timeout_seconds}" ]; then
			echo "Deleting pod/${pod} because init-lifecycle has been running for ${age}s."
			kubectl -n "${K8S_NAMESPACE}" delete pod "${pod}" --ignore-not-found=true
		fi
	done < <(kubectl -n "${K8S_NAMESPACE}" get pods \
		-l "arango_deployment=${K8S_DEPLOYMENT}" \
		--field-selector=status.phase=Pending \
		-o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.initContainerStatuses[?(@.name=="init-lifecycle")].state.running.startedAt}{"\n"}{end}' 2>/dev/null || true)
}

# dump_diagnostics prints ArangoDeployment, pod, and log details on test setup failures.
dump_diagnostics() {
	echo "=== Kubernetes diagnostics for ${K8S_NAMESPACE}/${K8S_DEPLOYMENT} ===" >&2
	kubectl -n "${K8S_NAMESPACE}" get arangodeployment "${K8S_DEPLOYMENT}" -o wide || true
	kubectl -n "${K8S_NAMESPACE}" describe arangodeployment "${K8S_DEPLOYMENT}" || true
	kubectl -n "${K8S_NAMESPACE}" get pods,svc,endpoints -l "arango_deployment=${K8S_DEPLOYMENT}" -o wide || true
	kubectl -n "${K8S_NAMESPACE}" get events --sort-by=.lastTimestamp || true

	for pod in $(kubectl -n "${K8S_NAMESPACE}" get pods -l "arango_deployment=${K8S_DEPLOYMENT}" -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null); do
		echo "=== describe pod/${pod} ===" >&2
		kubectl -n "${K8S_NAMESPACE}" describe pod "${pod}" || true

		for container in $(kubectl -n "${K8S_NAMESPACE}" get pod "${pod}" -o jsonpath='{range .status.initContainerStatuses[*]}{.name}{"\n"}{end}{range .status.containerStatuses[*]}{.name}{"\n"}{end}' 2>/dev/null); do
			echo "=== logs pod/${pod} container/${container} ===" >&2
			kubectl -n "${K8S_NAMESPACE}" logs "${pod}" -c "${container}" --tail=100 || true
			echo "=== previous logs pod/${pod} container/${container} ===" >&2
			kubectl -n "${K8S_NAMESPACE}" logs "${pod}" -c "${container}" --previous --tail=100 || true
		done
	done
}

# duration_to_seconds converts values like 15m or 30s to integer seconds.
duration_to_seconds() {
	case "$1" in
		*m) echo $((${1%m} * 60)) ;;
		*s) echo "${1%s}" ;;
		*) echo "$1" ;;
	esac
}

# start performs the full Kubernetes setup: operator, deployment, JWT wait, readiness wait.
start() {
	require_tool kubectl
	install_operator
	apply_deployment
	wait_for_jwt_secret_folder
	wait_for_deployment
}

# ingress_endpoint returns the driver URL using the ingress hostname (e.g. https://arangodb.local).
ingress_endpoint() {
	local scheme
	if [ "${K8S_INGRESS_TLS}" = "true" ]; then
		scheme="https"
	else
		scheme="http"
	fi

	if [ -n "${K8S_INGRESS_PORT}" ]; then
		echo "${scheme}://${K8S_INGRESS_HOST}:${K8S_INGRESS_PORT}"
	else
		echo "${scheme}://${K8S_INGRESS_HOST}"
	fi
}

# ingress_host_endpoint returns the driver URL using the ingress IP (e.g. https://127.0.0.1).
# Used by run-host; tests send Host: K8S_INGRESS_HOST so ingress routing still works.
ingress_host_endpoint() {
	local scheme address
	if [ "${K8S_INGRESS_TLS}" = "true" ]; then
		scheme="https"
	else
		scheme="http"
	fi

	address="$(ingress_address)"
	if [ -z "${address}" ]; then
		ingress_endpoint
		return
	fi

	if [ -n "${K8S_INGRESS_PORT}" ]; then
		echo "${scheme}://${address}:${K8S_INGRESS_PORT}"
	else
		echo "${scheme}://${address}"
	fi
}

# endpoint_scheme returns http or https based on ArangoDeployment TLS (K8S_TLS).
endpoint_scheme() {
	if [ "${K8S_TLS}" = "true" ]; then
		echo "https"
	else
		echo "http"
	fi
}

# test_authentication formats the driver auth string passed to go test (basic/jwt/none).
test_authentication() {
	if [ "${K8S_AUTHENTICATION}" != "true" ]; then
		echo ""
		return
	fi

	case "${K8S_TEST_AUTHENTICATION}" in
		basic)
			echo "basic:root:${ARANGO_ROOT_PASSWORD}"
			;;
		jwt)
			echo "jwt:root:${ARANGO_ROOT_PASSWORD}"
			;;
		none)
			echo ""
			;;
		*)
			echo "ERROR: unsupported K8S_TEST_AUTHENTICATION '${K8S_TEST_AUTHENTICATION}'" >&2
			exit 1
			;;
	esac
}

# ingress_address returns the IP used to reach ingress (K8S_INGRESS_ADDRESS or LB status).
ingress_address() {
	if [ -n "${K8S_INGRESS_ADDRESS}" ]; then
		echo "${K8S_INGRESS_ADDRESS}"
		return
	fi

	kubectl -n "${K8S_NAMESPACE}" get ingress "${K8S_INGRESS_NAME}" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true
}

# ingress_annotations emits nginx timeout annotations for the Ingress manifest.
ingress_annotations() {
	cat <<EOF
  annotations:
    nginx.ingress.kubernetes.io/proxy-connect-timeout: "300"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "300"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "300"
EOF

	if [ "${K8S_TLS}" = "true" ]; then
		cat <<EOF
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
EOF
	fi
}

# ingress_tls_spec emits the TLS section of the Ingress when K8S_INGRESS_TLS=true.
ingress_tls_spec() {
	if [ "${K8S_INGRESS_TLS}" = "true" ]; then
		cat <<EOF
  tls:
    - hosts:
        - ${K8S_INGRESS_HOST}
      secretName: ${K8S_INGRESS_TLS_SECRET}
EOF
	fi
}

# create_ingress_tls_secret creates a self-signed TLS secret for K8S_INGRESS_HOST.
create_ingress_tls_secret() {
	if [ "${K8S_INGRESS_TLS}" != "true" ]; then
		return
	fi

	require_tool openssl
	local cert_dir
	cert_dir="$(mktemp -d)"
	openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
		-keyout "${cert_dir}/tls.key" \
		-out "${cert_dir}/tls.crt" \
		-subj "/CN=${K8S_INGRESS_HOST}/O=${K8S_INGRESS_HOST}" -addext "subjectAltName=DNS:${K8S_INGRESS_HOST}" >/dev/null 2>&1

	kubectl -n "${K8S_NAMESPACE}" create secret tls "${K8S_INGRESS_TLS_SECRET}" \
		--key "${cert_dir}/tls.key" \
		--cert "${cert_dir}/tls.crt" \
		--dry-run=client -o yaml | kubectl apply -f -
	rm -rf "${cert_dir}"
}

# setup_ingress creates or updates the Ingress pointing at the ArangoDB external access service.
setup_ingress() {
	echo "Creating Ingress ${K8S_NAMESPACE}/${K8S_INGRESS_NAME} for ${K8S_INGRESS_HOST}..."
	create_ingress_tls_secret

	cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ${K8S_INGRESS_NAME}
  namespace: ${K8S_NAMESPACE}
$(ingress_annotations)
spec:
  ingressClassName: ${K8S_INGRESS_CLASS}
$(ingress_tls_spec)
  rules:
    - host: ${K8S_INGRESS_HOST}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: ${K8S_DEPLOYMENT}-ea
                port:
                  number: ${K8S_PORT}
EOF
}

# cleanup deletes the ArangoDeployment, Ingress, and related secrets.
cleanup() {
	require_tool kubectl
	echo "Cleaning up ArangoDeployment ${K8S_NAMESPACE}/${K8S_DEPLOYMENT}..."
	kubectl -n "${K8S_NAMESPACE}" delete ingress "${K8S_INGRESS_NAME}" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete arangodeployment "${K8S_DEPLOYMENT}" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-root-password" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-jwt" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-jwt-folder" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-ca" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_INGRESS_TLS_SECRET}" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-license" --ignore-not-found=true
	if [ "${K8S_DELETE_NAMESPACE}" = "true" ] && [ "${K8S_NAMESPACE}" != "default" ]; then
		kubectl delete namespace "${K8S_NAMESPACE}" --ignore-not-found=true
	fi
}

# run_tests deploys the cluster and runs the test command for Docker-based integration tests.
run_tests() {
	if [ "$#" -lt 1 ]; then
		usage
		exit 1
	fi

	require_tool kubectl

	trap cleanup_after_run EXIT
	start

	run_command_through_ingress "$@"
}

# run_host_tests deploys the cluster and runs the test command on the host (not in Docker).
# Use when tests need kubectl access (e.g. ingress restart resiliency scenarios).
run_host_tests() {
	if [ "$#" -lt 1 ]; then
		usage
		exit 1
	fi

	require_tool kubectl

	trap cleanup_after_run EXIT
	start

	run_command_on_host_through_ingress "$@"
}

# get_ingress_address ensures the Ingress exists and prints the IP to reach it.
get_ingress_address() {
	local address
	setup_ingress >&2
	address="$(ingress_address)"
	if [ -z "${address}" ]; then
		echo "ERROR: unable to determine ingress address. Set K8S_INGRESS_ADDRESS explicitly." >&2
		exit 1
	fi
	echo "${address}"
}

# endpoint prints the ingress hostname URL (used by the "endpoint" CLI command).
endpoint() {
	ingress_endpoint
}

# collect_ingress_test_env prints env lines for Docker-based tests (run).
# Sets TEST_ENDPOINTS_OVERRIDE to the hostname URL and TEST_NET_OVERRIDE for --add-host.
collect_ingress_test_env() {
	local address auth endpoint net_override
	address="$(get_ingress_address)"
	endpoint="$(ingress_endpoint)"
	auth="$(test_authentication)"
	net_override="--net=host --add-host=${K8S_INGRESS_HOST}:${address}"

	if [ -n "${K8S_TEST_ENDPOINTS_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_ENDPOINTS_ENV}=${endpoint}"
	fi
	if [ -n "${K8S_TEST_AUTHENTICATION_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_AUTHENTICATION_ENV}=${auth}"
	fi
	if [ -n "${K8S_TEST_LEGACY_AUTHENTICATION_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_LEGACY_AUTHENTICATION_ENV}=${auth}"
	fi
	if [ -n "${K8S_TEST_MODE_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_MODE_ENV}=k8s"
	fi
	if [ -n "${K8S_TEST_NOT_WAIT_UNTIL_READY_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_NOT_WAIT_UNTIL_READY_ENV}=1"
	fi
	if [ -n "${K8S_TEST_NET_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_NET_ENV}=${net_override}"
	fi
}

# run_command_through_ingress runs the caller command with Docker test env (normal k8s path).
run_command_through_ingress() {
	local address endpoint
	local -a test_env
	mapfile -t test_env < <(collect_ingress_test_env)
	address="$(get_ingress_address)"
	endpoint="$(ingress_endpoint)"

	echo "Running test command against ${endpoint} through ingress ${address}..."
	(
		cd "${K8S_TEST_WORKDIR}"
		env "${test_env[@]}" "$@"
	)
}

# collect_ingress_host_test_env prints env lines for host-based tests (run-host).
# Uses the ingress IP endpoint (e.g. http://127.0.0.1) instead of the hostname.
# run_command_on_host_through_ingress also sets TEST_ENDPOINTS, TEST_ENDPOINTS_OVERRIDE,
# and TEST_INGRESS_HOST so go test connects to the IP but sends Host: arangodb.local.
collect_ingress_host_test_env() {
	local address auth endpoint
	address="$(get_ingress_address)"
	endpoint="$(ingress_host_endpoint)"
	auth="$(test_authentication)"

	if [ -n "${K8S_TEST_ENDPOINTS_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_ENDPOINTS_ENV}=${endpoint}"
	fi
	if [ -n "${K8S_TEST_AUTHENTICATION_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_AUTHENTICATION_ENV}=${auth}"
	fi
	if [ -n "${K8S_TEST_LEGACY_AUTHENTICATION_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_LEGACY_AUTHENTICATION_ENV}=${auth}"
	fi
	if [ -n "${K8S_TEST_MODE_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_MODE_ENV}=k8s"
	fi
	if [ -n "${K8S_TEST_NOT_WAIT_UNTIL_READY_ENV}" ]; then
		printf '%s\n' "${K8S_TEST_NOT_WAIT_UNTIL_READY_ENV}=1"
	fi
}

# run_command_on_host_through_ingress runs go test on the host with IP + Host header env.
run_command_on_host_through_ingress() {
	local address endpoint
	local -a test_env
	mapfile -t test_env < <(collect_ingress_host_test_env)
	address="$(get_ingress_address)"
	endpoint="$(ingress_host_endpoint)"

	echo "Running host test command against ${endpoint} (ingress host ${K8S_INGRESS_HOST}) through ${address}..."
	(
		cd "${K8S_TEST_WORKDIR}"
		env \
			"${test_env[@]}" \
			TEST_ENDPOINTS="${endpoint}" \
			TEST_ENDPOINTS_OVERRIDE="${endpoint}" \
			TEST_INGRESS_HOST="${K8S_INGRESS_HOST}" \
			TEST_AUTHENTICATION="$(test_authentication)" \
			TEST_MODE=cluster \
			"$@"
	)
}

# cleanup_after_run is the EXIT trap: optionally removes deployment and kind cluster.
cleanup_after_run() {
	if [ "${K8S_KEEP_DEPLOYMENT}" != "true" ]; then
		cleanup
	fi
	if [ "${K8S_DELETE_KIND_CLUSTER}" = "true" ]; then
		cleanup_kind
	fi
}

# Main command dispatcher — maps CLI subcommands to functions above.
case "${1:-}" in
	run)          # deploy cluster, run tests in Docker (normal k8s integration tests)
		shift
		run_tests "$@"
		;;
	run-host)     # deploy cluster, run tests on host (needs kubectl from tests)
		shift
		run_host_tests "$@"
		;;
	start)        # deploy cluster only, no test command
		start
		;;
	setup-kind)   # create local kind cluster with ingress-nginx
		setup_kind
		;;
	cleanup-kind) # delete the kind cluster
		cleanup_kind
		;;
	endpoint)     # print ingress hostname URL (e.g. https://arangodb.local)
		endpoint
		;;
	ingress-address) # print ingress IP (e.g. 127.0.0.1)
		get_ingress_address
		;;
	cleanup|stop) # remove ArangoDeployment, Ingress, and secrets
		cleanup
		;;
	-h|--help|help|"")
		usage
		;;
	*)
		echo "ERROR: unknown command '$1'" >&2
		usage
		exit 1
		;;
esac
