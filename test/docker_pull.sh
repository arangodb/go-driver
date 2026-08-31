#!/usr/bin/env bash
# Pull a Docker image with retries so a transient registry timeout does not fail
# the whole test run (GCR timeouts are common in CircleCI docker-in-docker).
# Images already present are reused unless DOCKER_PULL_FORCE is set.
set -u

IMAGE="${1:-}"
if [ -z "$IMAGE" ]; then
	echo "usage: docker_pull.sh IMAGE" >&2
	exit 1
fi

# Locally built debug image; not in a registry.
case "$IMAGE" in
go-driver-tests:*)
	echo "Skipping pull of local image: $IMAGE"
	exit 0
	;;
esac

HAVE_LOCAL=no
if docker image inspect "$IMAGE" >/dev/null 2>&1; then
	HAVE_LOCAL=yes
fi

# Mutable tags (nightly, latest) are refreshed on every run via DOCKER_PULL_FORCE.
if [ "$HAVE_LOCAL" = "yes" ] && [ -z "${DOCKER_PULL_FORCE:-}" ]; then
	echo "Image already present: $IMAGE"
	exit 0
fi

RETRIES="${DOCKER_PULL_RETRIES:-5}"
attempt=1
while [ "$attempt" -le "$RETRIES" ]; do
	echo "Pulling $IMAGE (attempt ${attempt}/${RETRIES})"
	if [ -n "${DOCKER_PLATFORM:-}" ]; then
		# shellcheck disable=SC2086
		if docker pull ${DOCKER_PLATFORM} "$IMAGE"; then
			exit 0
		fi
	else
		if docker pull "$IMAGE"; then
			exit 0
		fi
	fi
	if [ "$attempt" -eq "$RETRIES" ]; then
		break
	fi
	sleep_sec=$((attempt * 5))
	echo "docker pull failed for $IMAGE; retrying in ${sleep_sec}s..."
	sleep "$sleep_sec"
	attempt=$((attempt + 1))
done

if [ "$HAVE_LOCAL" = "yes" ]; then
	echo "Failed to refresh $IMAGE after ${RETRIES} attempts; using local copy" >&2
	exit 0
fi

echo "Failed to pull $IMAGE after ${RETRIES} attempts" >&2
exit 1
