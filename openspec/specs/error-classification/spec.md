## Purpose

Define language-agnostic error categories A–F, phase- and scenario-specific acceptance rules, C vs D classification guidance, and universal reject conditions for resiliency and Toxiproxy suites.

## Requirements

### Requirement: Failures are classified into language-agnostic categories A–F

Every official driver resiliency/Toxiproxy suite SHALL classify observed failures into these categories (definitions are normative and correspond to the categories described in `driver-resiliency-reference.md`):


| Category | Meaning                                                                                     |
| -------- | ------------------------------------------------------------------------------------------- |
| **A**    | Connection lost — TCP/HTTP transport broken before a valid HTTP response                    |
| **B**    | Timeout — client or transport gave up waiting                                               |
| **C**    | Gateway HTTP error — HTTP 502/503/504 with JSON body                                        |
| **D**    | HTTP error status with a non-JSON body and/or non-JSON `Content-Type` (typical 502/503/504) |
| **E**    | Cursor gone — HTTP 404/409/410 with JSON body on cursor APIs                                |
| **F**    | Application error — auth, validation, or normal API not-found                               |


**Expected behavior:** Tests classify failures using categories (or equivalent language-specific predicates), rather than relying on a single exact error string. HTTP/1 and HTTP/2 MAY produce different message text for the same fault; category match is sufficient.

**Acceptance criteria:** Tests assert allowed categories per scenario phase; exact-string-only matching is insufficient as the sole rule.

#### Scenario: Same fault different protocol text

- **WHEN** the same network fault is observed under HTTP/1 and HTTP/2 with different error strings
- **THEN** both are accepted if they map to the same allowed category for that phase

#### Scenario: Category F never accepted during fault windows

- **WHEN** an error during an injected fault window is category F
- **THEN** the test MUST fail

### Requirement: Acceptance is phase- and scenario-specific

Drivers SHALL apply the master scenario→category mapping from the reference (Kubernetes Part A and Toxiproxy Part B). Other capability specifications reference the allowed category sets defined here; this capability is the normative classification specification.

**Summary of expected behavior by scenario:**

- **Active ingress/coordinator outages:** Categories A–D are typically accepted during the fault window.
- **Cursor interruption:** Interrupt-phase errors accept categories A–E.
- **Dead-cursor resume:** Only categories A or E are accepted; gateway errors (502/503/504) are not valid dead-cursor resume outcomes.
- **Dead-cursor close:** Category E is the primary expected outcome, although categories A–D are also acceptable.
- **Idle restart scenarios:** No failures are expected during the fault window.
- **Coordinator restart and insert scenarios:** These scenarios MAY complete without any observed failures during the injected fault. When no failures occur, the scenario still passes provided all required recovery criteria are satisfied. If failures do occur, they MUST belong to the categories defined for that scenario.
- **Toxiproxy transport faults:** Connection interruption, streaming interruption, and write interruption primarily produce category A. Latency, timeout, and full packet-loss scenarios primarily produce category B. Partial packet loss MAY produce either category A or B. Categories C–E are not expected for pure transport faults.

**Acceptance criteria:** Each automated scenario documents which categories are accepted at which checkpoint; violations fail the test.

#### Scenario: Cursor interrupt vs dead cursor resume

- **WHEN** coordinators are killed during cursor use
- **THEN** interrupt-phase errors accept A–E as listed for that scenario, and resume-phase errors follow the dead-cursor set (A or E), not unrestricted gateway success

#### Scenario: Optional during-fault failures

- **WHEN** no failures are observed during an injected fault in a scenario where this is permitted
- **THEN** the scenario passes if all required recovery criteria are satisfied

### Requirement: Prefer status and Content-Type when distinguishing C vs D

When the driver exposes HTTP status and `Content-Type`, category **C** MUST be used for 502/503/504 with JSON bodies, and category **D** for those statuses with non-JSON body and/or non-JSON `Content-Type`. If the driver cannot expose HTTP status or Content-Type and instead surfaces an equivalent decode or transport symptom, that observable MAY be classified as category D as described in the reference documentation.

**Expected behavior:** Classification prefers protocol metadata over substring matching of HTML.

**Acceptance criteria:** C and D are not conflated when headers/status are available; when status/`Content-Type` are not exposed, an equivalent decode/transport symptom of a non-JSON (e.g. HTML) body — such as a JSON parser rejecting a leading `'<'` — remains an allowed **D** observable.

#### Scenario: JSON gateway error is category C

- **WHEN** the fault response is HTTP 503 with `Content-Type: application/json`
- **THEN** the error is classified as category C

#### Scenario: HTML gateway error is category D

- **WHEN** the fault response is HTTP 502/503 with non-JSON `Content-Type` or non-JSON body
- **THEN** the error is classified as category D

### Requirement: Universal reject conditions across resiliency suites

All resiliency and Toxiproxy scenarios SHALL fail the test when any of the following occur: unexpected driver or test process termination; hang past the scenario/test timeout; missing required recovery after the fault is cleared; treating an interrupted cursor/query as a successful full completion; category F during a fault window.

**Expected behavior:** Failures are clean, classified, and bounded in time; recovery uses the same client where the reference requires it.

**Acceptance criteria:** CI marks the scenario failed on any universal reject condition.

#### Scenario: Hang is a failure

- **WHEN** a request or workload does not return within the scenario timeout under an injected fault
- **THEN** the test fails (hang), even if no wrong category was observed

#### Scenario: Recovery required after clear

- **WHEN** a scenario requires post-fault recovery and the fault has been cleared
- **THEN** a subsequent operation on the required client MUST succeed within the reference budget
