# Architecture and ownership

Paths are repository-relative. This is a navigation map, not a replacement for
reading the implementation being changed. **`v2/` is the ArangoDB 3.12.* line.
`v3/` is the ArangoDB 4.0.* line.** Package layout below is the same under both
module roots; examples use `v2/` paths.

## Packages

| Package | Responsibility |
| --- | --- |
| `v2/arangodb/` (and `v3/arangodb/`) | Public client API: interfaces, option/result types, and request mapping. |
| `…/arangodb/shared/` | Shared error types, Arango error numbers, and response envelopes. |
| `…/connection/` | HTTP/1.1 and HTTP/2 transports, request/response codecs, auth, endpoint selection, wrappers (retry, async, JWT, pool, compression). |
| `…/utils/` | Small helpers used by the driver (endpoint parsing, JWT helpers under `utils/jwt`). |
| `…/log/` | Optional driver logging adapters. |
| `…/version/` | Driver version string (`VERSION`) sent as `x-arango-driver`. |
| `…/tests/` | Integration tests against a live server. Kubernetes and Toxiproxy suites live under `v2/tests`. |
| `…/examples/` | Runnable consumer examples; keep in sync with public API changes. |

Import paths are `github.com/arangodb/go-driver/v2/...` and
`github.com/arangodb/go-driver/v3/...`. The repository root is the deprecated
v1 module. Shared orchestration lives in the root `Makefile` and
`.circleci/config.yml`.

`arangodb` owns the API; `connection` owns I/O. Do not make `arangodb` depend
on a concrete HTTP client type. Call through `connection.Connection` and
`connection.Call*`.

## Request lifecycle

```text
connection.NewHttpConnection / NewHttp2Connection
  (+ optional wrappers: JWT, retry, async, pool)
  -> arangodb.NewClient(conn)
  -> client embeds feature structs (database, users, admin, …)
  -> interface method on *fooImpl
  -> connection.NewUrl(...) + CallGet/Post/Put/Patch/Delete/Head
  -> RequestModifier(s) (query, headers, body, config)
  -> Connection.Do / Stream
  -> decode JSON (default) into response struct
  -> HTTP status + shared.ResponseStruct.AsArangoErrorWithCode
```

`NewClient` in `v2/arangodb/client_impl.go` wires sub-APIs onto `*client` and
attaches `Requests` for raw HTTP helpers. Child handles (database, collection,
graph, view, cursor) keep a reference to the same `connection.Connection`; they
are not independent clients.

A representative endpoint is `v2/arangodb/client_access_tokens_impl.go`
(the same shape exists under `v3/arangodb/`):
validate required fields, build `connection.NewUrl`, call `CallPost` /
`CallDelete`, switch on `resp.Code()`, convert failures through
`shared.ResponseStruct`. Document interfaces and types in the matching
`client_access_tokens.go`.

`connection.Call` / `CallWithChecks` create the request, apply
`ArangoDBConfiguration` (driver header, queue timeout, compression, Host
header), then `Do`. `CallStream` is the streaming counterpart; the caller
closes the body.

## Connection, auth and endpoints

`NewHttpConnection` (`connection_http.go`) and `NewHttp2Connection`
(`connection_http2.go`) implement `Connection`. Content type defaults to JSON.
VelocyPack decoding still exists in `decoder.go` but is unmaintained; do not
extend it.

Authentication is set on the connection (`SetAuthentication`) or via wrappers:

- `NewBasicAuth` - HTTP basic
- header auth in `auth_header_impl.go`
- `NewJWTAuthWrapper` - login at `/_open/auth`, attach bearer token, re-auth
  on 401 (see `wrapper_reauthentication.go` and `auth_jwt_impl.go`)

Endpoint selection is pluggable (`Endpoint`): single URL, round-robin
(`endpoints_round_robin.go`), Maglev hash (`endpoints_maglev_hash.go`).
`ArangoDBConfiguration.HostHeader` lets the request `Host` differ from the
dial address (Kubernetes ingress).

Retries (`RetryOn503`, `NewRetryWrapper`) and async jobs
(`NewConnectionAsyncWrapper`) wrap an existing connection. Do not assume every
method is safe to replay.

## API layout

Public surface is interfaces composed onto `Client`, `Database`, `Collection`,
and related handles. Typical file split:

- `foo.go` - interface and exported option/result types
- `foo_impl.go` - unexported struct and HTTP mapping

Optional request fields are pointers with `json:"…,omitempty"` so omission is
distinct from zero values. Path segments that come from user input are escaped
(`url.PathEscape`). Query and header names that the driver owns live in
`v2/arangodb/shared.go` (`QueryWaitForSync`, `HeaderTransaction`, …).

Cursors (`cursor.go`, `cursor_impl.go`) manage batched query results and
cleanup. Follow close, exhaustion, and context cancel paths when changing
them.
