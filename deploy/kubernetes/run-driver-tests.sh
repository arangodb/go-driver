#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

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
K8S_TEST_ENDPOINTS_ENV="${K8S_TEST_ENDPOINTS_ENV:-TEST_ENDPOINTS_OVERRIDE}"
K8S_TEST_AUTHENTICATION_ENV="${K8S_TEST_AUTHENTICATION_ENV:-TEST_AUTHENTICATION_OVERRIDE}"
K8S_TEST_LEGACY_AUTHENTICATION_ENV="${K8S_TEST_LEGACY_AUTHENTICATION_ENV:-TEST_AUTHENTICATION}"
K8S_TEST_MODE_ENV="${K8S_TEST_MODE_ENV:-TEST_MODE_K8S}"
K8S_TEST_NOT_WAIT_UNTIL_READY_ENV="${K8S_TEST_NOT_WAIT_UNTIL_READY_ENV:-TEST_NOT_WAIT_UNTIL_READY}"
K8S_TEST_NET_ENV="${K8S_TEST_NET_ENV:-TEST_NET_OVERRIDE}"
K8S_KIND_CLUSTER_NAME="${K8S_KIND_CLUSTER_NAME:-arangodb-driver-tests}"
K8S_KIND_NODE_IMAGE="${K8S_KIND_NODE_IMAGE:-kindest/node:v1.31.12}"
K8S_KIND_RETAIN="${K8S_KIND_RETAIN:-false}"
K8S_DELETE_KIND_CLUSTER="${K8S_DELETE_KIND_CLUSTER:-false}"
K8S_COORDINATORS_COUNT="${K8S_COORDINATORS_COUNT:-1}"
K8S_INGRESS_NGINX_VERSION="${K8S_INGRESS_NGINX_VERSION:-controller-v1.12.1}"
K8S_INGRESS_NGINX_MANIFEST="${K8S_INGRESS_NGINX_MANIFEST:-https://raw.githubusercontent.com/kubernetes/ingress-nginx/${K8S_INGRESS_NGINX_VERSION}/deploy/static/provider/kind/deploy.yaml}"
K8S_INCLUSTER_JOB_NAME="${K8S_INCLUSTER_JOB_NAME:-${K8S_DEPLOYMENT}-resiliency-incluster}"
K8S_INCLUSTER_IMAGE="${K8S_INCLUSTER_IMAGE:-go-driver-resiliency-incluster:local}"
K8S_INCLUSTER_TEST_RUN="${K8S_INCLUSTER_TEST_RUN:-TestResiliency_}"
K8S_INCLUSTER_SERVICE_ACCOUNT="${K8S_INCLUSTER_SERVICE_ACCOUNT:-${K8S_DEPLOYMENT}-resiliency-incluster}"
K8S_KIND_REPO_MOUNT="${K8S_KIND_REPO_MOUNT:-/go-driver-src}"
GOVERSION="${GOVERSION:-1.25.11}"
GOV2IMAGE="${GOV2IMAGE:-golang:${GOVERSION}}"

ARANGODB="${ARANGODB:-arangodb/enterprise-preview:latest}"
KUBE_ARANGODB_IMAGE="${KUBE_ARANGODB_IMAGE:-arangodb/kube-arangodb:${KUBE_ARANGODB_VERSION}}"
ARANGO_ROOT_PASSWORD="${ARANGO_ROOT_PASSWORD:-rootpw}"

usage() {
	cat <<EOF
Usage:
  $0 run <test-command> [args...]
  $0 run-incluster
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
  K8S_COORDINATORS_COUNT number of coordinators in Cluster mode (default: ${K8S_COORDINATORS_COUNT})
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
  K8S_INCLUSTER_IMAGE    Docker image for in-cluster resiliency Job (default: ${K8S_INCLUSTER_IMAGE})
  K8S_INCLUSTER_TEST_RUN go test -run filter for run-incluster (default: ${K8S_INCLUSTER_TEST_RUN})
EOF
}

require_tool() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "ERROR: required tool '$1' was not found in PATH" >&2
		exit 1
	fi
}

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
		if ! docker exec "${K8S_KIND_CLUSTER_NAME}-control-plane" test -f "${K8S_KIND_REPO_MOUNT}/v2/go.mod" 2>/dev/null; then
			echo "NOTE: existing kind cluster has no repo mount at ${K8S_KIND_REPO_MOUNT}."
			echo "      Recreate the cluster for in-cluster resiliency tests: bash $0 cleanup-kind && bash $0 setup-kind"
		fi
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
    extraMounts:
      - hostPath: ${ROOT_DIR}
        containerPath: ${K8S_KIND_REPO_MOUNT}
