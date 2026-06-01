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
K8S_INGRESS_NGINX_MANIFEST="${K8S_INGRESS_NGINX_MANIFEST:-https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml}"

ARANGODB="${ARANGODB:-arangodb/enterprise-preview:latest}"
KUBE_ARANGODB_IMAGE="${KUBE_ARANGODB_IMAGE:-arangodb/kube-arangodb:${KUBE_ARANGODB_VERSION}}"
ARANGO_ROOT_PASSWORD="${ARANGO_ROOT_PASSWORD:-rootpw}"

usage() {
	cat <<EOF
Usage:
  $0 run <test-command> [args...]
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
		-subj "/CN=${K8S_INGRESS_HOST}/O=${K8S_INGRESS_HOST}" >/dev/null 2>&1

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
                name: ${K8S_DEPLOYMENT}
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
	if [ "${K8S_DELETE_NAMESPACE}" = "true" ] && [ "${K8S_NAMESPACE}" != "default" ]; then
		kubectl delete namespace "${K8S_NAMESPACE}" --ignore-not-found=true
	fi
}

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

	echo "Running test command against ${endpoint} through ingress ${address}..."
	(
		cd "${K8S_TEST_WORKDIR}"
		env "${test_env[@]}" "$@"
	)
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
