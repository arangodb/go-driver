#!/bin/bash

# Start or stop a Toxiproxy instance for driver resiliency tests.
# The proxy listens on TOXIPROXY_LISTEN and forwards to TOXIPROXY_UPSTREAM.

if [ -z "$TESTCONTAINER" ]; then
    echo "TESTCONTAINER environment variable must be set"
    exit 1
fi

CMD=$1
TOXIPROXY_CONTAINER=${TESTCONTAINER}-toxiproxy
TOXIPROXY_IMAGE=${TOXIPROXY_IMAGE:-ghcr.io/shopify/toxiproxy:2.9.0}
TOXIPROXY_ADMIN_PORT=${TOXIPROXY_ADMIN_PORT:-8474}
TOXIPROXY_LISTEN_PORT=${TOXIPROXY_LISTEN_PORT:-17001}
TOXIPROXY_LISTEN=${TOXIPROXY_LISTEN:-127.0.0.1:${TOXIPROXY_LISTEN_PORT}}
TOXIPROXY_UPSTREAM=${TOXIPROXY_UPSTREAM:-127.0.0.1:7001}
TOXIPROXY_PROXY_NAME=${TOXIPROXY_PROXY_NAME:-arangodb}
DOCKER_NETWORK=${DOCKER_NETWORK:---net=host}

docker rm -f "${TOXIPROXY_CONTAINER}" &> /dev/null

toxiproxy_curl() {
    if curl -sf "$@" > /dev/null 2>&1; then
        curl -sf "$@"
        return $?
    fi

    docker run --rm ${DOCKER_NETWORK} curlimages/curl:8.5.0 -sf "$@"
}

if [ "$CMD" == "start" ]; then
    docker run -d --name="${TOXIPROXY_CONTAINER}" ${DOCKER_NETWORK} "${TOXIPROXY_IMAGE}"
    if [ $? -ne 0 ]; then
        echo "Failed to start Toxiproxy container"
        exit 1
    fi

    for i in $(seq 1 30); do
        if toxiproxy_curl "http://127.0.0.1:${TOXIPROXY_ADMIN_PORT}/version" > /dev/null; then
            break
        fi
        sleep 1
        if [ "$i" -eq 30 ]; then
            echo "Toxiproxy admin API did not become ready"
            exit 1
        fi
    done

    toxiproxy_curl -X POST "http://127.0.0.1:${TOXIPROXY_ADMIN_PORT}/proxies" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"${TOXIPROXY_PROXY_NAME}\",\"listen\":\"${TOXIPROXY_LISTEN}\",\"upstream\":\"${TOXIPROXY_UPSTREAM}\",\"enabled\":true}" > /dev/null
    if [ $? -ne 0 ]; then
        echo "Failed to create Toxiproxy proxy ${TOXIPROXY_PROXY_NAME}"
        exit 1
    fi
elif [ "$CMD" == "cleanup" ]; then
    :
else
    echo "Usage: $0 {start|cleanup}"
    exit 1
fi