EOF
	fi

	kubectl apply -f "${K8S_INGRESS_NGINX_MANIFEST}"
	kubectl -n ingress-nginx rollout status deployment/ingress-nginx-controller --timeout=6m
	echo "kind is ready. Run tests with K8S_INGRESS_ADDRESS=127.0.0.1."
}

cleanup_kind() {
	require_tool kind
	echo "Deleting kind cluster ${K8S_KIND_CLUSTER_NAME}..."
	kind delete cluster --name "${K8S_KIND_CLUSTER_NAME}"
}

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

apply_deployment() {
	echo "Creating ArangoDeployment ${K8S_NAMESPACE}/${K8S_DEPLOYMENT} (${K8S_MODE})..."
	if [ "${K8S_MODE}" = "Cluster" ]; then
		echo "Cluster topology: agents=1, dbservers=3, coordinators=${K8S_COORDINATORS_COUNT}"
	fi
	if kubectl -n "${K8S_NAMESPACE}" get arangodeployment "${K8S_DEPLOYMENT}" >/dev/null 2>&1; then
		echo "Removing existing ArangoDeployment ${K8S_NAMESPACE}/${K8S_DEPLOYMENT} before recreating..."
		kubectl -n "${K8S_NAMESPACE}" delete arangodeployment "${K8S_DEPLOYMENT}" --wait=true --timeout=10m
	fi
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
    count: ${K8S_COORDINATORS_COUNT}
$(render_arangod_args "    ")
EOF_CLUSTER
else cat <<EOF_SINGLE
  single:
$(render_arangod_args "    ")
EOF_SINGLE
fi)
EOF
}

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

wait_for_coordinator_pods() {
	if [ "${K8S_MODE}" != "Cluster" ]; then
		return
	fi

	echo "Waiting for ${K8S_COORDINATORS_COUNT} coordinator pod(s) to be running..."
	local deadline=$((SECONDS + $(duration_to_seconds "${K8S_WAIT_TIMEOUT}")))
	local running_count
	while [ "${SECONDS}" -lt "${deadline}" ]; do
		running_count="$(kubectl -n "${K8S_NAMESPACE}" get pods \
			-l "arango_deployment=${K8S_DEPLOYMENT}" \
			--field-selector=status.phase=Running \
			-o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -c '\-crdn\-' || true)"
		if [ "${running_count}" -ge "${K8S_COORDINATORS_COUNT}" ]; then
			echo "Found ${running_count} running coordinator pod(s)"
			return
		fi
		kubectl -n "${K8S_NAMESPACE}" get pods -l "arango_deployment=${K8S_DEPLOYMENT}" || true
		sleep 10
	done

	echo "ERROR: expected ${K8S_COORDINATORS_COUNT} running coordinator pods, found ${running_count}" >&2
	dump_diagnostics
	exit 1
}

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

duration_to_seconds() {
	case "$1" in
		*m) echo $((${1%m} * 60)) ;;
		*s) echo "${1%s}" ;;
		*) echo "$1" ;;
	esac
}

start() {
	require_tool kubectl
	install_operator
	apply_deployment
	wait_for_jwt_secret_folder
	wait_for_deployment
	wait_for_coordinator_pods
}

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

endpoint_scheme() {
	if [ "${K8S_TLS}" = "true" ]; then
		echo "https"
	else
		echo "http"
	fi
}

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

ingress_address() {
	if [ -n "${K8S_INGRESS_ADDRESS}" ]; then
		echo "${K8S_INGRESS_ADDRESS}"
		return
	fi

	kubectl -n "${K8S_NAMESPACE}" get ingress "${K8S_INGRESS_NAME}" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true
}

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
	kubectl -n "${K8S_NAMESPACE}" delete job "${K8S_INCLUSTER_JOB_NAME}" --ignore-not-found=true
	if [ "${K8S_DELETE_NAMESPACE}" = "true" ] && [ "${K8S_NAMESPACE}" != "default" ]; then
		kubectl delete namespace "${K8S_NAMESPACE}" --ignore-not-found=true
	fi
}

