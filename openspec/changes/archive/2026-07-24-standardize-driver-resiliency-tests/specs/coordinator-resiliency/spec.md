## ADDED Requirements

### Requirement: Coordinator restart while idle recovers on the same client

The driver SHALL successfully perform a version operation after all coordinator pods are deleted and recreated, with no client requests issued during the outage. Steps MUST follow reference scenario **CoordinatorRestartWhileIdle** (Part A #4) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** A version operation succeeds before the fault. No client requests are issued while coordinators are being recreated. Once all coordinators are ready, the same client successfully performs the version operation without requiring reinitialization.

**Acceptance criteria:** The post-recovery version operation succeeds within the reference coordinator-ready budget using the same client. No hang or unexpected driver termination occurs. No client errors are expected during the outage because no requests are issued while coordinators are being recreated.

#### Scenario: Idle client survives full coordinator recreation

- **WHEN** all coordinator pods are deleted while the client is idle
- **THEN** version on the same client succeeds

### Requirement: Coordinator restart during active workload recovers without requiring during-fault failures

The driver SHALL keep a version loop alive while all coordinator pods are deleted and recreated. Recovery is mandatory; during-fault failures are optional. Steps MUST follow **CoordinatorRestartDuringActiveWorkload** (Part A #5).

**Expected behavior:** Single client; `GET /_api/version` every ~100 ms with a bounded per-request timeout. Track successes/failures before, during, and after the fault. Recovery successes are counted only after coordinator readiness has been re-established.

**Acceptance criteria:** `successesBefore ≥ 1`; `successesAfter ≥ 1`; baseline and recovery quality rules defined in the reference MUST hold; failuresDuring MAY be 0.  If `failuresDuring > 0`, every observed error MUST belong to category A, B, C, or D. The workload MUST terminate cleanly after cancellation, with no hang or unexpected process termination, and category F MUST never occur during the fault window.

#### Scenario: Recovery with zero during-fault failures

- **WHEN** the version loop runs across full coordinator recreation and no request overlaps downtime
- **THEN** the scenario passes if recovery successes are recorded and other pass criteria hold

#### Scenario: Recovery with transient during-fault failures

- **WHEN** one or more version requests fail during the coordinator outage
- **THEN** each during-fault error is category A–D and recovery successes still occur

### Requirement: Coordinator kill during cursor read interrupts and marks the cursor unusable

The driver SHALL observe a clean interrupt when coordinators are killed during an active cursor read, then treat resume/close of that cursor according to dead-cursor rules. Steps MUST follow **CoordinatorKillDuringRead** (Part A #6).

**Expected behavior:** The streaming cursor is interrupted after the coordinator kill. Interrupt-phase errors MUST belong to categories A–E as defined for that phase. After recovery, attempting to resume the same cursor accepts only categories A or E; gateway errors (502/503/504) are not valid dead-cursor resume outcomes. Closing the dead cursor primarily produces category E, although categories A–D are also accepted as defined in the reference.

**Acceptance criteria:** The cursor MUST NOT complete successfully as though the coordinator interruption had not occurred. Interrupt and dead-cursor phases match allowed categories; cluster recovers for subsequent healthy operations; no hang or unexpected process termination.

#### Scenario: Cursor interrupt on coordinator kill

- **WHEN** all coordinators are deleted during an open cursor read
- **THEN** the next cursor operation fails with an allowed interrupt category (A–E) and does not pretend the query finished successfully

#### Scenario: Dead cursor after recovery

- **WHEN** the driver resumes or closes the same cursor after cluster recovery
- **THEN** errors match the dead-cursor acceptance set for that phase in the reference (see `error-classification`)

### Requirement: Coordinator kill during insert recovers with optional during-fault failures

The driver SHALL run an insert loop while one coordinator is deleted and SHALL recover afterward. Steps MUST follow **CoordinatorKillDuringInsert** (Part A #7).

**Expected behavior:** An insert workload remains active throughout the fault window. `failuresDuring` MAY be `0`. If during-fault errors occur, they MUST belong to categories A–D. The outcome of an insert interrupted by the coordinator failure MAY be unknown; therefore the suite validates clean failure classification and post-recovery progress rather than the commit outcome of an individual interrupted write.

**Acceptance criteria:** At least one successful insert (or equivalent post-recovery health check as defined in the reference) MUST occur after recovery. During-fault errors, if any, MUST belong only to categories A–D. The workload MUST terminate cleanly after cancellation with no hang or unexpected process termination, and category F MUST never occur during the fault window.

#### Scenario: Insert loop survives coordinator kill

- **WHEN** inserts run while one coordinator pod is deleted
- **THEN** the driver recovers after the fault, and any during-fault errors are only categories A–D

### Requirement: Coordinator kill during cursor iteration interrupts mid-stream

The driver SHALL interrupt cursor iteration after a defined number of documents when all coordinators are killed. Steps MUST follow **CoordinatorKillDuringCursorIteration** (Part A #8).

**Expected behavior:** After reading the reference threshold of documents, all coordinator pods are deleted. The active cursor iteration is interrupted with an allowed interrupt category. After the cluster recovers, resuming or closing the same cursor follows the dead-cursor acceptance rules defined in `error-classification`. Differences in protocol-specific error messages or response bodies (for example between HTTP/1 and HTTP/2) are evaluated by error category rather than exact message text.

**Acceptance criteria:** Cursor iteration MUST NOT complete successfully across the coordinator kill. Interrupt and dead-cursor phases MUST satisfy the allowed error categories defined in `error-classification`. The cluster MUST recover for subsequent healthy operations. The test MUST fail on hangs, unexpected process termination, or category F during the fault window.

#### Scenario: Mid-iteration interrupt

- **WHEN** coordinators are killed after partial cursor consumption
- **THEN** the next iteration step fails with an allowed category and the cursor is not treated as successfully exhausted

#### Scenario: Dead cursor remains unusable after recovery

- **WHEN** the cluster has recovered after the coordinator kill
- **THEN** resuming or closing the same cursor follows the dead-cursor acceptance rules defined in `error-classification`