## Why

The Go Driver already implements a comprehensive resiliency test suite using the shared Kubernetes and Toxiproxy test infrastructure.

To enable consistent resiliency testing across all official ArangoDB drivers, a language-independent OpenSpec is needed so other drivers can implement the same behaviors and acceptance criteria without relying on Go-specific implementation details or duplicating Go-specific test logic.

This proposal codifies the existing Go Driver resiliency behavior into capability-oriented specifications that serve as the shared behavioral contract for all official ArangoDB drivers.

## What Changes

- Introduce OpenSpec capabilities that standardize resiliency and Toxiproxy network-fault **behaviors** across official ArangoDB drivers (Go, Java, JavaScript, Python, and others).
- Organize requirements by logical capability (ingress, load balancing, coordinators, Toxiproxy fault classes, error classification) rather than one monolithic spec.
- Treat `deploy/kubernetes/documentation/driver-resiliency-reference.md` as the authoritative step-level reference; specs state requirements, scenarios, expected behavior, and acceptance criteria only.
- Treat the shared harness environment contract (documented in   
`deploy/kubernetes/documentation/driver-k8s-shared-infra-demo.html` and   
`deploy/kubernetes/documentation/driver-resiliency-reference.md`) as the interface that other drivers integrate with.
- The OpenSpec defines behavioral requirements only and does not duplicate Kubernetes commands, runner scripts, or implementation details.
- Use the Go Driver v2 suite (`resiliency` / `toxiproxy` build tags) as the reference implementation while keeping specification language driver-agnostic.
- **No new product functionality** — this change documents and standardizes existing test behaviors; it does not add driver APIs or cluster features.

## Capabilities

### New Capabilities

- `ingress-resiliency`: Behavior when ingress fails over or restarts (idle vs active workload), including recovery on the same or fresh clients.
- `load-balancing`: Observational expectations for coordinator distribution through ingress (no fault injection).
- `coordinator-resiliency`: Behavior when coordinators restart or are killed during idle, active version probes, cursor reads, inserts, and cursor iteration.
- `toxiproxy-connection-interruption`: Abrupt TCP close, proxy disable, and connection-reset faults with fail-then-recover expectations.
- `toxiproxy-latency`: High/extreme latency, latency removal, client context timeout, and server/header timeout behaviors.
- `toxiproxy-packet-loss`: Partial and full packet-loss faults and acceptable success/failure mixes.
- `toxiproxy-streaming-operations`: Disconnect during cursor iteration and during query startup.
- `toxiproxy-write-operations`: Disconnect during document insert and transaction commit (outcome unknown).
- `error-classification`: Language-agnostic categories A–F, phase-based acceptance, HTTP/1 vs HTTP/2 classification rules, and reject conditions (panic, hang, category F during faults).

### Modified Capabilities

- (none — no existing specs under `openspec/specs/`)

## Impact

- **Specs / process:** New capability specs under this change; after archive, main specs become the shared contract for all official drivers.
- **Go driver:** Reference implementation already exists; impact is alignment checks and any gaps vs the standardized acceptance rules (no intentional feature work).
- **Other drivers**: Implement the same scenarios against the shared Kubernetes/Toxiproxy test harness using the reference documentation for execution steps and the OpenSpec for behavioral requirements and acceptance criteria.
- **Shared infrastructure:** The existing shared Kubernetes and Toxiproxy test infrastructure (including `run-driver-tests.sh`, `test/toxiproxy.sh`, and the documented environment contract) serves as the common execution environment for all official drivers. The Go Driver already integrates with this infrastructure and acts as the reference implementation. Other drivers are expected to integrate with the same infrastructure while implementing the OpenSpec requirements.
- **Out of scope for this change:** Duplicating kubectl/Make recipes, Go helper code, CI log dumps, or language-specific assertion APIs inside the specs.
- This proposal standardizes existing behavior and serves as the basis for cross-driver implementation parity rather than introducing new functionality.

