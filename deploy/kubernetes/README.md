# Kubernetes Integration Tests

This folder contains the shared runner for executing **any** ArangoDB driver integration, **resiliency**, and **Toxiproxy** tests against a deployment managed by [kube-arangodb](https://github.com/arangodb/kube-arangodb).

The runner installs the kube-arangodb operator, creates an `ArangoDeployment`, creates an Ingress, and then runs your test command against that external endpoint. The test command can be anything: `make`, `npm test`, `pytest`, a shell script, etc.

**Supported local/CI cluster:** [kind](https://kind.sigs.k8s.io/) (create it with `setup-kind`). All ArangoDB drivers should use the same kind + ingress-nginx layout.

**Multi-driver onboarding (internal):** slide deck and resiliency / Toxiproxy scenario reference for other ArangoDB drivers live in [`documentation/`](documentation/) — start with [`documentation/README.md`](documentation/README.md). Complete scenario reference: [`documentation/driver-resiliency-reference.md`](documentation/driver-resiliency-reference.md).

## Quick Start

**Always run `setup-kind` first** on a machine (or CI job). It creates the kind cluster and installs ingress-nginx. Reuse that cluster across multiple test runs; run `setup-kind` again only after `cleanup-kind` or on a fresh machine.

**Step 1 — create the kind cluster (once):**

```bash
bash ./deploy/kubernetes/run-driver-tests.sh setup-kind
```

**Step 2 — run your driver-specific test command:**

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 bash ./deploy/kubernetes/run-driver-tests.sh run <driver-specific-run-command>
```

Examples:

```bash
# Go driver v2 (this repo)
K8S_INGRESS_ADDRESS=127.0.0.1 bash ./deploy/kubernetes/run-driver-tests.sh run make run-v2-tests-resiliency-k8s

# Any other driver — same runner, different command
K8S_INGRESS_ADDRESS=127.0.0.1 bash ./deploy/kubernetes/run-driver-tests.sh run npm run test:k8s
K8S_INGRESS_ADDRESS=127.0.0.1 bash ./deploy/kubernetes/run-driver-tests.sh run pytest tests/k8s
K8S_INGRESS_ADDRESS=127.0.0.1 bash ./deploy/kubernetes/run-driver-tests.sh run bash scripts/run-k8s-tests.sh
```

Version-specific commands for this repository are in `v2/tests/k8s-tests.md`.

## How It Works

```
run-driver-tests.sh run <your-test-command>
        │
        ├─ deploy ArangoDB on kind (operator, ArangoDeployment, Ingress)
        ├─ export endpoint / auth / Docker networking env vars
        ├─ export kubectl mount flags (for resiliency chaos inside Docker)
        └─ run <your-test-command>
```

Your test command only needs to **read those env vars**. It does not install the operator or create the Ingress.

### Two channels used by resiliency tests

| Channel | Env vars | Who uses it |
|---------|----------|-------------|
| **Driver → ArangoDB** | `TEST_ENDPOINTS_OVERRIDE`, `TEST_NET_OVERRIDE` | Every k8s test: HTTP(S) through Ingress (`http://arangodb.local`) |
| **Tests → Kubernetes API** | `K8S_TEST_DOCKER_EXTRA_ARGS` (and `KUBECTL_BIN` / `KUBECONFIG_PATH`) | **Resiliency only:** tests call `kubectl` to restart ingress or delete coordinator pods |

Plain integration tests (single/cluster) need only the first channel.

**Toxiproxy** needs the first channel redirected through a local proxy (`run-toxiproxy`); it does **not** call `kubectl` during the fault window. The cluster still comes from the same runner/`setup-kind`.

### Why `kubectl` on the host PATH?

1. **The runner itself** always needs `kubectl` on the host to apply manifests, wait for pods, and create Ingress (`setup-kind`, `start`, `run`, `run-toxiproxy`).
2. **Resiliency tests** also invoke `kubectl` **from inside the test process** (for go-driver: from the Docker test container). The runner therefore exports `K8S_TEST_DOCKER_EXTRA_ARGS` so your Docker `run` can mount the host `kubectl` binary and kubeconfig into the container.

Go-driver resiliency helpers call `kubectl` directly (e.g. `kubectl delete pod`, `kubectl rollout restart`) — see `v2/tests/resiliency_*_helper_test.go`. Other drivers should do the same for Part A chaos scenarios.

## Docker-based tests (recommended — same as go-driver v2)

All drivers should run tests **inside Docker**, the same way go-driver does:

1. Runner deploys the cluster and exports env vars (including `TEST_NET_OVERRIDE` and `K8S_TEST_DOCKER_EXTRA_ARGS`).
2. Your `<driver-specific-run-command>` starts a test container and **must pass those vars through**.

Go-driver wiring (for reference):

- Outer: `make run-k8s-v2-resiliency` → `run-driver-tests.sh run make run-v2-tests-resiliency-k8s`
- Inner Makefile uses `TEST_NET_OVERRIDE` for Ingress DNS and `K8S_TEST_DOCKER_EXTRA_ARGS` to mount kubectl/kubeconfig into `GOV2IMAGE`

Pattern for any language:

```bash
docker run --rm \
  ${TEST_NET_OVERRIDE} \
  ${K8S_TEST_DOCKER_EXTRA_ARGS} \
  -e TEST_ENDPOINTS="${TEST_ENDPOINTS_OVERRIDE}" \
  -e TEST_AUTHENTICATION="${TEST_AUTHENTICATION_OVERRIDE}" \
  -e K8S_NAMESPACE="${K8S_NAMESPACE}" \
  -e K8S_DEPLOYMENT="${K8S_DEPLOYMENT}" \
  -e K8S_COORDINATORS_COUNT="${K8S_COORDINATORS_COUNT}" \
  your-test-image \
  <your-test-entrypoint>
```

| Flag / env | Meaning |
|------------|---------|
| `${TEST_NET_OVERRIDE}` | Usually `--net=host --add-host=arangodb.local:127.0.0.1` so the container reaches kind Ingress |
| `${K8S_TEST_DOCKER_EXTRA_ARGS}` | Mounts host `kubectl` + kubeconfig into the container (required for resiliency chaos) |
| `TEST_ENDPOINTS` / auth | Where the driver connects |
| `K8S_NAMESPACE` / `K8S_DEPLOYMENT` | Which pods/services chaos helpers target |

If your driver maps names differently (`ARANGO_URL`, etc.), do that in a thin wrapper script and still pass the Docker flags above.

### Mapping runner env vars (optional adapter)

```bash
#!/usr/bin/env bash
# scripts/run-k8s-tests.sh — example for npm/pytest drivers
export ARANGO_URL="${TEST_ENDPOINTS_OVERRIDE}"
export ARANGO_AUTH="${TEST_AUTHENTICATION_OVERRIDE}"

docker run --rm \
  ${TEST_NET_OVERRIDE} \
  ${K8S_TEST_DOCKER_EXTRA_ARGS} \
  -e ARANGO_URL -e ARANGO_AUTH \
  -e K8S_NAMESPACE -e K8S_DEPLOYMENT -e K8S_COORDINATORS_COUNT \
  your-test-image \
  npm run test:integration
```

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 bash ./deploy/kubernetes/run-driver-tests.sh run bash scripts/run-k8s-tests.sh
```

## Resiliency and Toxiproxy entry points

```bash
# After setup-kind (see Quick Start)

# Resiliency — 3 coordinators; Docker tests + kubectl mounts (go-driver)
K8S_INGRESS_ADDRESS=127.0.0.1 bash ./deploy/kubernetes/run-driver-tests.sh run make run-v2-tests-resiliency-k8s

# Toxiproxy — 1 coordinator; Driver → Toxiproxy → Ingress (go-driver)
K8S_INGRESS_ADDRESS=127.0.0.1 bash ./deploy/kubernetes/run-driver-tests.sh run-toxiproxy make run-v2-tests-toxiproxy-k8s
```

Toxiproxy layout:

```text
Driver → Toxiproxy (127.0.0.1:17001) → Ingress → Coordinator
```

`run-toxiproxy` still uses the Docker-based path (`TEST_NET_OVERRIDE`). Chaos via `kubectl` is not required for Toxiproxy scenarios; network faults are injected by the proxy.

## Environment variables for other drivers

### Always exported by `run` / `run-toxiproxy` (read these in your tests)

| Variable | Example | Purpose |
|----------|---------|---------|
| `TEST_ENDPOINTS_OVERRIDE` | `http://arangodb.local` (`run`) or `http://127.0.0.1:17001` (`run-toxiproxy`) | Driver endpoint |
| `TEST_AUTHENTICATION_OVERRIDE` | `basic:root:rootpw` | Auth string |
| `TEST_AUTHENTICATION` | same | Legacy Go alias |
| `TEST_MODE_K8S` | `k8s` | Marks Kubernetes test mode |
| `TEST_NOT_WAIT_UNTIL_READY` | `1` | Skip non-k8s readiness waits |
| `TEST_NET_OVERRIDE` | `--net=host --add-host=arangodb.local:127.0.0.1` | Pass into `docker run` |
| `K8S_NAMESPACE` | `default` | Deployment namespace |
| `K8S_DEPLOYMENT` | `arangodb-driver-tests` | `ArangoDeployment` name |
| `K8S_COORDINATORS_COUNT` | `1` or `3` | Coordinator replicas (`run …resiliency…` auto-raises to 3) |

### Extra for resiliency (kubectl from inside Docker)

| Variable | Example | Purpose |
|----------|---------|---------|
| `K8S_TEST_DOCKER_EXTRA_ARGS` | `-v /usr/bin/kubectl:/usr/local/bin/kubectl:ro -v …/config:/root/.kube/config:ro -e KUBECONFIG=…` | Pass into `docker run` so tests can call `kubectl` |
| `KUBECTL_BIN` | `/usr/bin/kubectl` | Host kubectl path (informational; mounts come from `K8S_TEST_DOCKER_EXTRA_ARGS`) |
| `KUBECONFIG_PATH` | `~/.kube/config` | Host kubeconfig path |

### Extra for Toxiproxy (`run-toxiproxy`)

| Variable | Purpose |
|----------|---------|
| `TEST_ENDPOINTS_OVERRIDE` | Points at Toxiproxy listen URL, not Ingress hostname |
| `TEST_INGRESS_HOST` | Hostname for Ingress routing (e.g. `arangodb.local`) while connecting via `127.0.0.1:listen` |
| Toxiproxy listen/admin ports | Started by the driver’s toxiproxy helper (go-driver: `test/toxiproxy.sh`); runner sets listen port via `K8S_TOXIPROXY_LISTEN_PORT` (default `17001`) |

### Rename exports without changing the runner

| Override variable | Default export name |
|-------------------|---------------------|
| `K8S_TEST_ENDPOINTS_ENV` | `TEST_ENDPOINTS_OVERRIDE` |
| `K8S_TEST_AUTHENTICATION_ENV` | `TEST_AUTHENTICATION_OVERRIDE` |
| `K8S_TEST_LEGACY_AUTHENTICATION_ENV` | `TEST_AUTHENTICATION` |
| `K8S_TEST_MODE_ENV` | `TEST_MODE_K8S` |
| `K8S_TEST_NOT_WAIT_UNTIL_READY_ENV` | `TEST_NOT_WAIT_UNTIL_READY` |
| `K8S_TEST_NET_ENV` | `TEST_NET_OVERRIDE` |
| `K8S_TEST_DOCKER_EXTRA_ARGS_ENV` | `K8S_TEST_DOCKER_EXTRA_ARGS` |

Set a `K8S_TEST_*_ENV` variable to empty to disable that export.

## CircleCI

CircleCI runs the same shared runner through `run-k8s-integration-tests`. The job installs `kubectl` and `kind`, starts a Docker-backed kind cluster with ingress-nginx, and runs tests from the existing Docker test container through the Kubernetes Ingress endpoint.

Pull requests also run:

- `make run-k8s-v2-resiliency` (3 coordinators, ingress/coordinator chaos scenarios; ~10 minutes). Resiliency jobs use a longer `K8S_WAIT_TIMEOUT` because the cluster deploys three coordinators.
- `make run-k8s-v2-toxiproxy` (1 coordinator, network-fault scenarios via Toxiproxy → ingress).

The CircleCI jobs are guarded by the existing pull-request check and skip kind setup on non-PR pipelines.

Override the target or image:

```bash
ARANGODB=gcr.io/gcr-for-testing/arangodb/enterprise-preview:latest \
make run-k8s-v2-tests
```

## Useful Environment

- `KUBE_ARANGODB_VERSION`: kube-arangodb enterprise operator version, default `1.4.3` (Git manifests and image tag must match).
- `KUBE_ARANGODB_IMAGE`: operator image override, default `arangodb/kube-arangodb-enterprise:${KUBE_ARANGODB_VERSION}`. Do not use `:latest` unless it matches the manifest version — mismatched tags crash the operator.
- `K8S_INSTALL_OPERATOR`: set to `false` when the kube-arangodb operator is already installed in `default` (skips GitHub manifest download). Default `true`.
- `K8S_COORDINATORS_COUNT`: number of coordinators in Cluster mode, default `1`. Resiliency `run` commands auto-raise this to `3` when the test command contains `resiliency`. The `start` command alone does **not** auto-raise — set `K8S_COORDINATORS_COUNT=3` explicitly if you use `start` for a resiliency-shaped cluster.
- `K8S_NAMESPACE`: namespace for the temporary `ArangoDeployment`, default `default`. When `K8S_INSTALL_OPERATOR=true`, the raw kube-arangodb manifests install the operator in `default`, so keep the deployment in `default`. For another namespace, preinstall an operator watching that namespace and set `K8S_INSTALL_OPERATOR=false`.
- `K8S_DEPLOYMENT`: deployment name, default `arangodb-driver-tests`.
- `K8S_MODE`: `Cluster` or `Single`, default `Cluster`.
- `K8S_AUTHENTICATION`: set to `false` to disable ArangoDB authentication in the Kubernetes deployment, default `true`.
- `K8S_TEST_AUTHENTICATION`: driver authentication mode, `basic`, `jwt`, or `none`, default `basic`.
- `K8S_TLS`: set to `true` to enable TLS in the `ArangoDeployment` (Ingress uses `backend-protocol: HTTPS`). With `K8S_INGRESS_TLS=true` this is end-to-end TLS (`make run-k8s-v2-resiliency-e2e-tls`, `make run-k8s-v2-toxiproxy-e2e-tls`).
- `K8S_INGRESS_HOST`: host name used by ingress mode, default `arangodb.local`.
- `K8S_INGRESS_ADDRESS`: IP address mapped into the Docker test container for `K8S_INGRESS_HOST`. CircleCI sets this to `127.0.0.1` for the kind ingress port mapping. When empty, the runner uses the Ingress load balancer status.
- `K8S_INGRESS_TLS`: set to `false` to expose the Ingress over HTTP instead of HTTPS, default `true`. Ingress-only HTTPS (pods still HTTP): `make run-k8s-v2-resiliency-tls`, `make run-k8s-v2-toxiproxy-tls`.
- `K8S_STUCK_INIT_TIMEOUT`: force-delete ArangoDB pods stuck in any running init container (`uuid`, `init-lifecycle`, `version-check`, …) or stuck Terminating longer than this, default `5m`. Uses `--force --grace-period=0` because member pods often have a 3600s termination grace period.
- `K8S_KEEP_DEPLOYMENT`: set to `true` to keep the deployment after a run.
- `K8S_DELETE_NAMESPACE`: set to `true` to delete a non-default namespace during cleanup.
- `K8S_DELETE_KIND_CLUSTER`: set to `true` to delete the kind cluster after a run.
- `K8S_INGRESS_NGINX_VERSION`: ingress-nginx release used by `setup-kind`, default `controller-v1.12.1`.
- `K8S_TEST_WORKDIR`: working directory for the test command, default repository root.
- `KUBECTL_BIN`: override kubectl binary path (auto-detected when empty).
- `KUBECONFIG_PATH`: override kubeconfig file path (defaults to `$KUBECONFIG` or `~/.kube/config`).
- `K8S_TOXIPROXY_LISTEN_PORT`: Toxiproxy listen port for `run-toxiproxy`, default `17001`.
- `ARANGO_ROOT_PASSWORD`: root password configured in Kubernetes and passed to tests, default `rootpw`.
- `ARANGO_LICENSE_KEY`: optional Enterprise license key. When set, the runner creates the kube-arangodb license secret and references it from the `ArangoDeployment`.
- `ENABLE_VECTOR_INDEX`: set to `true` to add `--vector-index=true` and `--experimental-vector-index=true` to the ArangoDB pods.

The runner creates a self-signed TLS secret and an Ingress for `K8S_INGRESS_HOST`, then passes `https://K8S_INGRESS_HOST` to the Dockerized tests with a Docker `--add-host` mapping to the ingress IP.

Single mode starts one ArangoDB server. Cluster mode starts 1 Agent, 3 DBServers, and **1 Coordinator by default** (`K8S_COORDINATORS_COUNT=1`). Resiliency tests need **3 coordinators** — use `make run-k8s-v2-resiliency` (auto-raised) or set `K8S_COORDINATORS_COUNT=3` when calling `start` manually. The 3 DBServers are needed because some integration tests update collection replication factor to 3.
