# Change Log

## [master](https://github.com/arangodb/go-driver/tree/master) (N/A)
- Replication: stop using DBserver forwarding for inventory and logger-state (server allows it only for batch/dump); LoggerState is not supported on Coordinators
- Switch to Go 1.25.11 to fix security issues in the standard library (GO-2026-5039, GO-2026-5037)
- v3 module baseline: introduced `github.com/arangodb/go-driver/v3`, aligned with ArangoDB 4.0 API removals/field updates, and added v3 test suite wiring.
- Build/test infrastructure: added v3 make/CI support, including v3 test targets and CI image/release matrix updates.

