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

- `v1` (module `github.com/arangodb/go-driver`, [source](https://github.com/arangodb/go-driver)): **ArangoDB: none** — unsupported / EOL (no development, bug fixes, or security patches). Migration to v2 or v3 is strongly recommended. Compared with v2 over HTTP/2, v1 (HTTP/1.1) can be **dramatically slower** on some write-heavy batches (for example 100k creates, updates, or deletes), and published benches show **much higher** bytes-per-op and allocations there; reads tend to be closer because they are often I/O-bound. Tables and methodology: [v2/BENCHMARKS.md](../v2/BENCHMARKS.md).
- `v2` (module `github.com/arangodb/go-driver/v2`, [source](https://github.com/arangodb/go-driver/tree/master/v2)): **ArangoDB 3.x.x only** — maintenance mode (bug fixes and additive API changes only; breaking or 4.0-related work belongs in v3).
- `v3` (module `github.com/arangodb/go-driver/v3`, [source](https://github.com/arangodb/go-driver/tree/master/v3)): **ArangoDB 4.x.x and newer** — active development; ArangoDB 4.0 APIs; not compatible with ArangoDB 3.x.

| Driver | Module | ArangoDB | Support |
|--------|--------|----------|---------|
| v1 | `github.com/arangodb/go-driver` | none | Unsupported / EOL |
| v2 | `github.com/arangodb/go-driver/v2` | 3.x.x | Maintenance (bug fixes, additive changes) |
| v3 | `github.com/arangodb/go-driver/v3` | 4.x.x+ | Active development |

## Migration from v2 to v3

See [MIGRATION.md](./MIGRATION.md) for removed identifiers and their v3
replacements.

## Deprecation Notes

- From ArangoDB 4.0 onward, the MMFiles storage format is not supported (v3 only).