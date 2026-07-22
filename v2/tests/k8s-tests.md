# v2 Kubernetes Integration Tests

These tests run the v2 Go driver integration suite against an ArangoDB deployment managed by kube-arangodb.

The shared Kubernetes setup lives in `deploy/kubernetes/run-driver-tests.sh`. The `setup-kind` command creates the kind cluster and installs ingress-nginx. Each Make target below then installs kube-arangodb, creates the `ArangoDeployment`, and runs the tests.

## Local Run

**Step 1 — create the kind cluster (once per machine, or after `cleanup-kind`):**

```bash
bash ./deploy/kubernetes/run-driver-tests.sh setup-kind
```

**Step 2 — run tests** (`K8S_INGRESS_ADDRESS=127.0.0.1` maps the ingress hostname for kind):

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-tests
```

Run only the v2 cluster basic-auth scenario:

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 VERBOSE=1 ENABLE_VECTOR_INDEX=true make run-k8s-v2-cluster-basic-auth
```

## Make Targets

- `make run-k8s-v2-tests`
- `make run-k8s-v2-single`
- `make run-k8s-v2-cluster`
- `make run-k8s-v2-single-without-auth`
- `make run-k8s-v2-single-basic-auth`
- `make run-k8s-v2-single-tls-basic-auth`
- `make run-k8s-v2-cluster-basic-auth`
- `make run-k8s-v2-cluster-tls-basic-auth`
- `make run-k8s-v2-resiliency`
- `make run-k8s-v2-resiliency-tls` (HTTPS Ingress only; ArangoDB pods HTTP)
- `make run-k8s-v2-resiliency-e2e-tls` (HTTPS Ingress + HTTPS ArangoDB / `K8S_TLS=true`)
- `make run-k8s-v2-toxiproxy`
- `make run-k8s-v2-toxiproxy-tls` (HTTPS Ingress only; ArangoDB pods HTTP)
- `make run-k8s-v2-toxiproxy-e2e-tls` (HTTPS Ingress + HTTPS ArangoDB / `K8S_TLS=true`)

## Resiliency Tests

**Full scenario reference (steps, expected errors, observed Go v2 errors, timing budgets; internal driver use):** [driver-resiliency-reference.md](../../deploy/kubernetes/documentation/driver-resiliency-reference.md)

All resiliency scenarios (ingress restart, coordinator failure) run through a single entry point. Tests execute **inside Docker** with `kubectl` and `kubeconfig` mounted via `K8S_TEST_DOCKER_EXTRA_ARGS` from `run-driver-tests.sh`.

