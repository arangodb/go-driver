#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

KUBE_ARANGODB_VERSION="${KUBE_ARANGODB_VERSION:-1.4.3}"
K8S_NAMESPACE="${K8S_NAMESPACE:-default}"
K8S_DEPLOYMENT="${K8S_DEPLOYMENT:-go-driver-tests}"
K8S_MODE="${K8S_MODE:-Cluster}"
K8S_ENVIRONMENT="${K8S_ENVIRONMENT:-Development}"
K8S_EXTERNAL_ACCESS="${K8S_EXTERNAL_ACCESS:-NodePort}"
K8S_PORT="${K8S_PORT:-8529}"
K8S_LOCAL_PORT="${K8S_LOCAL_PORT:-18529}"
K8S_PORT_FORWARD_ADDRESS="${K8S_PORT_FORWARD_ADDRESS:-0.0.0.0}"
K8S_TEST_ENDPOINT_HOST="${K8S_TEST_ENDPOINT_HOST:-host.docker.internal}"
K8S_TEST_NET_OVERRIDE="${K8S_TEST_NET_OVERRIDE:---add-host=host.docker.internal:host-gateway}"
K8S_WAIT_TIMEOUT="${K8S_WAIT_TIMEOUT:-15m}"
K8S_STUCK_INIT_TIMEOUT="${K8S_STUCK_INIT_TIMEOUT:-5m}"
K8S_KEEP_DEPLOYMENT="${K8S_KEEP_DEPLOYMENT:-false}"
K8S_DELETE_NAMESPACE="${K8S_DELETE_NAMESPACE:-false}"
K8S_INSTALL_OPERATOR="${K8S_INSTALL_OPERATOR:-true}"
K8S_AUTHENTICATION="${K8S_AUTHENTICATION:-true}"
K8S_TEST_AUTHENTICATION="${K8S_TEST_AUTHENTICATION:-basic}"
K8S_TLS="${K8S_TLS:-false}"

ARANGODB="${ARANGODB:-arangodb/enterprise-preview:latest}"
ARANGO_ROOT_PASSWORD="${ARANGO_ROOT_PASSWORD:-rootpw}"
PORT_FORWARD_PID=""

usage() {
	cat <<EOF
Usage:
  $0 run <make-target> [make args...]
  $0 start
  $0 endpoint
  $0 cleanup

Environment:
  KUBE_ARANGODB_VERSION  kube-arangodb release to install (default: ${KUBE_ARANGODB_VERSION})
  K8S_NAMESPACE          namespace for the ArangoDeployment (default: ${K8S_NAMESPACE})
  K8S_DEPLOYMENT         ArangoDeployment name (default: ${K8S_DEPLOYMENT})
  K8S_MODE               ArangoDeployment mode: Cluster, Single, ActiveFailover (default: ${K8S_MODE})
  K8S_AUTHENTICATION     enable ArangoDB authentication in Kubernetes (default: ${K8S_AUTHENTICATION})
  K8S_TEST_AUTHENTICATION driver auth mode: basic, jwt, or none (default: ${K8S_TEST_AUTHENTICATION})
  K8S_TLS                enable TLS in the ArangoDeployment (default: ${K8S_TLS})
  ARANGODB               ArangoDB image used by kube-arangodb (default: ${ARANGODB})
  ARANGO_ROOT_PASSWORD   root password configured for driver tests (default: ${ARANGO_ROOT_PASSWORD})
  ARANGO_LICENSE_KEY     optional Enterprise license key, stored in a Kubernetes secret
  K8S_LOCAL_PORT         local port used for kubectl port-forward (default: ${K8S_LOCAL_PORT})
  K8S_TEST_ENDPOINT_HOST host name used by Dockerized tests (default: ${K8S_TEST_ENDPOINT_HOST})
  K8S_STUCK_INIT_TIMEOUT delete pods stuck in init-lifecycle longer than this (default: ${K8S_STUCK_INIT_TIMEOUT})
  K8S_KEEP_DEPLOYMENT    keep deployment after "run" (default: ${K8S_KEEP_DEPLOYMENT})
  K8S_DELETE_NAMESPACE   delete K8S_NAMESPACE during cleanup (default: ${K8S_DELETE_NAMESPACE})
EOF
}

require_tool() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "ERROR: required tool '$1' was not found in PATH" >&2
		exit 1
	fi
}

