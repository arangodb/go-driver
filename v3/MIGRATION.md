# Migration Guide: v2 -> v3

This guide lists identifiers removed in `v3` and the recommended replacements
or actions to take when migrating from `v2`.

## Removed Packages/Files

The following packages/files are removed in `v3`:

- `client_foxx`
- `client_foxx_impl`
- `database_transactionsjs_test`
- `tasks`
- `tasks_impl`
- `tasks_test`

## Removed Methods and Replacements

- `ClusterStatistics` (`/_admin/cluster/statistics`) -> use `Metrics` (`/_admin/metrics`)
- `GetMetrics` (`/_admin/metrics/v2`) -> use `Metrics` (`/_admin/metrics`)
- `HandleAdminVersion` (`/_admin/version`) -> use `Version` or `VersionWithOptions` (`/_api/version`)
- `ExecuteAdminScript` (`/_admin/execute`) -> removed with no direct replacement
- `GetUserDefinedFunctions` (`/_api/aqlfunction`) -> removed in ArangoDB 4.0
- `CreateUserDefinedFunction` (`/_api/aqlfunction`) -> removed in ArangoDB 4.0
- `DeleteUserDefinedFunction` (`/_api/aqlfunction/{name}`) -> removed in ArangoDB 4.0
- `TransactionJS` (`/_api/transaction`) -> removed in ArangoDB 4.0
- `ReloadRoutingTable` (`/_admin/routing/reload`) -> removed because Action/Foxx microservice route reloading is removed in ArangoDB 4.0

### Replication endpoints removed in ArangoDB 3.12.10+

These endpoints are removed in ArangoDB 3.12.10+ (applier endpoints for a security
issue inherent to the design; remaining single-single replication endpoints as part
of the same cleanup). Leader-follower was deprecated since 3.9; Active Failover was
deprecated in 3.11 and removed in 3.12.

- `GetApplierConfig` (`/_api/replication/applier-config`) -> removed with no direct replacement
- `UpdateApplierConfig` (`/_api/replication/applier-config`) -> removed with no direct replacement
- `ApplierStart` (`/_api/replication/applier-start`) -> removed with no direct replacement
- `ApplierStop` (`/_api/replication/applier-stop`) -> removed with no direct replacement
- `GetApplierState` (`/_api/replication/applier-state`) -> removed with no direct replacement
- `MakeFollower` (`/_api/replication/make-follower`) -> removed with no direct replacement
- `StartReplicationSync` (`/_api/replication/sync`) -> removed with no direct replacement
- `LoggerFirstTick` (`/_api/replication/logger-first-tick`) -> use `GetWALTail` (`/_api/wal/tail`)
- `LoggerTickRange` (`/_api/replication/logger-tick-ranges`) -> use `GetWALTail` (`/_api/wal/tail`)
- `GetReplicationServerId` (`/_api/replication/server-id`) -> removed with no direct replacement

## Removed Fields and Replacements

- `CollectionDocumentCreateOptions.Overwrite` -> use `OverwriteMode` (`overwriteMode`)
  - ArangoDB 4.0 rejects the `overwrite` query parameter on document create.
- `Health` response (`/_admin/cluster/health`):
  - removed fields: `LastHeartbeatAcked`, `LastHeartbeatSent`, `LastHeartbeatStatus`
- `GetServerStatus` response (`/_admin/status`):
  - removed fields: `Mode`, `FoxxApi`, `WriteOpsEnabled`, `CoordinatorInfo`
- `ServerRole` response (`/_admin/server/role`):
  - removed enum values: `SingleActive`, `SinglePassive`
- `EngineInfo.EngineType`:
  - removed `mmfiles`
- `EnsureInvertedIndex` (`/_api/index`) `consolidationPolicy`:
  - removed fields: `MinScore`, `SegmentsMin`, `SegmentsMax`, `SegmentsBytesFloor`
  - removed type: `bytes_accum`
- `CreateArangoSearchView` (`/_api/view`) `consolidationPolicy`:
  - removed fields: `MinScore`, `SegmentsMin`, `SegmentsMax`, `SegmentsBytesFloor`

## New Fields in v3

- `GetInventory` (`/_api/replication/inventory`) response adds:
  - `collections.parameters.supportsRBAC`
