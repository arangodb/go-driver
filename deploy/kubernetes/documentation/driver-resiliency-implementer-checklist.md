# Driver resiliency — implementer checklist

One-page handoff for official ArangoDB driver maintainers implementing shared resiliency / Toxiproxy tests. Behavioral contract and steps live elsewhere; this page only points.

## Sources of truth

| Need | Read |
| --- | --- |
| **Behavioral requirements & acceptance** | OpenSpec capabilities under `openspec/specs/<capability>/spec.md` |
| **Scenario steps, timing, error examples** | [`driver-resiliency-reference.md`](./driver-resiliency-reference.md) |
| **Scenario → capability map** | [Scenario → Capability Mapping](./driver-resiliency-reference.md#scenario--capability-mapping) in the reference |
| **Harness environment contract** | Reference [Harness env contract](./driver-resiliency-reference.md#harness-env-contract-all-drivers) and overview slides [`driver-k8s-shared-infra-demo.html`](./driver-k8s-shared-infra-demo.html) |
| **Shared runners** | `deploy/kubernetes/run-driver-tests.sh` (cluster + ingress) and `test/toxiproxy.sh` (proxy) — integrate; do not re-specify kubectl recipes here |

## OpenSpec capabilities (implement these)

| Capability | Covers |
| --- | --- |
| `error-classification` | Categories A–F, phase acceptance, C vs D, universal rejects |
| `load-balancing` | Part A #0 — observational coordinator distribution |
| `ingress-resiliency` | Part A #1–#3 — failover, idle/active ingress restart |
| `coordinator-resiliency` | Part A #4–#8 — coordinator restart/kill scenarios |
| `toxiproxy-connection-interruption` | Part B #1–#3 |
| `toxiproxy-latency` | Part B #4–#8 |
| `toxiproxy-packet-loss` | Part B #9–#10 |
| `toxiproxy-streaming-operations` | Part B #11–#12 |
| `toxiproxy-write-operations` | Part B #13–#14 |

## Recommended implementation order

1. `error-classification` (shared by all fault scenarios)
2. Harness env wiring (endpoints, auth, Host header, Toxiproxy admin/listen — per demo HTML / harness table)
3. `load-balancing`
4. `ingress-resiliency`
5. `coordinator-resiliency`
6. Toxiproxy subsets: connection → latency → packet loss → streaming → writes

## Checklist

- [ ] Honor the harness env contract (read or remap `TEST_*` / `TOXIPROXY_*` names; preserve semantics).
- [ ] Classify failures by **category A–F**, not by one exact error string (HTTP/1 vs HTTP/2 text may differ).
- [ ] For each scenario, follow reference steps and assert OpenSpec acceptance criteria for that capability.
- [ ] Cover HTTP/1 and HTTP/2 when the driver and server support both (skip only where the reference allows).
- [ ] Assert recovery on the required client after the fault is cleared.
- [ ] Reject hangs, unexpected process termination, category F during fault windows, and “successful” cursor completion across an intentional kill.
- [ ] Keep language-specific helpers local; do not copy another driver’s assertion APIs into the shared specs.

## Out of scope for this page

Kubernetes command dumps, Make targets, CI logs, and language-specific helper implementations — use the reference and shared scripts when those details are needed.
