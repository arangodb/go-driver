# Build and test guide

Contents: [environment](#environment), [focused checks](#focused-checks),
[database fixtures](#database-fixtures), [test placement](#test-placement-and-coverage),
[CI parameters](#ci-parameters).

## Environment

Run Make targets from the repository root unless noted. Integration tests use
Docker: a Go image (`GOVERSION`, currently 1.25.13) and an ArangoDB image.
v2 CI jobs and local `make run-v2-tests-*` use ArangoDB 3.12. v3 defaults to
`arangodb/core-preview:4.0-nightly` unless `ARANGODB` was set in the
environment or on the Make command line. See `.circleci/config.yml` and
`MAINTAINERS.md`.

`v2/` and `v3/` are separate modules. Package tests can run on the host from
the module directory. Integration tests should go through Make so
`TEST_ENDPOINTS`, auth, and `test/cluster.sh` stay consistent with CI.

## Focused checks

Compile and run database-free package tests:

```sh
# from v2/ (ArangoDB 3.12) or v3/ (ArangoDB 4.0)
go test ./connection ./arangodb/...
```

```sh
# from repository root, in Docker (v2 only; v3 has no unit-test Make target)
make run-v2-unit-tests
```

With a disposable database, run a focused integration test. Make starts
ArangoDB on `127.0.0.1:7001` unless `TEST_ENDPOINTS_OVERRIDE` is set. Use
`127.0.0.1`, not `localhost` (IPv6 vs IPv4). `TESTOPTIONS` is appended to
`go test` inside the selected module (`v2/` or `v3/`):

```sh
# 3.12 line
TESTOPTIONS="-test.run TestName -test.v" make run-v2-tests-single-with-auth
# 4.0 line
TESTOPTIONS="-test.run TestName -test.v" make run-v3-tests-single-with-auth
```

Replace `TestName` with the existing or new test. Confirm the selected test
actually ran (including `Wrap` subtests `HTTP JSON` / `HTTP2 JSON` and
skips). A misspelled `-test.run` can produce `ok` with no cases.

Common local targets. The same suffixes exist as `run-v3-tests-*` for the
4.0 line:

| Target | Topology / auth |
| --- | --- |
| `run-v2-tests-single-without-auth` | single, no auth |
| `run-v2-tests-single-with-auth` | single, basic auth, TLS |
| `run-v2-tests-cluster-with-basic-auth` | cluster, basic auth, TLS |
| `run-v2-tests-cluster-with-jwt-auth` | cluster, JWT, TLS |
| `run-v2-tests-cluster-without-auth` | cluster, no auth, TLS |
| `run-v2-tests-cluster-without-ssl` | cluster, no auth, no TLS |

`RACE=on` enables the race detector (`CGO_ENABLED=1`). `VERBOSE=1` adds `-v`.
`TESTV2PARALLEL` defaults to 4; cluster tests under load may need
`TESTV2PARALLEL=1`.

Avoid `make run-v2-tests` and `make run-v3-tests` for everyday work: each
runs the full single and cluster matrix for that line. Do not run v1
`run-tests-*`. Do not run `run-v3-tests-*` to validate a 3.12 change, or
`run-v2-tests-*` to validate a 4.0 change.

## Database fixtures

`test/cluster.sh` (invoked by `__test_prepare`) creates Docker resources and
exposes the server at `TEST_ENDPOINTS` (`http://127.0.0.1:7001` or
`https://127.0.0.1:7001` when `TEST_SSL=auto`). Auth comes from `TEST_AUTH`
(`none`, `rootpw`, `jwt`) and is passed into the test container as
`TEST_AUTHENTICATION` (`basic:root:…`, `jwt:root:…`).

`v2/tests` reads `TEST_ENDPOINTS` and `TEST_AUTHENTICATION`.
`TEST_ENDPOINTS_OVERRIDE` / `TEST_AUTHENTICATION_OVERRIDE` remap those values
in the Makefile (used by Kubernetes and remote runs). `TEST_MODE` is
`single`, `cluster`, or `resilientsingle`. Active failover
(`resilientsingle`) is skipped from ArangoDB 3.12 in `Wrap`.

Helpers in `v2/tests`:

- `Wrap` / `WrapConnection` — HTTP JSON and HTTP/2 JSON clients
- `WithDatabase`, `WithCollectionV2` — unique names, cleanup
- `requireClusterMode`, `requireSingleMode`, `skipNoEnterprise`
- `skipBelowVersion`, `skipFromVersion`, `skipVersionNotInRange`
- `requireV8Enabled` — tasks, Foxx, JS transactions
- `skipIfVectorIndexDisabled` — `ENABLE_VECTOR_INDEX`

Foxx tests need demo zip files via `TEST_RESOURCES`. Vector-index tests need
`ENABLE_VECTOR_INDEX=true` (CI sets this). Extra compression features need
`ENABLE_DATABASE_EXTRA_FEATURES=true`.

Fixtures are destructive. Do not point them at a shared or production server.
Do not run two Make integration targets against the same instance at once.

## Test placement and coverage

| Change | Start with |
| --- | --- |
| Pure Go logic (URL, version compare, JWT cache, decoders) | `connection/*_test.go` or `arangodb/*_test.go` in the module being changed |
| Public API, requests, options, response mapping | that module's `tests` package, via `Wrap` |
| Retry, timeout, failover, pooling | existing resiliency / Toxiproxy tests; do not invent a parallel harness |
| Kubernetes ingress, coordinator loss | `make run-k8s-v2-*` after `deploy/kubernetes` kind setup |
| Network faults | `make run-v2-tests-toxiproxy` or `run-k8s-v2-toxiproxy` |
| Performance | `v2/BENCHMARKS.md`; compare equivalent environments |

Follow `testing` and `testify/require`, and the nearby fixture style. Use
version/license/topology skips for genuine capability differences, not to hide
regressions. A green single-server run does not cover cluster-only APIs
(graphs with smart options, cluster admin, some replication).

Kubernetes and Toxiproxy suites have extra infrastructure
(`deploy/kubernetes/README.md`, `v2/tests/k8s-tests.md`). Run them only when
the change touches those paths.

## CI parameters

Choose relevant CI dimensions, not the whole matrix. Read
`.circleci/config.yml` for the jobs of the line you changed: single/cluster, auth/JWT/none,
TLS, HTTP/2, Kubernetes, resiliency, Toxiproxy, and a 3.12.9 pin used for
some cases.

`make run-v2-unit-tests`, `make license-verify`, and `make fmt-verify` need no
database. Lint from the repository root after `make tools`: `make linter` and
`.tmp/bin/golangci-lint run ./...` cover the root module only. Lint a nested
module with `(cd v2 && ../.tmp/bin/golangci-lint run ./...)` or the same
command for `v3`. See the change guide for the full sequence.
For documentation-only changes, check referenced paths and commands against
the Makefile and `git diff --check`.

For code changes with missing infrastructure, still run available checks and
state which behavior and configurations remain unverified. A zero-test,
skipped, or stale report is not a passing regression test.
