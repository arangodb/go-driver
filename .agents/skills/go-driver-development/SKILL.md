---
name: go-driver-development
description: >-
  Maintain and extend the ArangoDB Go driver v2 and v3 modules. Use v2
  (github.com/arangodb/go-driver/v2) for ArangoDB 3.12.* changes and v3
  (github.com/arangodb/go-driver/v3) for ArangoDB 4.0.* changes. Use for
  implementing driver features or bug fixes, refactoring Go APIs and
  internals, changing HTTP connections, authentication, retries or
  serialization, updating configuration or dependencies, and selecting tests
  for Make, Kubernetes, or Toxiproxy suites. Not a guide to using the driver
  in an application. Out of scope for the deprecated v1 root module unless a
  shared Makefile or CI change is required.
---

# Go driver v2 and v3 development

Choose the module from the server line: **`v2/` for ArangoDB 3.12.***, **`v3/`
for ArangoDB 4.0.***. Do not edit both unless the task covers both lines.

Apply the repository's [AGENTS.md](../../../AGENTS.md). Use the following workflow
at the scale of the change; do not load every reference for every task.

## Locate the change

Trace one neighboring operation from the public interface in that module's
`arangodb` package to
URL construction, `connection.Call*`, and response / error mapping before
choosing the implementation layer. For a bug, identify the violated contract
and a reproducer. For a feature, verify the server HTTP API and supported
ArangoDB versions rather than extrapolating from another endpoint. For a
refactor, identify the observable behavior that must remain fixed.

| Work | Read when needed |
| --- | --- |
| Locate ownership or trace a request | [Architecture](references/architecture.md) |
| Change API, options, or wire mapping | [Change guide: API](references/changes.md#api-options-and-errors) |
| Change HTTP, auth, retries, or cursors | [Change guide: connection](references/changes.md#connection-auth-and-execution) |
| Change dependencies or Go version | [Change guide: packaging](references/changes.md#dependencies-and-tooling) |
| Add tests or choose validation commands | [Testing](references/testing.md) |

## Implement and validate

Keep public contracts in `*_impl.go`-free files (`client.go`, `collection.go`,
…) and implementations in the matching `*_impl.go`. Check related overloads
(`Foo` vs `FooWithOptions`) and both HTTP/1.1 and HTTP/2 where the `Connection`
layer is involved.

Add regression coverage at the affected boundary. Database-free tests belong
next to the package (`connection`, `arangodb` under `v2/` or `v3/`).
Integration tests belong in that module's `tests` package and should use
`Wrap` plus the existing fixtures. Where
practical, demonstrate that a bug's reproducer fails before the fix. Select
the smallest meaningful test run first, then expand for changed contracts:
HTTP vs HTTP/2, single vs cluster, auth or TLS, Kubernetes, or Toxiproxy. Use
the testing guide; a successful `go test` of unselected packages is not
validation of server behavior.

Review the diff for contract drift, then report as required by AGENTS.md.
