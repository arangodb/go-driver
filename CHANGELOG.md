# Change Log

## [master](https://github.com/arangodb/go-driver/tree/master) (N/A)

## [1.3.2](https://github.com/arangodb/go-driver/tree/v1.3.2) (2022-05-16)

- Fix selectivityEstimate Index field type

## [1.3.1](https://github.com/arangodb/go-driver/tree/v1.3.1) (2022-03-23)

- Add support for `exclusive` field for transaction options
- Fix cursor executionTime statistics getter
- Fix cursor warnings field type
- Fix for DocumentMeta name field overrides name field

## [1.3.0](https://github.com/arangodb/go-driver/tree/v1.3.0) (2022-03-17)

- Disallow unknown fields feature
- inBackground parameter in ArangoSearch links
- ZKD indexes
- Hybrid SmartGraphs
- Segmentation and Collation Analyzers
- Bypass caching for specific collections
- Overload Control
- [V2] Add support for streaming the response body by the caller.
- [V2] Bugfix with escaping the URL path twice.
- Bugfix for the satellites' collection shard info.
- [V2] Support for satellites' collections.

## [1.2.1](https://github.com/arangodb/go-driver/tree/v1.2.1) (2021-09-21)
- Add support for fetching shards' info by the given collection name.
- Change versioning to be go mod compatible
- Add support for ForceOneShardAttributeValue in Query

## [1.2.0](https://github.com/arangodb/go-driver/tree/1.2.0) (2021-08-04)
- Add support for AQL, Pipeline, Stopwords, GeoJSON and GeoPoint Arango Search analyzers.
- Add `estimates` field to indexes properties.
- Add tests for 3.8 ArangoDB and remove tests for 3.5.
- Add Plan support in Query execution.
- Change Golang version from 1.13.4 to 1.16.6.
- Add graceful shutdown for the coordinators.
- Replace 'github.com/dgrijalva/jwt-go' with 'github.com/golang-jwt/jwt'

## [1.1.1](https://github.com/arangodb/go-driver/tree/1.1.1) (2020-11-13)
- Add Driver V2 in Alpha version
- Add HTTP2 support for V1 and V2
- Don't omit the `stopwords` field. The field is mandatory in 3.6 ArangoDB

## [1.1.0](https://github.com/arangodb/go-driver/tree/1.1.0) (2020-08-11)
- Use internal coordinator communication for cursors if specified coordinator was not found on endpoint list
- Add support for Overwrite Mode (ArangoDB 3.7)
- Add support for Schema Collection options (ArangoDB 3.7)
- Add support for Disjoint and Satellite Graphs options (ArangoDB 3.7)

## [1.0.0](https://github.com/arangodb/go-driver/tree/1.0.0) (N/A)
- Enable proper CHANGELOG and versioning
