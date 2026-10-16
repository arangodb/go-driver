## Purpose

Define driver behavior when Toxiproxy disconnects during cursor iteration or query startup, requiring category A failures and recovery of new operations after the proxy is restored.

## Requirements

### Requirement: Disconnect during cursor iteration fails the next read

The driver SHALL fail the next cursor read with category A when the Toxiproxy proxy is disabled after a partial number of documents have been read (reference: after 5 docs). Steps MUST follow **DisconnectDuringCursorIteration** (#11) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** Cursor streaming begins successfully; proxy disabled mid-iteration; subsequent read fails with connection-loss (category A); after proxy re-enable, the driver can perform new healthy operations (per reference recovery checks). Mid-stream cancel/connection errors that are still category A are acceptable.

**Acceptance criteria:** The next cursor read fails with category A; the cursor does not continue successfully across the disconnect; after proxy re-enable the driver performs new operations successfully; no hang or unexpected process termination; category F rejected. Run HTTP/1 and HTTP/2 when supported.

#### Scenario: Mid-cursor proxy disable

- **WHEN** the proxy is disabled after partial cursor consumption
- **THEN** the next cursor read fails with category A

### Requirement: Disconnect during query startup fails the query

The driver SHALL fail query startup (`POST /_api/cursor` or driver `Query` equivalent) with category A when the proxy is disabled while the query is starting. Steps MUST follow **DisconnectDuringQueryExecution** (#12).

**Expected behavior:** Fault overlaps query creation; the operation fails with connection-loss; HTTP/1 vs HTTP/2 message text may differ (e.g. EOF vs unexpected EOF) while category remains A.

**Acceptance criteria:** Query fails with category A; no hang or unexpected process termination; recovery of new operations after proxy re-enable per reference; category F rejected.

#### Scenario: Query start interrupted

- **WHEN** the Toxiproxy proxy is disabled during query startup before a cursor is returned
- **THEN** the query fails with category A and no usable cursor is returned
