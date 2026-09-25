## ADDED Requirements

### Requirement: Load balancer coordinator distribution is observable through ingress

The driver SHALL probe coordinator identity through ingress with no fault injected, recording which coordinator answered each request. Steps MUST follow reference scenario **LoadBalancerCoordinatorDistribution** (Part A #0).

**Expected behavior:** Repeated `GET /_admin/status` probes succeed and expose `serverInfo.serverId` (or equivalent). Requests may be served by a single coordinator or distributed across multiple coordinators. The distribution is observational and is not required to follow any specific balancing pattern.

**Acceptance criteria:** Every probe succeeds. The responding coordinator ID for each successful probe is recorded. The scenario MUST NOT fail solely because only one coordinator is observed. No error-category validation applies because no fault is injected.

#### Scenario: Probes succeed and record coordinator IDs

- **WHEN** the driver issues status probes through ingress using the shared and fresh HTTP/1 and HTTP/2 client modes defined by the reference
- **THEN** every probe succeeds and the responding coordinator ID is recorded

#### Scenario: Coordinator distribution is observational

- **WHEN** probes are answered by one or more distinct coordinator IDs
- **THEN** the scenario still passes because ingress routing behavior is implementation-dependent

### Requirement: Load-balancing baseline uses the shared resiliency harness endpoints

Drivers SHALL connect using the Part A harness endpoint and auth contract (`TEST_ENDPOINTS_OVERRIDE`, `TEST_AUTHENTICATION_OVERRIDE` / `TEST_AUTHENTICATION`) as documented in the reference and `deploy/kubernetes/documentation/driver-k8s-shared-infra-demo.html`, remapping names only if semantics are preserved.

**Expected behavior:** Traffic path is driver → ingress (`arangodb.local`) → coordinators. ClientIP session affinity on the coordinator Service does not apply on this out-of-cluster path.

**Acceptance criteria:** Probes use the harness-exported URL and auth; Host header behavior matches the reference when required by the URL form.

#### Scenario: Harness endpoint wiring

- **WHEN** the load-balancing scenario runs under the shared Kubernetes driver-test runner
- **THEN** the driver uses the exported endpoints/auth (or an equivalent remap) to reach the cluster through ingress
