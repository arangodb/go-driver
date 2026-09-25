## Purpose

Define driver behavior for Toxiproxy partial and full packet-loss faults, including mixed success/failure acceptance and recovery after the toxic is cleared.

## Requirements

### Requirement: Partial packet loss yields a mix of successes and connection or timeout failures

The driver SHALL run a burst of version requests under intermittent `reset_peer` toxics (reference: 40% toxicity, 40 requests) and accept a mix of successes and failures. Steps MUST follow **PartialPacketLoss** (#9) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** Some requests succeed while others fail due to intermittent connection interruptions. Failed requests belong to category A or B. The scenario does not require either all requests to fail or all requests to succeed.

**Acceptance criteria:** At least one request succeeds and at least one request fails. Every failure belongs to category A or B. After the toxic is removed, subsequent version requests succeed. The driver does not hang or unexpectedly terminate, and category F is rejected. This scenario is exercised on HTTP/1. Drivers that cannot meaningfully model per-request packet loss on HTTP/2 MAY skip the HTTP/2 variant.

#### Scenario: Intermittent packet loss

- **WHEN** intermittent `reset_peer` toxics are active during multiple version requests
- **THEN** the results include both successes and failures, and every failure belongs to category A or B

#### Scenario: Recovery after packet loss

- **WHEN** the intermittent packet-loss toxic is removed
- **THEN** subsequent version requests succeed

### Requirement: Full packet loss fails then recovers

The driver SHALL fail a version operation when Toxiproxy applies a timeout toxic that simulates complete packet loss. Steps MUST follow **FullPacketLoss** (#10) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** While the timeout toxic is active, the version operation does not complete successfully. The failure is either a timeout (category B) or an equivalent transport failure caused by the connection becoming unusable (category A). After the toxic is removed, the same client successfully performs the version operation again.

**Acceptance criteria:** The fault error belongs to category A or B. Recovery succeeds on the same client after the toxic is removed. The request returns within the reference timeout budget (that is, it does not hang beyond the allowed safety timeout). The driver does not unexpectedly terminate, and category F is rejected.

#### Scenario: Full packet loss then recover

- **WHEN** a full packet-loss (timeout) toxic is applied and the version operation is performed
- **THEN** the operation fails with category A or B and succeeds after the toxic is removed
