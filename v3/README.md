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

## Deprecation Notice

From ArangoDB 4.0 onward, the MMFiles storage format is not supported.

The v2 driver is deprecated and will not receive new features.
Please use v3 for ArangoDB 4.0+ deployments.