# Change Log

## [master](https://github.com/arangodb/go-driver/tree/master) (N/A)
- Replication: removed single-single replication applier API (`GetApplierConfig`, `UpdateApplierConfig`, `ApplierStart`, `ApplierStop`, `GetApplierState`, `MakeFollower`) and related types. Those endpoints were removed in ArangoDB 3.12.10 for a security issue and are absent in 4.0. WAL server identity type renamed to `ReplicationServer`.
- Switch to Go 1.25.12 to fix Encrypted Client Hello privacy leak in crypto/tls (GO-2026-5856)
- Replication: stop using DBserver forwarding for inventory and logger-state (server allows it only for batch/dump); LoggerState is not supported on Coordinators
- Switch to Go 1.25.11 to fix security issues in the standard library (GO-2026-5039, GO-2026-5037)
- v3 module baseline: introduced `github.com/arangodb/go-driver/v3`, aligned with ArangoDB 4.0 API removals/field updates, and added v3 test suite wiring.
- Build/test infrastructure: added v3 make/CI support, including v3 test targets and CI image/release matrix updates.

