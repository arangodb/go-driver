# Change Log

## [master](https://github.com/arangodb/go-driver/tree/master) (N/A)
- Vector index: ArangoDB 3.12.10 support — optional/scaling `nLists` (`*VectorNLists`; use `NewVectorNLists(n)` for a fixed count), `numberOfDocsPerCentroid`, factory `{}` placeholder, per-shard `shards`/`resolvedNLists` via `IndexesWithOptions` (`withHidden`), `fields`/`storedValues` on vector responses, and `ErrQueryVectorIndexNotReady` (1555).
- Tests: run v3 integration jobs against `arangodb/core-preview:4.0-nightly` with Starter `0.20.0-preview-16` (4.0 server-only image). v2 stays on `enterprise-preview:latest`.
- Replication: remove APIs gone in ArangoDB 3.12.10+ (applier methods, `StartReplicationSync`, `GetReplicationServerId`, and related types); see `MIGRATION.md`. WAL server identity type renamed to `ReplicationServer`.
- Switch to Go 1.25.13 to fix standard library security issues (GO-2026-6218, GO-2026-6090, GO-2026-5972, GO-2026-5026)
- Switch to Go 1.25.12 to fix Encrypted Client Hello privacy leak in crypto/tls (GO-2026-5856)
- Replication: stop using DBserver forwarding for inventory and logger-state (server allows it only for batch/dump); LoggerState is not supported on Coordinators
- Switch to Go 1.25.11 to fix security issues in the standard library (GO-2026-5039, GO-2026-5037)
- v3 module baseline: introduced `github.com/arangodb/go-driver/v3`, aligned with ArangoDB 4.0 API removals/field updates, and added v3 test suite wiring.
- Build/test infrastructure: added v3 make/CI support, including v3 test targets and CI image/release matrix updates.