Prerequisite: run `setup-kind` first (see [Local Run](#local-run)).

**Local tip:** After the first successful run, set `K8S_INSTALL_OPERATOR=false` to avoid GitHub `429` rate limits on operator manifests. See [deploy/kubernetes/README.md — Troubleshooting](../../deploy/kubernetes/README.md#troubleshooting).

The runner deploys a cluster with **3 coordinators** (`K8S_COORDINATORS_COUNT=3`; agency stays at default **1** agent unless you set `K8S_AGENTS_COUNT=3`) and runs every `TestResiliency_*` test via:

```text
run-k8s-v2-resiliency → run-driver-tests.sh run → make run-v2-tests-resiliency-k8s
```

HTTP ingress:

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency
```

HTTPS ingress (self-signed TLS secret created by the runner; the driver skips certificate verification in tests). Ingress TLS only — coordinators stay HTTP:

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency-tls
```

End-to-end TLS (HTTPS Driver↔Ingress **and** HTTPS Ingress↔ArangoDB):

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency-e2e-tls
```

Test logs are enabled by default (`-v`). To reduce output:

```bash
K8S_RESILIENCY_TEST_VERBOSE= K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency
```

Run a single resiliency test (same `TESTOPTIONS` pattern as other v2 tests):

```bash
export TESTOPTIONS="-test.run TestResiliency_CoordinatorRestartWhileIdle -test.v"
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency
```

When `TESTOPTIONS` is unset, all `TestResiliency_*` tests run (default `-run '^TestResiliency_'`).

### Scenarios covered (Part A — aligns with reference #0–#8)

| # | Scenario | What it covers |
|---|----------|----------------|
| **0** | `LoadBalancerCoordinatorDistribution` | Observational: probes through ingress; logs which coordinator handled requests (no fault) |
| **1** | `IngressCoordinatorFailover` | Delete **1** random coordinator; probes continue / recover through ingress |
| **2** | `IngressRestartWhileIdle` | Idle client survives ingress-nginx controller restart |
| **3** | `IngressRestartDuringActiveWorkload` | Active `Version()` loop survives ingress restart (transient errors A–D allowed) |
| **4** | `CoordinatorRestartWhileIdle` | Idle client survives delete/recreate of **all 3** coordinators |
| **5** | `CoordinatorRestartDuringActiveWorkload` | Active `Version()` loop while all 3 coordinators are recreated (`failuresDuring` may be 0) |
| **6** | `CoordinatorKillDuringRead` | Kill all 3 coordinators during streaming cursor read; after recovery, dead-cursor resume must fail with cursor-gone / closed connection (not gateway-down) |
| **7** | `CoordinatorKillDuringInsert` | Kill **1** coordinator during insert loop (`failuresDuring` may be 0) |
| **8** | `CoordinatorKillDuringCursorIteration` | Kill all 3 coordinators after ~30 cursor docs; same post-recovery dead-cursor check as #6 |

What is validated:

- The driver does not panic, hang, or deadlock
- Temporary connection errors during failure windows are allowed (categories **A–E** per scenario; see reference)
- After recovery, `client.Version()` and new operations succeed again
- Active-workload recovery requires successes **after** ready (mid-chaos successes do not count as `successesAfter`)

What is not validated:

- Every request succeeding during the failure window
- Exact error strings (classify by category, not one message)

## Toxiproxy Network Fault Tests

**Full scenario reference (steps, expected errors, observed Go v2 errors, timing budgets; internal driver use):** [driver-resiliency-reference.md](../../deploy/kubernetes/documentation/driver-resiliency-reference.md)

Toxiproxy sits between the driver and ingress to simulate production network failures without touching Kubernetes or ArangoDB pods:

```text
Go Driver → Toxiproxy → Ingress → Coordinator
```

Prerequisite: run `setup-kind` first (see [Local Run](#local-run)).

HTTP ingress:

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy
```

HTTPS ingress:

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy-tls
```

End-to-end TLS (HTTPS through Toxiproxy→Ingress and HTTPS Ingress↔ArangoDB):

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy-e2e-tls
```

Run a single test:

```bash
# Prefer TOXIPROXY_TEST_RUN for regex patterns with (|) — avoids shell quoting issues.
export TOXIPROXY_TEST_RUN='TestToxiproxy_(Partial|Full)PacketLoss'
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy VERBOSE=1

# Or a single test without special characters:
export TESTOPTIONS='-test.run TestToxiproxy_AbruptTCPConnectionClose'
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy VERBOSE=1
```

### Scenarios covered (Part B — aligns with reference #1–#14)

| # | Scenario | Fault |
|---|----------|--------|
| **1** | `AbruptTCPConnectionClose` | Upstream `reset_peer` on live connection |
| **2** | `NetworkDisconnect` | `proxy.Disable()` / `Enable()` |
| **3** | `ConnectionResetByPeer` | Downstream `reset_peer` |
| **4** | `HighLatency` | 2s upstream latency; request still succeeds |
| **5** | `ExtremeLatency` | 30s upstream; short deadline → timeout (B) |
| **6** | `LatencyRemoved` | Duration recovers after toxic removal |
| **7** | `ContextTimeout` | Caller context deadline on slow path |
| **8** | `ServerTimeout` | Response-header timeout (HTTP/1 only) |
| **9** | `PartialPacketLoss` | ~40% upstream `reset_peer` over 40 attempts (HTTP/1 only; A or B) |
| **10** | `FullPacketLoss` | 100% upstream `timeout` toxic |
| **11** | `DisconnectDuringCursorIteration` | Disable proxy mid-cursor read |
| **12** | `DisconnectDuringQueryExecution` | Disable proxy during `db.Query()` startup |
| **13** | `DisconnectDuringInsert` | Disable proxy mid-insert; write outcome **unknown** |
| **14** | `DisconnectDuringTransactionCommit` | Disable proxy mid-commit; commit outcome **unknown** |

## CircleCI

On pull requests, CircleCI runs the same targets via `run-k8s-integration-tests`:

- `make run-k8s-v2-single`
- `make run-k8s-v2-cluster`
- `make run-k8s-v2-resiliency` (3 coordinators; ingress and coordinator failure scenarios; `k8s-wait-timeout: 35m`, `no_output_timeout: 40m`)
- `make run-k8s-v2-toxiproxy` (1 coordinator; network-fault scenarios via Toxiproxy → ingress; `no_output_timeout: 30m`)

All k8s jobs use kube-arangodb enterprise **1.4.3** (`docker-hub` context). Expect longer wall time for resiliency (~10–30+ minutes depending on chaos windows).

See `deploy/kubernetes/README.md` for the shared runner details.

## Cleanup

The runner removes the temporary ArangoDeployment, Ingress, and secrets after `run` unless `K8S_KEEP_DEPLOYMENT=true` is set.

To delete the kind cluster as well:

```bash
bash ./deploy/kubernetes/run-driver-tests.sh cleanup-kind
```

For a fully clean run that deletes the kind cluster after tests complete:

```bash
K8S_DELETE_KIND_CLUSTER=true K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-tests
```
