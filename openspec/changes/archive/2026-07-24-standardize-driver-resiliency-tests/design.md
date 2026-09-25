## Context

The Go Driver already implements a comprehensive resiliency test suite (Part A Kubernetes resiliency #0–#8 and Part B Toxiproxy #1–#14 under build tags `resiliency` and `toxiproxy`) using the shared Kubernetes and Toxiproxy test infrastructure — primarily `deploy/kubernetes/run-driver-tests.sh` and `test/toxiproxy.sh`, plus the documented environment contract.

To enable consistent resiliency testing across all official ArangoDB drivers, a language-independent OpenSpec is needed so other drivers can implement the same behaviors and acceptance criteria without relying on Go-specific implementation details or duplicating Go-specific test logic. Other official drivers are expected to integrate with the same shared Kubernetes and Toxiproxy test infrastructure (including `run-driver-tests.sh`, `test/toxiproxy.sh`, and the documented environment contract) while implementing the behavioral requirements defined by this OpenSpec.

This design codifies the existing Go Driver resiliency behavior into capability-oriented specifications that serve as the shared behavioral contract for all official drivers. It does not add driver product features. Step-level detail remains in `deploy/kubernetes/documentation/driver-resiliency-reference.md`; harness env naming is also summarized in `deploy/kubernetes/documentation/driver-k8s-shared-infra-demo.html`.

Stakeholders: Go, Java, JavaScript, Python, and other official driver maintainers; shared k8s harness owners.

## Goals / Non-Goals

**Goals:**

- Codify existing Go Driver resiliency behavior into language-independent requirements, scenarios, expected behavior, and acceptance criteria per capability.
- Provide a shared behavioral contract so other official drivers can implement the same suite against the shared Kubernetes/Toxiproxy harness.
- Split the suite into reviewable capabilities (ingress, LB, coordinators, Toxiproxy fault classes, error classification).
- Point implementers to the reference doc for fault injection steps, timing budgets, and example messages — not to Go helpers or runner internals.
- Require classification by error category (A–F), not exact strings; require HTTP/1 and HTTP/2 coverage where the driver supports both.
- Document the shared harness environment contract that drivers MUST honor when integrating with `run-driver-tests.sh` and `test/toxiproxy.sh`.

**Non-Goals:**

- Re-implement or modify the Go Driver resiliency and Toxiproxy test suites as part of this change (Go is the reference implementation; work is alignment verification only).
- Embed Kubernetes commands, Make targets, Docker recipes, Go helper names, or CI log dumps in specs.
- Mandate identical helper APIs or assertion libraries across languages.
- Require Ingress TLS / end-to-end TLS for Part B CI (HTTP path is the default contract; TLS modes remain optional).
- Change shared infrastructure scripts (`run-driver-tests.sh`, `toxiproxy.sh`) unless a gap is found during later apply work.
- Introduce new product functionality or driver APIs.

## Decisions

### 1. Capability split over a monolithic resiliency spec

**Choice:** Nine capabilities as listed in the proposal.

**Rationale:** Part A and Part B already group by fault domain. Splitting Toxiproxy into connection, latency, packet loss, streaming, and write mirrors how other drivers will implement and CI-gate subsets. Error classification is shared cross-cutting and must not be re-copied into every scenario spec.

**Alternatives considered:** Single `driver-resiliency` spec (harder to review/partially implement); Part A vs Part B only (too coarse for Toxiproxy).

### 2. Reference doc remains the step-level source of truth

**Choice:** Specs state *what* must hold; `driver-resiliency-reference.md` remains authoritative for *how* to inject faults, wait budgets, and observed message patterns.

**Rationale:** Avoid duplicating and drifting from the long reference. Specs cite scenario names (#0–#8, Toxiproxy #1–#14) that map 1:1 to the reference catalog.

**Alternatives considered:** Copy all steps into OpenSpec (duplication/drift); replace the reference with OpenSpec only (loses operational detail other teams already use).

### 3. Language-agnostic norms; Go as reference implementation

**Choice:** Normative text uses driver-neutral terms (`Version()`, cursor read, document create, transaction commit, HTTP client). Go paths are examples only.

**Rationale:** Other drivers map APIs differently; acceptance is behavioral (fail cleanly, recover, correct categories).

### 4. Harness env contract is mandatory wiring, not a separate product capability

**Choice:** Document env expectations in design + light requirements inside relevant specs / error-classification cross-cutting notes; do not create a tenth “harness” capability.

**Rationale:** Env vars are integration plumbing shared by all scenarios. Source of truth for names/values: reference harness table + demo HTML.


| Concern             | Default env (drivers MUST read or remap)                                                                               |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| Connection URL      | `TEST_ENDPOINTS_OVERRIDE` (Part A: `http(s)://arangodb.local`; Part B: `http(s)://127.0.0.1:17001`)                    |
| Auth                | `TEST_AUTHENTICATION_OVERRIDE` / `TEST_AUTHENTICATION` — `basic:<user>:<password>`                                     |
| Ingress Host header | `TEST_INGRESS_HOST` (required for Toxiproxy IP URLs → `arangodb.local`)                                                |
| Docker net / mounts | `TEST_NET_OVERRIDE`, `K8S_TEST_DOCKER_EXTRA_ARGS`                                                                      |
| Toxiproxy           | `TOXIPROXY_LISTEN_PORT` / `TOXIPROXY_ADMIN_PORT`, `TOXIPROXY_UPSTREAM` / `TOXIPROXY_PROXY_NAME`, `TEST_TOXIPROXY_ADMIN`, `TEST_TOXIPROXY_PROXY` |


Drivers MAY map these environment variables through `K8S_TEST_*_ENV` or equivalent wrapper scripts, provided the semantics remain unchanged.

### 5. Category-based acceptance; zero during-fault failures allowed where documented

**Choice:** Accept any error in the allowed category set for the phase. For coordinator restart during active workload and coordinator kill during insert, `failuresDuring = 0` is a valid pass when recovery succeeds.

**Rationale:** Matches reference and real CI timing variance. Exact string matching is forbidden as the sole acceptance rule.

### 6. Universal reject conditions

**Choice:** Across all capabilities, tests MUST fail on: process panic/crash, hang past the scenario timeout, category F during a fault window, failure to recover after the fault condition has been removed (where recovery is required), and cursor “success” completing as if no kill occurred when interrupt is required.

## Risks / Trade-offs

- **[Drift between OpenSpec and reference]** → Specs cite scenario IDs/names; update both in the same PR when behavior changes; prefer shortening specs over copying tables.
- **[Over-constraining other drivers]** → Keep APIs abstract; allow language-idiomatic timeouts/clients as long as categories and recovery hold.
- **[HTTP/1 vs HTTP/2 message differences]** → Classify by category; document that different text for the same fault is expected.
- **[Partial suite adoption]** → Capability split allows phased implementation; error-classification SHOULD land early because other capabilities depend on it.
- **[Go-only observability gaps (e.g. category D via decode error)]** → Specs prefer status + Content-Type when exposed; allow equivalent observable symptoms when the driver cannot surface headers.

## Migration Plan

1. Land this OpenSpec change in the Go Driver repository as the authoritative specification for cross-driver resiliency behavior.
2. Archive into `openspec/specs/` when approved (no code migration required for Go if already compliant — it remains the reference implementation on `run-driver-tests.sh` / `toxiproxy.sh`).
3. Other official driver repositories consume the archived specifications together with the reference documentation, integrate with the shared test infrastructure, and implement only language-specific wrappers where required.
4. Rollback: revert/archive-revert of OpenSpec artifacts only; no cluster or product rollback.

## Open Questions

- Whether other driver repos will vendor these specs, link to this repo, or mirror after archive (process, not behavior).
- Whether optional TLS Toxiproxy modes should later become a separate capability or remain “optional modes” of Part B (currently optional / non-blocking for the contract).