ensure_resiliency_coordinator_count() {
	if ! printf '%s' "$*" | grep -qi resiliency; then
		return
	fi

	if [ "${K8S_COORDINATORS_COUNT:-1}" -ge 3 ]; then
		return
	fi

	echo "NOTE: resiliency tests need at least 3 coordinators; raising K8S_COORDINATORS_COUNT from ${K8S_COORDINATORS_COUNT:-1} to 3"
	K8S_COORDINATORS_COUNT=3
	export K8S_COORDINATORS_COUNT
}

run_tests() {
	if [ "$#" -lt 1 ]; then
		usage
		exit 1
	fi

	require_tool kubectl
	ensure_resiliency_coordinator_count "$@"

	trap cleanup_after_run EXIT
	start

	run_command_through_ingress "$@"
}

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

endpoint() {
	ingress_endpoint
}

run_command_through_ingress() {
	local address auth endpoint net_override
	local -a test_env
	address="$(get_ingress_address)"
	endpoint="$(ingress_endpoint)"
	auth="$(test_authentication)"
	net_override="--net=host --add-host=${K8S_INGRESS_HOST}:${address}"
	test_env=()

	if [ -n "${K8S_TEST_ENDPOINTS_ENV}" ]; then
		test_env+=("${K8S_TEST_ENDPOINTS_ENV}=${endpoint}")
	fi
	if [ -n "${K8S_TEST_AUTHENTICATION_ENV}" ]; then
		test_env+=("${K8S_TEST_AUTHENTICATION_ENV}=${auth}")
	fi
	if [ -n "${K8S_TEST_LEGACY_AUTHENTICATION_ENV}" ]; then
		test_env+=("${K8S_TEST_LEGACY_AUTHENTICATION_ENV}=${auth}")
	fi
	if [ -n "${K8S_TEST_MODE_ENV}" ]; then
		test_env+=("${K8S_TEST_MODE_ENV}=k8s")
	fi
	if [ -n "${K8S_TEST_NOT_WAIT_UNTIL_READY_ENV}" ]; then
		test_env+=("${K8S_TEST_NOT_WAIT_UNTIL_READY_ENV}=1")
	fi
	if [ -n "${K8S_TEST_NET_ENV}" ]; then
		test_env+=("${K8S_TEST_NET_ENV}=${net_override}")
	fi
	test_env+=("K8S_NAMESPACE=${K8S_NAMESPACE}")
	test_env+=("K8S_DEPLOYMENT=${K8S_DEPLOYMENT}")
	test_env+=("K8S_COORDINATORS_COUNT=${K8S_COORDINATORS_COUNT}")

	echo "Running test command against ${endpoint} through ingress ${address}..."
	(
		cd "${K8S_TEST_WORKDIR}"
		env "${test_env[@]}" "$@"
	)
}

incluster_service_endpoint() {
	echo "http://${K8S_DEPLOYMENT}.${K8S_NAMESPACE}.svc.cluster.local:${K8S_PORT}"
}

ensure_coordinator_service_clientip_affinity() {
	local svc
	for svc in "${K8S_DEPLOYMENT}" "${K8S_DEPLOYMENT}-ea"; do
		if ! kubectl -n "${K8S_NAMESPACE}" get svc "${svc}" >/dev/null 2>&1; then
			continue
		fi
		echo "Setting sessionAffinity=ClientIP on service/${svc}..."
		kubectl -n "${K8S_NAMESPACE}" patch svc "${svc}" --type=merge -p \
			'{"spec":{"sessionAffinity":"ClientIP","sessionAffinityConfig":{"clientIP":{"timeoutSeconds":10800}}}}'
	done
}

ensure_incluster_rbac() {
	echo "Ensuring in-cluster resiliency RBAC (service account ${K8S_INCLUSTER_SERVICE_ACCOUNT})..."
	cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ${K8S_INCLUSTER_SERVICE_ACCOUNT}
  namespace: ${K8S_NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ${K8S_INCLUSTER_SERVICE_ACCOUNT}
  namespace: ${K8S_NAMESPACE}
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "delete"]
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ${K8S_INCLUSTER_SERVICE_ACCOUNT}
  namespace: ${K8S_NAMESPACE}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: ${K8S_INCLUSTER_SERVICE_ACCOUNT}
subjects:
  - kind: ServiceAccount
    name: ${K8S_INCLUSTER_SERVICE_ACCOUNT}
    namespace: ${K8S_NAMESPACE}
EOF
}

