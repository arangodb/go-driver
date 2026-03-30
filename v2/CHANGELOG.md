# Change Log

## [master](https://github.com/arangodb/go-driver/tree/master) (N/A)
- Add `DontFollowRedirect` fields to HTTP connection configuration
- Switched to Go 1.25.8 to fix a security vulnerability
- QueryProperties: Added slowStreamingQueryThreshold field
- RunningAQLQuery: Added exitCode (slow queries), modificationQuery, and warnings
- QuerySubOptions/ExplainQueryOptions: Added usePlanCache, failOnWarning, fullCount, maxNodesPerCallstack, maxWarningCount, profile, and other missing options
- CollectionDocumentImportOptions: Added overwriteCollectionPrefix, ignoreMissing, details fields
- KeyGeneratorType: Added support for 'uuid' and 'padded'
- License: Added diskUsage and upgrading fields for Community Edition dataset limits
- DatabaseSharding: Added support for "flexible" value
- ServerHealth: Added LastAckedTime, Timestamp, SyncTime for Coordinators/DB-Servers
- Vector index: Added 3.12.9+ trainingState/errorMessage response support and stopped sending deprecated inBackground in EnsureVectorIndex requests
- Tests: Removed `Test_EnsureVectorIndex` subtest "Vector index reports errorMessage when unusable in 3.12.9+" that asserted sparse vector index creation with insufficient training data returns 201 with `trainingState` unusable and `errorMessage`. ArangoDB Nightly `go_driver_0` often fails with ~300s server error `timed out waiting for vector index to become ready` while standalone `arangod --vector-index true` behaves as expected; see comment in `v2/tests/database_collection_indexes_test.go` to restore the subtest when server behavior in Nightly CI matches standalone.

### Deprecations and Removals (v2.x, in preparation for ArangoDB v4.0)
- CollectionStatus and collection status/statusString: Only values 3 (loaded) and 5 (deleted) are in use; other status values and status/statusString properties deprecated (may be removed in v4.0)
- minReplicationFactor: Deprecated, use writeConcern instead
- QueryOverwrite (AQL queries): Deprecated, will be removed in v4.0
- CollectionDocumentCreateOptions.Overwrite: Deprecated, will be removed in v4.0
- KeyGeneratorType: Deprecated types not in docs
- EngineType: 'mmfiles' is deprecated/removed
- Consolidation policy 'bytes_accum': deprecated for inverted indexes (per devel inverted index API docs); not deprecated for ArangoSearch views
- CacheRespObject: Reviewed for missing fields per API docs
- Foxx, User-defined AQL functions (UDFs), JavaScript Transactions, Foxx Queues, /_api/tasks, Task/TaskOptions: Deprecated, will be removed in v4.0. Deprecation comments added on: ClientFoxx, ClientFoxxService, Manifest; CreateUserDefinedFunction, DeleteUserDefinedFunction, GetUserDefinedFunctions, UserDefinedFunctionObject; TransactionJS, TransactionJSOptions; ClientTasks, Task, TaskOptions; ExecuteAdminScript; ServerStatusResponse.FoxxApi, CoordinatorInfo.Foxxmaster/IsFoxxmaster; replication ApplierConfig.IncludeFoxxQueues
- CollectionDocumentImportOptions: Deprecated/removed legacy fields as per API docs
- CreateGraphOptions: Only 'satellites' is valid; others moved to GraphDefinition
- Fulltext index type: Deprecated, will be removed in v4.0. minLength option is only used for fulltext indexes. FulltextIndexType constant added (deprecated); MinLength and fulltext-related comments in collection_indexes.go and client_admin_cluster.go
- DatabaseSharding: Both "" and "flexible" supported for creation; server response checked
- EdgeDetails: $label clarified as user-defined
- ExplainQueryOptions: Deprecated/removed legacy options, added missing ones
- KeyOpts.LastValue: Used internally only
- LicenseStatus: 'expired' not used; added diskUsage, upgrading if missing
- Enterprise Edition features: Now available in Community Edition (v3.12.5+); docs/comments updated.  Comments updated across graph (SmartGraph, Satellite, IsSmart, IsDisjoint, Satellites), query options (AllowDirtyReads, SatelliteSyncWait, SkipInaccessibleCollections, EnterpriseOnly), collection opts (IsSmart, SmartJoinAttribute, SmartGraphAttribute), ArangoSearch/views (OptimizeTopK, PrimarySortCache, PrimaryKeyCache, Nested, Cache), inverted index (OptimizeTopK, Nested), backup (Force), and shared (PrimarySort Cache)
- ServerStatusResponse: 'mode' deprecated, use 'operationMode'
- ServerHealth: LastHeartbeatAcked, LastHeartbeatSent, LastHeartbeatStatus deprecated (not returned by current server, older format)
- ServerInformation: 'writeOpsEnabled' is deprecated
- ServerRole: SingleActive and SinglePassive deprecated (Active Failover removed in v3.12.0)
- SetCollectionPropertiesOptionsV2: 'journalSize' is deprecated (no longer existent since v3.7; MMFiles removed)
- Collection option fields: IsVolatile, DoCompact, IndexBuckets, JournalSize are deprecated

