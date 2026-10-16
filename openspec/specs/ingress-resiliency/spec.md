## Purpose

Define driver behavior when ingress fails over a coordinator or restarts (idle vs active workload), including recovery on the same or fresh clients.

## Requirements

### Requirement: Ingress coordinator failover recovers through ingress

The driver SHALL continue to reach the cluster through ingress after one coordinator pod is deleted. Implementation steps and wait budgets MUST follow reference scenario **IngressCoordinatorFailover** (Part A #1) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** A baseline `GET /_admin/status` probe succeeds. After deleting one coordinator pod, the driver retries the probe using fresh clients until a coordinator responds successfully. After operator recovery, the original coordinator count is restored and a final probe succeeds. Whether the responding coordinator ID changes is observational only; either outcome is valid.

**Acceptance criteria:** A probe succeeds within the reference failover budget after the coordinator kill. After operator recovery, the healthy coordinator count matches the pre-fault count and a final probe succeeds. The scenario MUST complete without hangs or unexpected process termination. Transient probe errors observed during the failover window, if any, MUST belong only to categories A–D (see `error-classification`).

#### Scenario: Failover then full recovery

- **WHEN** the driver repeatedly probes `GET /_admin/status` through ingress, one coordinator pod is deleted, and probing continues until success
- **THEN** a probe succeeds after the kill, the deployment recovers to the original coordinator count, and a post-recovery probe succeeds

#### Scenario: HTTP/1 and HTTP/2 coverage

- **WHEN** the driver supports both HTTP/1 and HTTP/2
- **THEN** the failover scenario SHALL be executed for both protocols (skipping HTTP/2 only when unsupported by the server version, as defined in the reference).

### Requirement: Ingress restart while idle recovers on the same client

The driver SHALL successfully perform a version operation on the same client instance after an ingress-nginx restart, while no client requests are issued during the outage. Steps MUST follow reference scenario **IngressRestartWhileIdle** (Part A #2).

**Expected behavior:** A version operation succeeds before the restart. No client requests are issued while ingress is unavailable. After ingress becomes ready again, the same client successfully performs the version operation without requiring reinitialization.

**Acceptance criteria:** Version operations succeed before and after the ingress restart using the same client. The scenario MUST complete without hangs or unexpected process termination. Category F MUST NOT occur. No during-fault errors are expected because no client requests are issued while ingress is restarting.

#### Scenario: Idle client survives ingress restart

- **WHEN** version succeeds, ingress is restarted with no driver traffic during the outage, and ingress becomes ready
- **THEN** version on the same client succeeds after recovery

### Requirement: Ingress restart during active workload tolerates transient errors and recovers

The driver SHALL keep a version-request loop alive while ingress-nginx is restarted. Recovery is mandatory; transient failures during the fault window are permitted. Steps MUST follow **IngressRestartDuringActiveWorkload** (Part A #3).

**Expected behavior:** A single client continuously performs `GET /_api/version` approximately every 100 ms with a bounded per-request timeout. Successes and failures are tracked before, during, and after the ingress restart. Recovery successes are counted only after ingress has become ready again.

**Acceptance criteria:** `successesBefore ≥ 1`; `successesAfter ≥ 1`; baseline and recovery quality rules defined in the reference MUST hold. `failuresDuring` MAY be `0`. If `failuresDuring > 0`, every observed error MUST belong to category A, B, C, or D. The workload MUST terminate cleanly after cancellation, with no hang or unexpected process termination, and category F MUST never occur during the fault window.

#### Scenario: Recovery with zero during-fault failures

- **WHEN** the version loop runs across an ingress-nginx restart and no request overlaps the outage
- **THEN** the scenario passes if recovery successes are recorded and the other acceptance criteria hold

#### Scenario: Recovery with transient during-fault failures

- **WHEN** one or more version requests fail during the ingress-nginx restart
- **THEN** each during-fault error belongs to category A, B, C, or D and recovery successes still occur
