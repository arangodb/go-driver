#!/usr/bin/env bash
# Print the Starter image matching an ArangoDB server image.
# The server image ships the starter binary; parse its --version output
# instead of pinning a Starter tag here.
set -eu

IMAGE="${1:-}"
if [ -z "$IMAGE" ]; then
	echo "usage: starter_for_arangodb.sh ARANGODB_IMAGE" >&2
	exit 1
fi

STARTER_REPO="${STARTER_REPO:-docker.io/arangodb/arangodb-starter}"

# shellcheck disable=SC2086
VERSION_LINE=$(docker run --rm ${DOCKER_PLATFORM:-} --entrypoint arangodb "$IMAGE" --version 2>/dev/null | head -1)
# First line is "Version <tag>, build <hash>"; keep only <tag>.
VERSION=$(printf '%s' "$VERSION_LINE" | sed -n 's/^[Vv]ersion[[:space:]]\+\([^,[:space:]]\+\).*/\1/p')

if [ -z "$VERSION" ]; then
	echo "Failed to detect Starter version from $IMAGE (got: '$VERSION_LINE')" >&2
	echo "Set STARTER to pick the Starter image explicitly." >&2
	exit 1
fi

echo "${STARTER_REPO}:${VERSION}"
