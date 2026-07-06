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
- `make run-k8s-v2-resiliency-tls`
- `make run-k8s-v2-toxiproxy`
- `make run-k8s-v2-toxiproxy-tls`

## Resiliency Tests

**Full scenario reference (steps, expected errors, observed Go v2 errors, timing budgets):** [driver-resiliency-reference.md](driver-resiliency-reference.md)

All resiliency scenarios (ingress restart, coordinator failure) run through a single entry point. Tests execute **inside Docker** with `kubectl` and `kubeconfig` mounted via `K8S_TEST_DOCKER_EXTRA_ARGS` from `run-driver-tests.sh`.

Prerequisite: run `setup-kind` first (see [Local Run](#local-run)).

The runner deploys a cluster with **3 coordinators** (`K8S_COORDINATORS_COUNT=3`) and runs every `TestResiliency_*` test via:

```text
run-k8s-v2-resiliency → run-driver-tests.sh run → make run-v2-tests-resiliency-k8s
```

HTTP ingress:

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency
```

HTTPS ingress (self-signed TLS secret created by the runner; the driver skips certificate verification in tests):

```bash
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-resiliency-tls
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

### Scenarios covered

**Ingress restart**

- Idle client survives ingress-nginx controller restart
- Active `Version()` workload survives ingress restart (temporary failures allowed)

**Coordinator failure** (requires 3 coordinators)

- Coordinator restart while idle
- Coordinator restart during active `Version()` workload
- Kill coordinator during read
- Kill coordinator during insert
- Kill coordinator during cursor iteration

What is validated:

- The driver does not panic, hang, or deadlock
- Temporary connection errors during failure windows are allowed
- After recovery, `client.Version()` and new operations succeed again

What is not validated:

- Every request succeeding during the failure window

## Toxiproxy Network Fault Tests

**Full scenario reference (steps, expected errors, observed Go v2 errors, timing budgets):** [driver-resiliency-reference.md](driver-resiliency-reference.md)

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

Run a single test:

```bash
# Prefer TOXIPROXY_TEST_RUN for regex patterns with (|) — avoids shell quoting issues.
export TOXIPROXY_TEST_RUN='TestToxiproxy_(Partial|Full)PacketLoss'
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy VERBOSE=1

# Or a single test without special characters:
export TESTOPTIONS='-test.run TestToxiproxy_AbruptTCPConnectionClose'
K8S_INGRESS_ADDRESS=127.0.0.1 make run-k8s-v2-toxiproxy VERBOSE=1
```

### Scenarios covered

**Category 1 — Connection loss**

- Abrupt TCP connection close (`reset_peer` toxic): next request fails with a connection error (no panic); future requests succeed after toxic removal
- Network disconnect (`proxy.Disable()` / `proxy.Enable()`): requests fail while proxy is disabled; driver usable after reconnect
- Connection reset by peer (`reset_peer` downstream): RST injection yields connection reset or unexpected EOF; driver recovers after toxic removal

**Category 2 — Latency**

- High latency (2s upstream): `Version()` succeeds within a longer context timeout
- Extreme latency (30s upstream): `Version()` fails with context deadline exceeded when timeout is shorter
- Latency removed: request duration returns to normal after toxic removal

**Category 3 — Timeouts**

- Context timeout (20s upstream latency, 2s caller deadline): `Version()` fails with context deadline exceeded before the full latency elapses
- Server timeout (20s downstream response delay, 2s response-header deadline): `Version()` reports a driver timeout without hanging for the full delay

**Category 4 — Packet loss**

- Partial packet loss (~30% upstream `reset_peer` toxicity): some `Version()` calls fail with transport errors, no panic, all succeed after toxic removal
- Full packet loss (100% upstream `timeout` toxic): `Version()` times out while data is blocked, driver recovers after toxic removal

**Category 5 — Bandwidth limits**

- Slow upload (20 KB/s upstream): bulk insert is slower than baseline; stored documents remain intact
- Slow download (20 KB/s downstream): full-collection read is slower than baseline; document payloads remain intact

**Category 6 — Streaming responses**

- Disconnect during cursor iteration: `ReadDocument()` returns a clean transport error after proxy disable mid-cursor; no panic; driver recovers after reconnect
- Disconnect during large AQL query: long-running query is interrupted with a clean error when the proxy is disabled mid-stream; driver recovers after reconnect

## CircleCI

On pull requests, CircleCI runs the same targets via `run-k8s-integration-tests`:

- `make run-k8s-v2-single`
- `make run-k8s-v2-cluster`
- `make run-k8s-v2-resiliency` (3 coordinators; ingress and coordinator failure scenarios)

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
