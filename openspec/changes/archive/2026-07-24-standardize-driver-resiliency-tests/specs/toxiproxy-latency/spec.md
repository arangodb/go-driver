## ADDED Requirements

### Requirement: High latency completes successfully but slower

The driver SHALL successfully perform a version operation when a moderate Toxiproxy latency toxic (reference: 2000 ms) is applied, provided the client timeout exceeds the injected latency. Steps MUST follow `HighLatency` (#4) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** A baseline version operation completes successfully. After applying a 2000 ms latency toxic, a version operation also succeeds using a client timeout greater than the injected latency. The observed elapsed time SHOULD reflect approximately the additional network latency.

**Acceptance criteria:** The version operation succeeds under the injected latency. The observed duration is measurably greater than the baseline and consistent with the injected delay. The operation MUST NOT hang beyond the scenario timeout or unexpected process termination. No error-category validation applies because successful completion is expected. The scenario SHALL run for HTTP/1 and HTTP/2 when supported.

#### Scenario: Version succeeds under high latency

- **WHEN** a 2000 ms latency toxic is applied and a version request is issued with a sufficient client timeout
- **THEN** the request succeeds and completes more slowly than the baseline

### Requirement: Extreme latency surfaces a client timeout

The driver SHALL fail a version request with category B when injected latency exceeds the client timeout, then SHALL successfully perform a version operation on the same client after the latency toxic is removed. Steps MUST follow `ExtremeLatency` (#5) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** A version operation fails when latency (reference: 30 s) exceeds the client timeout (reference: 10 s). The error is category B. After the latency toxic is removed, a subsequent version operation on the same client succeeds.

**Acceptance criteria:** During-fault failure is category B; recovery succeeds after the toxic is removed; the operation MUST NOT hang or unexpectedly terminate; category F MUST NOT occur.

#### Scenario: Extreme latency times out then recovers

- **WHEN** extreme latency is applied with a shorter client timeout
- **THEN** version fails with category B and succeeds after the latency toxic is removed

### Requirement: Removing latency restores faster responses

The driver SHALL successfully perform a version operation after a Toxiproxy latency toxic is removed, with the observed response time improving relative to requests made while the toxic was active. Steps MUST follow `LatencyRemoved` (#6) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** Latency toxic applied then removed; subsequent version succeeds without the prior delay penalty.

**Acceptance criteria:** Post-removal version succeeds; timing improves per reference checkpoints; no hang or unexpected process termination.

#### Scenario: Latency removed

- **WHEN** a latency toxic is removed after a slowed successful call
- **THEN** a subsequent version request succeeds without the injected latency effect

### Requirement: Context timeout fails quickly under high latency

The driver SHALL fail a version operation with category B when a client/context timeout is shorter than an injected Toxiproxy latency. Steps MUST follow **ContextTimeout** (#7) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** A latency toxic greater than the client/context timeout is applied. The version operation fails because the client deadline expires rather than waiting for the full injected latency.

**Acceptance criteria:** The failure is category B. The request fails within the reference fail-fast bound. The driver does not hang or unexpectedly terminate.

#### Scenario: Short context deadline under latency

- **WHEN** injected latency exceeds the client/context timeout
- **THEN** the version operation fails with category B before the injected latency fully elapses

### Requirement: HTTP/1 response-header timeout fails under downstream latency

Where the driver supports configuring an HTTP/1 response-header timeout, it SHALL fail a version operation with category B when downstream latency exceeds that timeout. Steps MUST follow **ServerTimeout** (#8) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** A downstream latency toxic exceeds the configured response-header timeout, causing a category B timeout failure.

**Acceptance criteria:** The failure is category B. The driver does not hang or unexpectedly terminate. Drivers that do not support configuring a response-header timeout SHALL document the limitation and skip this scenario. HTTP/2 is not applicable.

#### Scenario: HTTP/1 response-header timeout

- **WHEN** downstream latency exceeds the configured HTTP/1 response-header timeout
- **THEN** the version operation fails with category B

