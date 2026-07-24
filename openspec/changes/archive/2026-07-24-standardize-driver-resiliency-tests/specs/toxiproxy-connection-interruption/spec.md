## ADDED Requirements

### Requirement: Abrupt TCP close on a live connection fails then recovers

The driver SHALL fail a version request with category A when Toxiproxy injects an upstream `reset_peer` toxic on an established connection, then SHALL succeed on the same client after the toxic is removed. Steps MUST follow Toxiproxy **AbruptTCPConnectionClose** (#1) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** Baseline version succeeds (keep-alive established); after `reset_peer` upstream, version fails quickly with a connection-loss error; after toxic removal, version succeeds again.

**Acceptance criteria:** Fault error is category A; recovery succeeds on the same client; fail-fast (no hang until the safety deadline); no unexpected process termination; category F rejected. Run HTTP/1 and HTTP/2 when supported.

#### Scenario: Reset peer then recover

- **WHEN** `reset_peer` is applied upstream on the Toxiproxy proxy and version is called
- **THEN** the call fails with category A and a later version call succeeds after the toxic is removed

### Requirement: Network disconnect via disabled proxy fails then recovers

The driver SHALL fail a version request with **category A** when the Toxiproxy proxy is disabled, then SHALL successfully perform a version operation on the same client after the proxy is re-enabled. Steps MUST follow **NetworkDisconnect** (#2) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** A baseline version operation succeeds. While the proxy is disabled, a version request fails with a transport-level connection error (category A). After the proxy is re-enabled, the same client successfully performs a version operation again.

**Acceptance criteria:** The during-fault error MUST be category A. Recovery on the same client MUST succeed after the proxy is re-enabled. The operation MUST NOT hang or unexpectedly terminate. Category F MUST NOT occur.

#### Scenario: Proxy disable then recover

- **WHEN** the Toxiproxy proxy is disabled during a version call
- **THEN** the call fails with category A and succeeds after the proxy is re-enabled

### Requirement: Downstream connection reset fails then recovers

The driver SHALL fail a version request with category A when Toxiproxy applies a `reset_peer` toxic on the downstream path, then SHALL successfully perform a version operation on the same client after the toxic is removed. Steps MUST follow `ConnectionResetByPeer` (#3) in `deploy/kubernetes/documentation/driver-resiliency-reference.md`.

**Expected behavior:** A baseline version operation succeeds. While a downstream `reset_peer` toxic is active, a version request fails with a category A transport error. HTTP/1 and HTTP/2 MAY surface different messages (for example `EOF` or `unexpected EOF`), but both classify as category A. After the toxic is removed, the same client successfully performs a version operation again.

**Acceptance criteria:** The during-fault error MUST be category A. Recovery on the same client MUST succeed after the toxic is removed. The operation MUST NOT hang or unexpectedly terminate. Category F MUST NOT occur.

#### Scenario: Downstream reset then recover

- **WHEN** a downstream `reset_peer` toxic is active and a version request is issued
- **THEN** the request fails with category A, and a later version request succeeds after the toxic is removed

### Requirement: Toxiproxy tests use the shared harness proxy contract

Drivers SHALL send traffic to the Toxiproxy listen endpoint from the harness (`TEST_ENDPOINTS_OVERRIDE` → listen port) and MUST send `Host: TEST_INGRESS_HOST` (`arangodb.local`) on every request. Admin control uses `TEST_TOXIPROXY_ADMIN` / `TEST_TOXIPROXY_PROXY` (see demo HTML and reference harness table).

**Expected behavior:** Driver → Toxiproxy listen → ingress → coordinator. Without the Host header, ingress will not route correctly.

**Acceptance criteria:** Connection scenarios only pass when wired through the proxy with correct Host header; faults are injected via the admin API, not by changing the driver URL mid-test except as the reference describes.

#### Scenario: Host header required through Toxiproxy

- **WHEN** the driver connects via the Toxiproxy listen address for connection-interruption tests
- **THEN** every request includes the ingress Host header from `TEST_INGRESS_HOST`

