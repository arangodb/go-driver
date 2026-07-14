#!/bin/bash

# Start or stop a Toxiproxy instance for driver resiliency / network-fault tests.
# The proxy listens on TOXIPROXY_LISTEN and forwards to TOXIPROXY_UPSTREAM.
#
# Networking:
# - Default: publish admin + listen ports to 127.0.0.1 (works on Docker Desktop/WSL
#   and Linux). Upstream 127.0.0.1/localhost is rewritten to host.docker.internal
#   so the container can reach kind ingress / local ArangoDB on the Docker host.
# - Optional: DOCKER_NETWORK=--net=host (CircleCI / native Linux Docker) keeps
#   host networking; upstream is used as-is.

if [ -z "$TESTCONTAINER" ]; then
    echo "TESTCONTAINER environment variable must be set"
    exit 1
fi

CMD=$1
TOXIPROXY_CONTAINER=${TESTCONTAINER}-toxiproxy
TOXIPROXY_IMAGE=${TOXIPROXY_IMAGE:-ghcr.io/shopify/toxiproxy:2.9.0}
TOXIPROXY_ADMIN_PORT=${TOXIPROXY_ADMIN_PORT:-8474}
TOXIPROXY_LISTEN_PORT=${TOXIPROXY_LISTEN_PORT:-17001}
TOXIPROXY_PROXY_NAME=${TOXIPROXY_PROXY_NAME:-arangodb}
TOXIPROXY_UPSTREAM=${TOXIPROXY_UPSTREAM:-127.0.0.1:7001}
# Empty DOCKER_NETWORK => published-port mode (Docker Desktop / WSL safe).
# Set DOCKER_NETWORK=--net=host for Linux CI when host networking is preferred.
DOCKER_NETWORK=${DOCKER_NETWORK:-}

use_host_network=false
case " ${DOCKER_NETWORK} " in
    *" --net=host "*|*" --network=host "*|*" --network host "*)
        use_host_network=true
        ;;
esac

if [ "${use_host_network}" = true ]; then
    TOXIPROXY_LISTEN=${TOXIPROXY_LISTEN:-127.0.0.1:${TOXIPROXY_LISTEN_PORT}}
else
    # Must bind all interfaces inside the container for -p publish to work.
    TOXIPROXY_LISTEN=${TOXIPROXY_LISTEN:-0.0.0.0:${TOXIPROXY_LISTEN_PORT}}
    case "${TOXIPROXY_UPSTREAM}" in
        127.0.0.1:*|localhost:*)
            TOXIPROXY_UPSTREAM="host.docker.internal:${TOXIPROXY_UPSTREAM#*:}"
            ;;
    esac
fi

docker rm -f "${TOXIPROXY_CONTAINER}" &> /dev/null || true

# Prefer host curl once. Do not probe with a throwaway request first — non-idempotent
# calls (POST /proxies) would create the resource and then fail on a second attempt.
toxiproxy_curl() {
    if command -v curl >/dev/null 2>&1; then
        curl -sf "$@"
        return $?
    fi

    if [ "${use_host_network}" = true ]; then
        docker run --rm ${DOCKER_NETWORK} curlimages/curl:8.5.0 -sf "$@"
    else
        docker run --rm --network container:"${TOXIPROXY_CONTAINER}" curlimages/curl:8.5.0 -sf "$@"
    fi
}

toxiproxy_fail() {
    echo "$*" >&2
    docker logs "${TOXIPROXY_CONTAINER}" 2>&1 | tail -n 40 >&2 || true
    docker rm -f "${TOXIPROXY_CONTAINER}" &> /dev/null || true
    exit 1
}

if [ "$CMD" == "start" ]; then
    if [ "${use_host_network}" = true ]; then
        docker run -d --name="${TOXIPROXY_CONTAINER}" ${DOCKER_NETWORK} "${TOXIPROXY_IMAGE}"
    else
        docker run -d --name="${TOXIPROXY_CONTAINER}" \
            --add-host=host.docker.internal:host-gateway \
            -p "127.0.0.1:${TOXIPROXY_ADMIN_PORT}:8474" \
            -p "127.0.0.1:${TOXIPROXY_LISTEN_PORT}:${TOXIPROXY_LISTEN_PORT}" \
            "${TOXIPROXY_IMAGE}"
    fi
    if [ $? -ne 0 ]; then
        echo "Failed to start Toxiproxy container"
        exit 1
    fi

    for i in $(seq 1 30); do
        if toxiproxy_curl "http://127.0.0.1:${TOXIPROXY_ADMIN_PORT}/version" > /dev/null; then
            break
        fi
        # Container may have exited (e.g. admin port bind conflict).
        if [ -z "$(docker ps -q -f name="^/${TOXIPROXY_CONTAINER}$")" ]; then
            toxiproxy_fail "Toxiproxy container exited before admin API became ready"
        fi
        sleep 1
        if [ "$i" -eq 30 ]; then
            toxiproxy_fail "Toxiproxy admin API did not become ready on 127.0.0.1:${TOXIPROXY_ADMIN_PORT} (mode=$([ "${use_host_network}" = true ] && echo host || echo publish))"
        fi
    done

    create_body="{\"name\":\"${TOXIPROXY_PROXY_NAME}\",\"listen\":\"${TOXIPROXY_LISTEN}\",\"upstream\":\"${TOXIPROXY_UPSTREAM}\",\"enabled\":true}"
    create_out="$(toxiproxy_curl -X POST "http://127.0.0.1:${TOXIPROXY_ADMIN_PORT}/proxies" \
        -H "Content-Type: application/json" \
        -d "${create_body}" 2>&1)" || {
        echo "Failed to create Toxiproxy proxy ${TOXIPROXY_PROXY_NAME}" >&2
        echo "  listen=${TOXIPROXY_LISTEN} upstream=${TOXIPROXY_UPSTREAM}" >&2
        echo "  response: ${create_out}" >&2
        docker logs "${TOXIPROXY_CONTAINER}" 2>&1 | tail -n 40 >&2 || true
        docker rm -f "${TOXIPROXY_CONTAINER}" &> /dev/null || true
        exit 1
    }
    echo "Toxiproxy proxy ${TOXIPROXY_PROXY_NAME} ready (listen ${TOXIPROXY_LISTEN} → ${TOXIPROXY_UPSTREAM})"
elif [ "$CMD" == "cleanup" ]; then
    docker rm -f "${TOXIPROXY_CONTAINER}" &> /dev/null || true
else
    echo "Usage: $0 {start|cleanup}"
    exit 1
fi
