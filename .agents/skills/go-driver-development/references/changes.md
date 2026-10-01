# Change guide

Read only the sections touching the change. Paths are repository-relative.
Apply them under **`v2/` for ArangoDB 3.12.*** or **`v3/` for ArangoDB 4.0.***.
Examples below use `v2/` paths; open the same relative path under `v3/` when
that is the line being changed.

## API, options and errors

For an endpoint change, trace the public interface, its `*Impl`, option/result
structs, and a neighboring operation with the same request/response shape — not
just a similar name. Update applicable overloads and godoc together.

Preserve the distinction between absent options and explicit `false`, `0`, or
empty strings. Pointer fields with `omitempty` let the server choose defaults.
Keep parameters in their specified location (path, query, headers, body).
Reuse `connection.NewUrl`, `connection.WithQuery`, `connection.WithBody`, and
the header helpers in `v2/arangodb/shared.go` / `v2/connection/modifiers.go`.
Test non-ASCII or reserved characters when changing names, handles, or path
construction; escape path segments.

Adding a method to a public interface (`Client`, `Collection`,
`CollectionIndexes`, …) is source-breaking for external implementers. Prefer
an additive `FooWithOptions` when `Foo` already exists. If the interface must
grow, call it out in that module's `CHANGELOG.md`.

Check success and error contracts, not just successful decoding: missing
documents, precondition headers (`If-Match` / `If-None-Match`), silent writes,
bulk per-document errors, dirty-read and transaction headers. Convert
non-success statuses through `shared.ResponseStruct.AsArangoErrorWithCode`
(or the existing helper on that type). Preserve HTTP status, Arango error
numbers, and `shared.IsArangoError` / `IsNotFound` behavior.

When adding a client-level API, register it in `newClient` (`client_impl.go`)
and on the `Client` interface in `client.go`. Follow
`client_access_tokens.go` / `client_access_tokens_impl.go` for a small
complete example.

## Connection, auth and execution

Keep HTTP/1.1 and HTTP/2 on the `Connection` interface. Changes to
`Do` / `Stream`, content type, compression, or `HostHeader` must be valid for
both `NewHttpConnection` and `NewHttp2Connection`. Integration `Wrap` already
parameterizes those two; use it rather than inventing a second client helper.

Before changing retry wrappers, distinguish failures before a request is sent
from ambiguous outcomes after sending it. Do not broaden retries in a
refactor: a replay can duplicate writes. `RetryOn503` is the existing
narrow helper.

For JWT changes, follow `NewJWTAuthWrapper` and `auth_jwt_impl.go`: a 401 must
re-authenticate even if a cached token has not reached `exp` (secret reload).
Cover wrapper tests in `v2/connection` without a database when the logic is
local; add `v2/tests` coverage when the server login path is involved.

For cursor changes, preserve batch state and server-side cleanup through
exhaustion, explicit close, and failure. For pool or wrapper changes, check
shutdown and that the inner connection still owns the transport.

Do not add VelocyStream. Do not expand VelocyPack beyond what `decoder.go`
already does.

## Dependencies and tooling

`v2/go.mod` and `v3/go.mod` are separate modules. Run `go mod tidy` from the
module you changed. Do not tidy or upgrade the other current module, or the
root module, unless the task includes them.

Go version bumps must stay aligned across the affected `go.mod` files, root
`Makefile` `GOVERSION`, and CircleCI `goImage` / privileged golang-docker
images. Follow `MAINTAINERS.md`; do not bump Go to silence a single test.

New `.go` files need the Apache header used by neighbors. After `make tools`:

```sh
make license
make fmt
```

Do not install `golangci-lint` onto `PATH` (including via snap). From the
repository root, install the repo binary first, then lint. `make tools` writes
it to `.tmp/bin/golangci-lint`.

```sh
make tools
make linter
.tmp/bin/golangci-lint run ./...
```

`make linter` and `.tmp/bin/golangci-lint run ./...` both stay in the root Go
module. `./...` does not enter nested modules that have their own `go.mod`, so
"0 issues" from those commands does not mean `v2/` or `v3/` was linted.
CircleCI `check-code` runs only `make tools` and `make linter`, so it has the
same limit.

Lint `v2/` and `v3/` from the repository root with a separate command for each
module. Each command lints only that module:

```sh
(cd v2 && ../.tmp/bin/golangci-lint run ./...)
(cd v3 && ../.tmp/bin/golangci-lint run ./...)
```

Run the command for the module you changed. The config is the repository-root
`.golangci.yaml`. Do not disable linters or add excludes to make an unrelated
change pass.

Driver version sent to the server comes from `v2/version/VERSION` or
`v3/version/VERSION` via `version.DriverVersion()`. Do not edit `VERSION` as
part of a feature change; release targets (`make release-v2-*` or
`make release-v3-*`) own that file. User-visible notes go under `master` in
`v2/CHANGELOG.md` or `v3/CHANGELOG.md`.
