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
- `LoggerFirstTick` (`/_api/replication/logger-first-tick`) -> removed in ArangoDB 4.0
- `LoggerTickRange` (`/_api/replication/logger-tick-ranges`) -> removed in ArangoDB 4.0
- `GetUserDefinedFunctions` (`/_api/aqlfunction`) -> removed in ArangoDB 4.0
- `CreateUserDefinedFunction` (`/_api/aqlfunction`) -> removed in ArangoDB 4.0
- `DeleteUserDefinedFunction` (`/_api/aqlfunction/{name}`) -> removed in ArangoDB 4.0
- `TransactionJS` (`/_api/transaction`) -> removed in ArangoDB 4.0
- `ReloadRoutingTable` (`/_admin/routing/reload`) -> removed because Action/Foxx microservice route reloading is removed in ArangoDB 4.0

The following Foxx service methods are removed because Foxx is removed in ArangoDB 4.0:

- `InstallFoxxService`
- `UninstallFoxxService`
- `ListInstalledFoxxServices`
- `GetInstalledFoxxService`
- `ReplaceFoxxService`
- `UpgradeFoxxService`
- `GetFoxxServiceConfiguration`
- `UpdateFoxxServiceConfiguration`
- `ReplaceFoxxServiceConfiguration`
- `GetFoxxServiceDependencies`
- `UpdateFoxxServiceDependencies`
- `ReplaceFoxxServiceDependencies`
- `GetFoxxServiceScripts`
- `RunFoxxServiceScript`
- `RunFoxxServiceTests`
- `EnableDevelopmentMode`
- `DisableDevelopmentMode`
- `GetFoxxServiceReadme`
- `GetFoxxServiceSwagger`
- `CommitFoxxService`
- `DownloadFoxxServiceBundle`

The following task methods are removed because the `/_api/tasks` API is removed in ArangoDB 4.0:

- `Task`
- `Tasks`
- `CreateTask`
- `CreateTaskWithID`
- `RemoveTask`

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
