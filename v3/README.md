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

- `v1`: deprecated legacy line; no new features. Migration to v2 or v3 is strongly recommended.
- `v2`: for **ArangoDB 3.x**; deprecated for new feature development; maintenance-only for existing users.
- `v3`: active version for **ArangoDB 4.0+** and all new development.

Compared with v2 over HTTP/2, v1 (HTTP/1.1) can be **dramatically slower** on some write-heavy batches (for example 100k creates, updates, or deletes), and published benches show **much higher** bytes-per-op and allocations there; reads tend to be closer because they are often I/O-bound. Full tables and setup: [v2/BENCHMARKS.md](../v2/BENCHMARKS.md).

## Migration from v2 to v3

See [MIGRATION.md](./MIGRATION.md) for removed identifiers and their v3
replacements.

## Deprecation Notes

- From ArangoDB 4.0 onward, the MMFiles storage format is not supported.
- The v2 driver targets ArangoDB 3.x and does not receive new features; use v3 for 4.0+.