install_operator() {
	if [ "${K8S_INSTALL_OPERATOR}" != "true" ]; then
		return
	fi

	echo "Installing kube-arangodb ${KUBE_ARANGODB_VERSION} operator..."
	kubectl apply -f "https://raw.githubusercontent.com/arangodb/kube-arangodb/${KUBE_ARANGODB_VERSION}/manifests/arango-crd.yaml"
	kubectl apply -f "https://raw.githubusercontent.com/arangodb/kube-arangodb/${KUBE_ARANGODB_VERSION}/manifests/arango-deployment.yaml"
	kubectl -n default rollout status deployment/arango-deployment-operator --timeout="${K8S_WAIT_TIMEOUT}"
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
	# Let kube-arangodb generate the JWT secret. ArangoDB 4.0 needs the
	# operator-created format; a plain token secret leaves the mounted JWT
	# folder unusable and pods fail with "empty JWT secrets directory".
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

endpoint() {
	echo "$(endpoint_scheme)://127.0.0.1:${K8S_LOCAL_PORT}"
}

test_endpoint() {
	echo "$(endpoint_scheme)://${K8S_TEST_ENDPOINT_HOST}:${K8S_LOCAL_PORT}"
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

wait_for_local_port_forward() {
	echo "Waiting for local port-forward on 127.0.0.1:${K8S_LOCAL_PORT}..."
	local deadline=$((SECONDS + 60))
	while [ "${SECONDS}" -lt "${deadline}" ]; do
		if ! kill -0 "$1" >/dev/null 2>&1; then
			echo "ERROR: kubectl port-forward stopped. Logs:" >&2
			sed 's/^/  /' /tmp/go-driver-k8s-port-forward.log >&2 || true
			exit 1
		fi
		if (echo >"/dev/tcp/127.0.0.1/${K8S_LOCAL_PORT}") >/dev/null 2>&1; then
			return
		fi
		sleep 1
	done

	echo "ERROR: local port-forward did not become reachable. Logs:" >&2
	sed 's/^/  /' /tmp/go-driver-k8s-port-forward.log >&2 || true
	exit 1
}

cleanup() {
	require_tool kubectl
	echo "Cleaning up ArangoDeployment ${K8S_NAMESPACE}/${K8S_DEPLOYMENT}..."
	kubectl -n "${K8S_NAMESPACE}" delete arangodeployment "${K8S_DEPLOYMENT}" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-root-password" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-jwt" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-jwt-folder" --ignore-not-found=true
	kubectl -n "${K8S_NAMESPACE}" delete secret "${K8S_DEPLOYMENT}-ca" --ignore-not-found=true
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
	require_tool make

	start

	trap 'if [ -n "${PORT_FORWARD_PID}" ]; then kill "${PORT_FORWARD_PID}" >/dev/null 2>&1 || true; fi; if [ "${K8S_KEEP_DEPLOYMENT}" != "true" ]; then cleanup; fi' EXIT

	echo "Forwarding service/${K8S_DEPLOYMENT}-ea to ${K8S_PORT_FORWARD_ADDRESS}:${K8S_LOCAL_PORT}..."
	kubectl -n "${K8S_NAMESPACE}" port-forward --address "${K8S_PORT_FORWARD_ADDRESS}" "service/${K8S_DEPLOYMENT}-ea" "${K8S_LOCAL_PORT}:${K8S_PORT}" >/tmp/go-driver-k8s-port-forward.log 2>&1 &
	PORT_FORWARD_PID="$!"
	wait_for_local_port_forward "${PORT_FORWARD_PID}"

	echo "Running driver tests against $(test_endpoint)..."
	(
		cd "${ROOT_DIR}"
		TEST_ENDPOINTS_OVERRIDE="$(test_endpoint)" \
		TEST_AUTHENTICATION_OVERRIDE="$(test_authentication)" \
		TEST_MODE_K8S="k8s" \
		TEST_NOT_WAIT_UNTIL_READY="1" \
		TEST_NET_OVERRIDE="${K8S_TEST_NET_OVERRIDE}" \
		make TEST_AUTHENTICATION="$(test_authentication)" "$@"
	)
}

case "${1:-}" in
	run)
		shift
		run_tests "$@"
		;;
	start)
		start
		;;
	endpoint)
		endpoint
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