## [2.2.0](https://github.com/arangodb/go-driver/tree/v2.2.0) (2026-02-17)
- Add endpoint to fetch deployment id
- Add ARM Support for V2 testcases 
- Set TESTV2PARALLEL from 1 to 4
- Disabled V8 related testcases in V1 and V2
- Added new ConsolidationPolicy attributes to support updated configuration options for ArangoSearch Views properties and Inverted Indexes
- Add Vector index feature
- Add Len() method to response readers for bulk CRUD operations; add ReadAll() helpers; improve thread-safety with mutexes; fix OldObject/NewObject pointer reuse in readers
- Add shutdown endpoints to v2
- Switch to Go 1.24.11
- Switched to Go 1.24.13 to fix a security vulnerability
- Modified Test_UserCreation test case to use parallel execution and replaced hardcoded usernames with dynamically generated values.

## [2.1.6](https://github.com/arangodb/go-driver/tree/v2.1.6) (2025-11-06)
- Add missing endpoints from replication
- Add missing endpoints from monitoring
- Add missing endpoints from administration
- Add missing endpoints from cluster
- Add missing endpoints from security
- Add missing endpoints from authentication
- Add missing endpoints from general-request-handling
- Add benchmark tests for v1 and v2 to compare performance
- Switch to Go 1.24.9

## [2.1.5](https://github.com/arangodb/go-driver/tree/v2.1.5) (2025-08-31)
- Add tasks endpoints to v2
- Add missing endpoints from collections to v2
- Add missing endpoints from query to v2
- Add SSO auth token implementation
- Add missing endpoints from foxx to v2
- Switch to Go 1.23.12

## [2.1.3](https://github.com/arangodb/go-driver/tree/v2.1.3) (2025-02-21)
- Switch to Go 1.22.11
- Switch to jwt-go v5
- Fix incorrect Http method for ReplaceDocuments
- Fix unmarshalling error due to field name collision in Documents.
- Add bulk operations on Collections to VertexCollection and Edges (General and Satellite Graphs only)
- Add OldRev to CollectionDocumentUpdateResponse and CollectionDocumentReplaceResponse


## [2.1.2](https://github.com/arangodb/go-driver/tree/v2.1.2) (2024-11-15)
- Expose `NewType` method
- Connection configuration helper
- Adjust Cursor options
- Switch to Go 1.22.8
- Remove deprecated context functions
- Fix Error Handler in CreateCollectionWithOptions

## [2.1.1](https://github.com/arangodb/go-driver/tree/v2.1.1) (2024-09-27)
- Improve backup tests stability
- CheckAvailability function for the specific member
- Switch to Go 1.22.6
- Support for missing dirty read options (query, transaction apis)
- Get inbound and outbound edges
- Deprecate VPACK support

## [2.1.0](https://github.com/arangodb/go-driver/tree/v2.1.0) (2024-04-02)
- Switch to Go 1.21.5
- Disable AF mode in tests (not supported since 3.12)
- Allow skipping validation for Database and Collection existence
- Add support for Graph API
- Add support for Graph API - Vertex
- Add support for Graph API - Edge
- Align ArangoSearchView and ArangoSearchAliasView with API
- `MDI` and `MDI-Prefixed` indexes. Deprecate `ZKD` index
- Fix url encoding for names with slashes
- Users API support
- Add ArangoDBConfiguration to Client config. Deprecate Context config options
- External versioning
- Switch to Go 1.21.8
- multi_delimiter analyzer support
- Wildcard analyzer support
- Backup API support
- Admin Cluster API support
- Set Licence API support
- Transparent compression of requests and responses (ArangoDBConfiguration.Compression)
- Fix Cursor batch


## [2.0.3](https://github.com/arangodb/go-driver/tree/v2.0.3) (2023-10-31)
- Add optional status code checks. Consistent return of response
- JavaScript Transactions API
- Async Client
- Fix connection.NewRequestWithEndpoint()
- Add support for MaglevHashEndpoints
- Add basic support for Views and Analyzers
- Add ServerMode/SetServerMode/ServerID
- Add collection Truncate, Count, Properties, SetProperties
- Add and re-organize missing collection properties fields
- Rename CreateCollectionOptions to CreateCollectionProperties
- Add support for missing query options (create documents, remove collection, remove view)
- Adjust CursorStats and JournalSize types
- Improve returning old doc handling in CollectionDocumentDelete
- Agency: Supply ClientID with agency transactions
- Automate release process
