# Change Log

## [master](https://github.com/arangodb/go-driver/tree/master) (N/A)
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