build_incluster_test_image() {
	require_tool docker
	echo "Building in-cluster resiliency test image ${K8S_INCLUSTER_IMAGE}..."
	docker build \
		--build-arg "GOVERSION=${GOVERSION}" \
		-t "${K8S_INCLUSTER_IMAGE}" \
		-f "${ROOT_DIR}/deploy/kubernetes/Dockerfile.resiliency-incluster" \
		"${ROOT_DIR}"

	if command -v kind >/dev/null 2>&1 && kubectl config current-context 2>/dev/null | grep -q '^kind-'; then
		echo "Loading ${K8S_INCLUSTER_IMAGE} into kind cluster ${K8S_KIND_CLUSTER_NAME}..."
		if ! kind load docker-image "${K8S_INCLUSTER_IMAGE}" --name "${K8S_KIND_CLUSTER_NAME}" 2>/dev/null; then
			echo "kind load failed (common with native snapshotter); importing via containerd ctr..."
			docker save "${K8S_INCLUSTER_IMAGE}" | docker exec -i "${K8S_KIND_CLUSTER_NAME}-control-plane" \
				ctr --namespace=k8s.io images import -
		fi
	fi
}

kind_node_has_repo_mount() {
	docker exec "${K8S_KIND_CLUSTER_NAME}-control-plane" test -f "${K8S_KIND_REPO_MOUNT}/v2/go.mod" 2>/dev/null
}

ensure_kind_repo_available() {
	if ! command -v docker >/dev/null 2>&1; then
		echo "ERROR: docker is required to stage sources on the kind node" >&2
		exit 1
	fi
	if ! kubectl config current-context 2>/dev/null | grep -q '^kind-'; then
		return
	fi
	if kind_node_has_repo_mount; then
		return
	fi

	echo "Copying repository into kind node at ${K8S_KIND_REPO_MOUNT} for in-cluster tests..."
	docker exec "${K8S_KIND_CLUSTER_NAME}-control-plane" mkdir -p "${K8S_KIND_REPO_MOUNT}"
	docker cp "${ROOT_DIR}/." "${K8S_KIND_CLUSTER_NAME}-control-plane:${K8S_KIND_REPO_MOUNT}/"
	if ! kind_node_has_repo_mount; then
		echo "ERROR: failed to stage ${ROOT_DIR} on kind node at ${K8S_KIND_REPO_MOUNT}" >&2
		exit 1
	fi
}

incluster_job_image() {
	if kubectl config current-context 2>/dev/null | grep -q '^kind-' && kind_node_has_repo_mount; then
		echo "${GOV2IMAGE}"
		return
	fi
	echo "${K8S_INCLUSTER_IMAGE}"
}

incluster_job_volumes_spec() {
	if kubectl config current-context 2>/dev/null | grep -q '^kind-' && kind_node_has_repo_mount; then
		cat <<EOF
      volumes:
        - name: repo
          hostPath:
            path: ${K8S_KIND_REPO_MOUNT}
            type: Directory
EOF
	else
		echo ""
	fi
}

incluster_job_volume_mounts_spec() {
	if kubectl config current-context 2>/dev/null | grep -q '^kind-' && kind_node_has_repo_mount; then
		cat <<EOF
          volumeMounts:
            - name: repo
              mountPath: /usr/code
              readOnly: true
EOF
	else
		echo ""
	fi
}

incluster_job_image_pull_policy() {
	if kubectl config current-context 2>/dev/null | grep -q '^kind-' && kind_node_has_repo_mount; then
		echo "IfNotPresent"
	else
		echo "Never"
	fi
}

use_kind_repo_mount() {
	kubectl config current-context 2>/dev/null | grep -q '^kind-' && kind_node_has_repo_mount
}

