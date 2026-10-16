## ADDED Requirements

### Requirement: Disconnect during document insert fails cleanly with unknown write outcome

The driver SHALL fail document create with category A when the Toxiproxy proxy is disabled during the insert, and MUST treat the write outcome as unknown (neither assert committed nor assert absent unless the reference requires a specific follow-up check). Steps MUST follow **DisconnectDuringInsert** (#13) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** The insert request is in flight when the proxy is disabled. The create operation fails with a category A transport error. The final commit state of the interrupted write is intentionally unspecified.

**Acceptance criteria:** Error is category A; no hang or unexpected process termination; category F rejected; recovery after clear per reference. HTTP/1 and HTTP/2 when supported.

#### Scenario: Insert interrupted by proxy disable

- **WHEN** the proxy is disabled during document create
- **THEN** the create fails with category A and the test does not assert a definitive write success or failure for that document’s durability

### Requirement: Disconnect during transaction commit fails cleanly with unknown commit outcome

The driver SHALL fail transaction commit with category A when the Toxiproxy proxy is disabled while the commit request is in flight, and MUST treat the transaction outcome as unknown (neither assert committed nor assert aborted unless the reference requires a specific follow-up check). Steps MUST follow **DisconnectDuringTransactionCommit** (#14) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** A transaction containing writes is prepared successfully; the commit request is sent; the proxy is disabled while the commit is in flight; commit fails with a connection-loss error. After the proxy is re-enabled, new operations succeed. Tests MUST NOT require the interrupted transaction to have either committed or rolled back.

**Acceptance criteria:** Commit fails with category A; the test MUST NOT assert whether the transaction committed or aborted; recovery succeeds after the proxy is restored; no hang or unexpected process termination; category F rejected. Run HTTP/1 and HTTP/2 when supported.

#### Scenario: Transaction commit interrupted by disconnect

- **WHEN** the proxy is disabled while a transaction commit request is in flight
- **THEN** the commit fails with category A, the transaction outcome is treated as unknown, and new operations succeed after recovery

