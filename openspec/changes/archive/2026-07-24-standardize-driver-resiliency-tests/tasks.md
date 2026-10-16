## 1. Spec coherence and reference linkage

- [x] 1.1 Verify each Part A scenario (#0–#8) and Part B scenario (#1–#14) in `driver-resiliency-reference.md` maps to exactly one OpenSpec capability specification
- [x] 1.2 Add a short Scenario → Capability Mapping table in `driver-resiliency-reference.md` linking scenario groups to `openspec` capability names without duplicating requirements
- [x] 1.3 Confirm environment contract in specs/design matches the shared harness documentation
- [x] 1.4 Verify every requirement in each capability spec is traceable back to an existing scenario in driver-resiliency-reference.md

## 2. Error classification (Categories A–F)

- [x] 2.1 Document error classification requirements (Categories A–F) in `error-classification` — meanings, phase/scenario acceptance, C vs D, universal rejects
- [x] 2.2 Verify the Go Driver reference implementation conforms to the specification — helpers in `v2/tests/network_fault_error_util_test.go` map to A–F; HTTP/1 vs HTTP/2 dual coverage/skip rules match specs; scenarios that allow `failuresDuring = 0` are not over-asserted

## 3. Capability gap check — Kubernetes resiliency

- [x] 3.1 Gap-check Go Part A against `load-balancing` and `ingress-resiliency` acceptance criteria
- [x] 3.2 Gap-check Go Part A against `coordinator-resiliency` (idle, active restart, kill during read/insert/iteration)
- [x] 3.3 Record any deviations between the Go Driver reference implementation and the specification; fix only clear acceptance mismatches (no new product features)
  - Deviations: none that require code changes. Non-blocking notes: Go remaps `TEST_ENDPOINTS_OVERRIDE`→`TEST_ENDPOINTS` via Makefile; coordinator idle baseline uses `prepareResiliencyClient`/`waitForClusterStable` rather than a single explicit pre-fault `Version()` call; internal `markIngressReady` naming is reused for coordinator recovery; kill-during-read threshold is 1 doc and kill-during-iteration is 30 docs (aligned with reference).

## 4. Capability gap check — Toxiproxy

- [x] 4.1 Gap-check Toxiproxy #1–#3 against `toxiproxy-connection-interruption`
- [x] 4.2 Gap-check Toxiproxy #4–#8 against `toxiproxy-latency`
- [x] 4.3 Gap-check Toxiproxy #9–#10 against `toxiproxy-packet-loss`
- [x] 4.4 Gap-check Toxiproxy #11–#12 against `toxiproxy-streaming-operations`
- [x] 4.5 Gap-check Toxiproxy #13–#14 against `toxiproxy-write-operations` (unknown write/commit outcome)
  - Fixes applied: corrected ExtremeLatency expected behavior/scenario text in delta+main `toxiproxy-latency` (had LatencyRemoved content); removed duplicate ContextTimeout scenario; Go `TestToxiproxy_ExtremeLatency` now asserts post-toxic recovery.

## 5. Cross-driver handoff package

- [x] 5.1 Draft a one-page implementer checklist pointing to OpenSpec capabilities + reference doc + demo HTML env contract (no kubectl/Go helper dumps)
  - Added `deploy/kubernetes/documentation/driver-resiliency-implementer-checklist.md`; linked from reference "How to use"; removed Go helper-name table from the reference checklist section.
- [x] 5.2 List recommended implementation order for other drivers: `error-classification` → harness env wiring → load-balancing → ingress → coordinator → Toxiproxy subsets
  - Included in the implementer checklist "Recommended implementation order" section.
- [x] 5.3 Run `openspec validate --change standardize-driver-resiliency-tests` and resolve any schema issues before archive
- [x] 5.4 Verify all capability examples and acceptance criteria are language-independent (no Go-specific APIs, helper names, or implementation details).
  - Removed Go-only `invalid character '<'` wording (kept language-neutral decode/HTML symptom for category D); replaced “panic” with “unexpected process termination” across capability specs. Re-validated change + main specs.