run_incluster_resiliency_job() {
	local endpoint auth testoptions job_image pull_policy
	endpoint="$(incluster_service_endpoint)"
	auth="$(test_authentication)"
	testoptions="-run ${K8S_INCLUSTER_TEST_RUN}"
	job_image="$(incluster_job_image)"
	pull_policy="$(incluster_job_image_pull_policy)"

	if use_kind_repo_mount; then
		echo "Using kind repo mount at ${K8S_KIND_REPO_MOUNT} with image ${job_image}"
	else
		echo "Using pre-built image ${job_image}"
		build_incluster_test_image
	fi

	echo "Running in-cluster resiliency test against ${endpoint} (TEST_MODE_K8S=k8s-incluster)..."

	kubectl -n "${K8S_NAMESPACE}" delete job "${K8S_INCLUSTER_JOB_NAME}" --ignore-not-found=true --wait=true

	cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: ${K8S_INCLUSTER_JOB_NAME}
  namespace: ${K8S_NAMESPACE}
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 600
  template:
    spec:
      serviceAccountName: ${K8S_INCLUSTER_SERVICE_ACCOUNT}
      restartPolicy: Never
$(incluster_job_volumes_spec)
      containers:
        - name: resiliency-test
          image: ${job_image}
          imagePullPolicy: ${pull_policy}
          workingDir: /usr/code/v2
          command:
            - sh
            - -c
            - |
              if ! command -v kubectl >/dev/null 2>&1; then
                apt-get update >/dev/null \
                  && apt-get install -y --no-install-recommends ca-certificates curl >/dev/null \
                  && curl -fsSL "https://dl.k8s.io/release/$(curl -fsSL https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
                    -o /usr/local/bin/kubectl \
                  && chmod +x /usr/local/bin/kubectl \
                  && rm -rf /var/lib/apt/lists/*
              fi
              go test -timeout 120m -tags "resiliency auth" -v -parallel 1 ./tests ${testoptions}
$(incluster_job_volume_mounts_spec)
          env:
            - name: TEST_ENDPOINTS
              value: "${endpoint}"
            - name: TEST_AUTH
              value: "${ARANGO_ROOT_PASSWORD}"
            - name: TEST_AUTHENTICATION
              value: "${auth}"
            - name: TEST_MODE
              value: cluster
            - name: TEST_MODE_K8S
              value: k8s-incluster
            - name: TEST_ENABLE_RESILIENCY
              value: "1"
            - name: TEST_NOT_WAIT_UNTIL_READY
              value: "1"
            - name: TEST_JWTSECRET
              value: testing
            - name: K8S_NAMESPACE
              value: "${K8S_NAMESPACE}"
            - name: K8S_DEPLOYMENT
              value: "${K8S_DEPLOYMENT}"
            - name: K8S_COORDINATORS_COUNT
              value: "${K8S_COORDINATORS_COUNT}"
            - name: TESTOPTIONS
              value: "${testoptions}"
            - name: CGO_ENABLED
              value: "0"
            - name: GOTOOLCHAIN
              value: auto
EOF

	kubectl -n "${K8S_NAMESPACE}" wait --for=condition=complete "job/${K8S_INCLUSTER_JOB_NAME}" --timeout="${K8S_WAIT_TIMEOUT}" &
	local wait_pid=$!
	echo "Waiting for in-cluster resiliency job (timeout ${K8S_WAIT_TIMEOUT}); streaming pod logs..."
	kubectl -n "${K8S_NAMESPACE}" logs -f "job/${K8S_INCLUSTER_JOB_NAME}" --pod-running-timeout=10m 2>/dev/null || true
	wait "${wait_pid}" || {
		echo "ERROR: in-cluster resiliency job failed" >&2
		kubectl -n "${K8S_NAMESPACE}" logs "job/${K8S_INCLUSTER_JOB_NAME}" || true
		kubectl -n "${K8S_NAMESPACE}" describe "job/${K8S_INCLUSTER_JOB_NAME}" || true
		exit 1
	}
	kubectl -n "${K8S_NAMESPACE}" logs "job/${K8S_INCLUSTER_JOB_NAME}"
}

run_incluster_tests() {
	require_tool kubectl
	if [ "${K8S_COORDINATORS_COUNT:-1}" -lt 3 ]; then
		echo "NOTE: in-cluster resiliency tests need at least 3 coordinators; raising K8S_COORDINATORS_COUNT to 3"
		K8S_COORDINATORS_COUNT=3
		export K8S_COORDINATORS_COUNT
	fi

	trap cleanup_after_run EXIT
	if [ "${K8S_SKIP_START:-false}" != "true" ]; then
		start
	fi
	ensure_coordinator_service_clientip_affinity
	ensure_incluster_rbac
	ensure_kind_repo_available
	run_incluster_resiliency_job
}

cleanup_after_run() {
	if [ "${K8S_KEEP_DEPLOYMENT}" != "true" ]; then
		cleanup
	fi
	if [ "${K8S_DELETE_KIND_CLUSTER}" = "true" ]; then
		cleanup_kind
	fi
}

case "${1:-}" in
	run)
		shift
		run_tests "$@"
		;;
	run-incluster)
		run_incluster_tests
		;;
	start)
		start
		;;
	setup-kind)
		setup_kind
		;;
	cleanup-kind)
		cleanup_kind
		;;
	endpoint)
		endpoint
		;;
	ingress-address)
		get_ingress_address
		;;
	cleanup|stop)
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
