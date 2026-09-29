# ArangoDB Go driver: agent instructions

HTTP/1.1 and HTTP/2 clients with runtime JSON serialization, in two current
modules:

- **`v2/`** (`github.com/arangodb/go-driver/v2`) for **ArangoDB 3.12.***
- **`v3/`** (`github.com/arangodb/go-driver/v3`) for **ArangoDB 4.0.***

Work in the module that matches the server line. A 3.12 change belongs in
`v2/`; a 4.0 change belongs in `v3/`. Do not copy a change into the other
module unless the task asks for both. Do not change the deprecated root (v1)
module unless the task is explicitly about v1, or a shared `Makefile` /
`.circleci/config.yml` change is required to keep v2 or v3 tests working.

For implementation, bug fixes, refactoring, dependencies, or tests, use the
[go-driver-development skill](.agents/skills/go-driver-development/SKILL.md).
Open its references only as needed. Agents without skill discovery can read the
same files directly.

## Environment

From `v2/` or `v3/`, compile and run package tests that do not need a database:

```sh
go test ./connection ./arangodb/...
```

From the repository root, the v2 check through Docker (`GOVERSION` from the
Makefile, currently 1.25.13). There is no matching `run-v3-unit-tests` target;
run the command above inside `v3/` for that module:

```sh
make run-v2-unit-tests
```

With a disposable database ready, run a focused integration area (Make starts
the server unless `TEST_ENDPOINTS_OVERRIDE` is set). v2 defaults to the 3.12
image; `run-v3-tests-*` defaults to the 4.0 image:

```sh
TESTOPTIONS="-test.run TestName -test.v" make run-v2-tests-single-with-auth
TESTOPTIONS="-test.run TestName -test.v" make run-v3-tests-single-with-auth
```

## Compatibility boundaries

- Preserve public source compatibility and HTTP wire behavior unless the task
  explicitly changes that contract. Adding a method to a public interface is a
  breaking change for external implementers; document it and prefer an additive
  `*WithOptions` method when an existing method cannot grow arguments.
- Each module's `go.mod` pins the supported Go version (`go 1.25.0`, toolchain
  `go1.25.13` at the time of writing). Do not raise it, or `GOVERSION` in the
  Makefile / CircleCI images, unless the task is a Go upgrade. See
  `MAINTAINERS.md`.
- v2 is tested against ArangoDB 3.12; v3 against ArangoDB 4.0. Server features
  that do not exist on the CI image must be gated with the existing
  `skipBelowVersion` / `skipFromVersion` helpers. A green run on one server
  line does not prove the other.
- Neither v2 nor v3 supports VelocyStream. VelocyPack remains in the HTTP
  decoder but is not developed; do not extend it. Do not add VST code.
- Keep HTTP/1.1 and HTTP/2 behavior aligned through the shared `Connection`
  interface. Integration `Wrap` already runs both; a JSON-over-HTTP-only test
  does not establish HTTP/2.

## Validation boundaries

Use the root `Makefile` for integration tests; they run `go test ./tests`
inside `v2/` or `v3/` in a Go container after `test/cluster.sh` starts
ArangoDB. Package tests under `connection` and `arangodb` are the
database-free layer. Selecting only those packages is not validation of
request/response behavior. Kubernetes and Toxiproxy Make targets exist for
v2; do not assume the same targets exist for v3.

Database tests create and drop randomly named databases and collections, and
some suites change users or server state. Use a disposable server. Do not run
suites concurrently against a shared instance. Do not use `release-*` or
`prerelease-*` Make targets for local validation.

The
[testing guide](.agents/skills/go-driver-development/references/testing.md)
covers fixtures, `TEST_*` environment variables, Kubernetes and Toxiproxy
suites, and CI dimensions.

## Change scope

Match nearby code and tests in the module you are changing; avoid unrelated
formatting, dependency upgrades, or version/release changes. New Go files
carry the Apache license header used by neighboring files (`make tools` then
`make license`). Edit source, not generated or vendor trees. Update affected
godoc and that module's `examples` when behavior changes. Add a bullet under
the `master` section of `v2/CHANGELOG.md` or `v3/CHANGELOG.md` for
user-visible changes; do not rewrite released version sections.

Treat the module source, the root `Makefile`, and `.circleci/config.yml` as
evidence of current behavior. If this guidance has drifted, follow those files
and flag the drift. Report which line changed (v2 / 3.12 or v3 / 4.0),
compatibility implications, checks actually run with results, and untested
relevant variants (HTTP vs HTTP/2, single vs cluster, auth/TLS, and for v2
Kubernetes), separating environment limits from real failures.
