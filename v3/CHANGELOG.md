# Change Log

## [master](https://github.com/arangodb/go-driver/tree/master) (N/A)
- Switch to Go 1.25.11 to fix security issues in the standard library (GO-2026-5039, GO-2026-5037)
- v3 module baseline: introduced `github.com/arangodb/go-driver/v3`, aligned with ArangoDB 4.0 API removals/field updates, and added v3 test suite wiring.
- ArangoDB 4.0 removals: removed legacy APIs that fail at runtime against ArangoDB 4.0, including `ClusterStatistics`, `GetMetrics`, `HandleAdminVersion`, `ExecuteAdminScript`, `LoggerFirstTick`, `LoggerTickRange`, `CreateUserDefinedFunction`, `DeleteUserDefinedFunction`, `GetUserDefinedFunctions`, `TransactionJS`, `ReloadRoutingTable`, all Foxx service methods, and task methods. See `MIGRATION.md` for replacements and details.
- Build/test infrastructure: added v3 make/CI support, including v3 test targets and CI image/release matrix updates.

