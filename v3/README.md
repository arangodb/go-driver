# ArangoDB Go Driver Version 3

This is the official Go driver for ArangoDB 4.0+.

Version 3 contains the API and behavior updates needed for ArangoDB 4.0,
including removal of obsolete endpoints and legacy fields.

## Installation

```bash
go get github.com/arangodb/go-driver/v3
```

Import path:

```go
import "github.com/arangodb/go-driver/v3/arangodb"
```

## Driver Version Support Tiers

The repository currently has three major driver lines:

- `v1`: deprecated; no new features, migration to newer versions is strongly recommended.
- `v2`: deprecated for new feature development; maintenance-only for existing users.
- `v3`: active version for ArangoDB 4.0+ and all new development.

## Migration from v2 to v3

See [MIGRATION.md](./MIGRATION.md) for removed identifiers and their v3
replacements.

## Deprecation Notes

- From ArangoDB 4.0 onward, the MMFiles storage format is not supported.
- The v2 driver does not receive new features.