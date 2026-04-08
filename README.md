# ArangoDB Go Driver

This project contains the official Go driver for the [ArangoDB database system](https://arangodb.com).

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/arangodb/go-driver/tree/master.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/arangodb/go-driver/tree/master)
[![GoDoc](https://godoc.org/github.com/arangodb/go-driver?status.svg)](http://godoc.org/github.com/arangodb/go-driver)

Version 2:
- [Tutorial](https://docs.arangodb.com/stable/develop/drivers/go/)
- [Code examples](v2/examples/)
- [Reference documentation](https://godoc.org/github.com/arangodb/go-driver/v2)

Version 1:
- ⚠️ This version is deprecated and will not receive any new features.
  Please use version 2 ([v2/](v2/)) instead.
- [Tutorial](Tutorial_v1.md)
- [Code examples](examples/)
- [Reference documentation](https://godoc.org/github.com/arangodb/go-driver)

## Supported Go Versions

### Version 2

| Driver                  | Go 1.21 | Go 1.22 | Go 1.23 | Go 1.24 | Go 1.25 |
|-------------------------|---------|---------|---------|---------|---------|
| `2.1.0`                 | ✓       | -       | -       | -       | -       |
| `2.1.1`-`2.1.2`-`2.1.3` | -       | ✓       | -       | -       | -       |
| `2.1.5`                 | -       | -       | ✓       | -       | -       |
| `2.1.6`-`2.2.0`         | -       | -       | -       | ✓       | -       |
| `master`                | -       | -       | -       | -       | ✓       |

### Version 1 (deprecated)

| Driver                   | Go 1.19 | Go 1.20 | Go 1.21 | Go 1.22 | Go 1.23 | Go 1.24 | Go 1.25 |
|--------------------------|---------|---------|---------|---------|---------|---------|---------|
| `1.5.0`-`1.5.2`          | ✓       | -       | -       | -       | -       | -       | -       |
| `1.6.0`-`1.6.1`          | -       | ✓       | -       | -       | -       | -       | -       |
| `1.6.2`                  | -       | -       | ✓       | -       | -       | -       | -       |
| `1.6.4`-`1.6.5`-`1.6.6`  | -       | -       | -       | ✓       | -       | -       | -       |
| `1.6.7`                  | -       | -       | -       | -       | ✓       | -       | -       |
| `1.6.9`                  | -       | -       | -       | -       | -       | ✓       | -       |
| `master`                 | -       | -       | -       | -       | -       | -       | ✓       |

## Supported ArangoDB Versions

| Driver                  | ArangoDB 3.10 | ArangoDB 3.11 | ArangoDB 3.12 |
|-------------------------|---------------|---------------|---------------|
| `1.5.0`                 | ✓             | -             | -             |
| `1.6.0`                 | ✓             | ✓             | -             |
| `2.1.0`-`2.2.0`         | ✓             | ✓             | ✓             |
| `master`                | +             | +             | +             |

Key:

* **Go:** `✓` marks the Go minor version used to build and test that driver tag (and typically the minimum you need per `go.mod`); `-` otherwise. Patch toolchains are listed in [v2/CHANGELOG.md](v2/CHANGELOG.md) (v2) and [CHANGELOG.md](CHANGELOG.md) (v1).
* **ArangoDB:**
  * `✓` Exactly the same features in both the driver and the ArangoDB version.
  * `+` Features included in the driver may be not present in the ArangoDB API.
    Calls to ArangoDB may result in unexpected responses (404).
  * `-` The ArangoDB version has features that are not supported by the driver.
