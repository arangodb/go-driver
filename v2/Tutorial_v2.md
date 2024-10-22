# Tutorial for the Go driver version 1

## Install the driver

To use the driver, fetch the sources into your `GOPATH` first.

```sh
go get github.com/arangodb/go-driver/v2
```

Import the driver in your Go program using the `import` statement.
Two packages are necessary:

```go
import (
    "github.com/arangodb/go-driver/v2/arangodb"
    "github.com/arangodb/go-driver/v2/connection"
)
```

If you use Go modules, you can also import the driver and run `go mod tidy`
instead of using the `go get` command.

## Connect to ArangoDB

Using the driver, you always need to create a `Client`. The following example
shows how to create a `Client` for an ArangoDB single server running on localhost.
For more options, see [Connection management](#connection-management).

```go
import (
    "context"
    "log"

    "github.com/arangodb/go-driver/v2/arangodb"
    "github.com/arangodb/go-driver/v2/connection"
)

/*...*/

endpoint := connection.NewRoundRobinEndpoints([]string{"http://localhost:8529"})
conn := connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, true))

// Add authentication
auth := connection.NewBasicAuth("root", "")
err := conn.SetAuthentication(auth)
if err != nil {
    log.Fatalf("Failed to set authentication: %v", err)
}

// Create a client
client := arangodb.NewClient(conn)
```

Once you have a `Client` object, you can use this handle to create and edit
objects, such as databases, collections, documents, and graphs. These database
objects are mapped to types in Go. The methods for these types are used to read
and write data.

### Asynchronous client

The driver supports an asynchronous client that can be used to run multiple operations concurrently.


```go
import (
    "context"
    "log"
    
    "github.com/arangodb/go-driver/v2/arangodb"
    "github.com/arangodb/go-driver/v2/connection"
)

/*...*/

// Create an HTTP connection to the database
endpoint := connection.NewRoundRobinEndpoints([]string{"http://localhost:8529"})
conn := connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, true))

auth := connection.NewBasicAuth("root", "")
err := conn.SetAuthentication(auth)
if err != nil {
    log.Fatalf("Failed to set authentication: %v", err)
}

// Create ASYNC wrapper for the connection
conn = connection.NewConnectionAsyncWrapper(conn)

// Create a client
client := arangodb.NewClient(conn)

// Trigger async request
info, err := client.Version(connection.WithAsync(context.Background()))
if err != nil {
    log.Printf("this is expected error since we are using async mode and response is not ready yet: %v", err)
}
if info.Version != "" {
    log.Printf("Expected empty version if async request is in progress, got %s", info.Version)
}

// Fetch an async job id from the error
id, isAsyncId := connection.IsAsyncJobInProgress(err)
if !isAsyncId {
    log.Fatalf("Expected async job id, got %v", id)
}

// Wait for an async result
time.Sleep(3 * time.Second)

// List async jobs - there should be one, till the result is fetched
jobs, err := client.AsyncJobList(context.Background(), arangodb.JobDone, nil)
if err != nil {
    log.Fatalf("Failed to list async jobs: %v", err)
}
if len(jobs) != 1 {
    log.Fatalf("Expected 1 async job, got %d", len(jobs))
}

// Fetch an async job result
info, err = client.Version(connection.WithAsyncID(context.Background(), id))
if err != nil {
    log.Fatalf("Failed to fetch async job result: %v", err)
}
log.Printf("Async job result: %s", info.Version)
```

## Important types for Go

Key types you need to know about to work with ArangoDB using the Go driver:

- `Database` – to maintain a handle to an open database
- `Collection` – as a handle for a collection of records (vertex, edge, or document) within a database
- `Graph` – as a handle for a graph overlay containing vertices and edges (nodes and links)
- `EdgeDefinition` – a named collection of edges used to help a graph in distributed searching

These are declared as in the following examples:

```go
var err error
var client arangodb.Client
var conn   connection.Connection
var db     arangodb.Database
var col    arangodb.Collection
```

The following example shows how to open an existing collection in an existing
database and create a new document in that collection.

```go
// Setup a client connection
endpoint := connection.NewRoundRobinEndpoints([]string{"https://5a812333269f.arangodb.cloud:8529/"})
conn := connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, false))

// Add authentication
auth := connection.NewBasicAuth("root", "wnbGnPpCXHwbP")
err := conn.SetAuthentication(auth)
if err != nil {
    log.Fatalf("Failed to set authentication: %v", err)
}

// Create a client
client := arangodb.NewClient(conn)

// Open "examples_books" database
db, err := client.Database(nil, "examples_books")
if err != nil {
    // Handle error
}

// Open "books" collection
col, err := db.Collection(nil, "books")
if err != nil {
    // Handle error
}

// Create document
book := Book{
    Title:   "ArangoDB Cookbook",
    NoPages: 257,
}

meta, err := col.CreateDocument(nil, book)
if err != nil {
    // Handle error
}
log.Printf("Created document in collection '%s' in database '%s'\n", col.Name(), db.Name())
```

## Relationships between Go types and JSON

A basic principle of the integration between Go and ArangoDB is the mapping from
Go types to JSON documents. Data in the database map to types in Go through JSON.
You need at least two types in a Golang program to work with graphs.

Go uses a special syntax to map values like struct members like Key, Weight,
Data, etc. to JSON fields. Remember that member names should start with a capital
letter to be accessible outside a packaged scope. You declare types and their
JSON mappings once, as in the examples below.

```go
// A typical document type
type IntKeyValue struct {
    Key    string  `json:"_key"`    // mandatory field (handle) - short name
    Value  int     `json:"value"`
}

// A typical vertex type must have field matching _key
type MyVertexNode struct {
    Key     string    `json:"_key"` // mandatory field (handle) - short name
    // other fields … e.g.
    Data    string `json: "data"`   // Longer description or bulk string data
    Weight float64 `json:"weight"`  // importance rank
}

// A typical edge type must have fields matching _from and _to
type MyEdgeLink struct {
    Key       string `json:"_key"`  // mandatory field (handle)
    From      string `json:"_from"` // mandatory field
    To        string `json:"_to"`   // mandatory field
    // other fields … e.g.
    Weight  float64 `json:"weight"`
}
```

When reading data from ArangoDB with, say, `ReadDocument()`, the API asks you to
submit a variable of some type, say `MyDocumentType`, by reference using the
`&` operator:

```go
var variable MyDocumentType
mycollection.ReadDocument(nil, rawkey, &variable)
```

This submitted type is not necessarily a fixed type, but it must be a type whose
members map (at least partially) to the named fields in the database's JSON
document representation. Only matching fields are filled in. This means you could
create several different Go types to read the same documents in the database, as
long as they have some type fields that match JSON fields. In other words, the
mapping need not be unique or one-to-one, so there is great flexibility in making
new types to extract a subset of the fields in a document.

The document model in ArangoDB does not require all documents in a collection to
have the same fields. You can choose to have ad hoc schemas and extract only a
consistent set of fields in a query, or rigidly check that all documents have
the same schema. This is a user choice.

## Working with databases

### Create a new database

```go
ctx := context.Background()
options := arangodb.CreateDatabaseOptions{ /*...*/ }
db, err := client.CreateDatabase(ctx, "myDB", &options)
if err != nil {
    // handle error 
}
```

### Open a database

```go
ctx := context.Background()
db, err := client.Database(ctx, "myDB")
if err != nil {
    // handle error 
}
```

## Working with collections

### Create a collection

```go
ctx := context.Background()
options := arangodb.CreateCollectionProperties{ /* ... */ }
col, err := db.CreateCollection(ctx, "myCollection", &options)
if err != nil {
    // handle error 
}
```
### Check if a collection exists

```go
ctx := context.Background()
found, err := db.CollectionExists(ctx, "myCollection")
if err != nil {
    // handle error 
}
```

### Open a collection

```go
ctx := context.Background()
col, err := db.Collection(ctx, "myCollection")
if err != nil {
    // handle error 
}
```

## Working with documents

### Create a document

```go
type MyDocument struct {
    Name    string `json:"name"`
    Counter int    `json:"counter"`
}

doc := MyDocument{
    Name: "jan",
    Counter: 23,
}
ctx := context.Background()
meta, err := col.CreateDocument(ctx, doc)
if err != nil {
    // handle error 
}
fmt.Printf("Created document with key '%s', revision '%s'\n", meta.Key, meta.Rev)
```

### Read a document 

```go
var result MyDocument 
ctx := context.Background()
meta, err := col.ReadDocument(ctx, "myDocumentKey (meta.Key)", &result)
if err != nil {
    // handle error 
}
```

### Read a document with an explicit revision

```go
var doc MyDocument
ctx := context.Background()
options := arangodb.CreateDatabaseOptions{
    IfMatch: "mySpecificRevision (meta.Rev)",
}
meta, err := col.ReadDocumentWithOptions(revCtx, "myDocumentKey (meta.Key)", &doc, &options)
if err != nil {
    // handle error 
}
```

### Delete a document

```go
ctx := context.Background()
meta, err := col.DeleteDocument(ctx, myDocumentKey)
if err != nil {
    // handle error 
}
```

### Delete a document with an explicit revision

```go
ctx := context.Background()
options := arangodb.CollectionDocumentDeleteOptions{
    IfMatch: "mySpecificRevision (meta.Rev)",
}
meta, err := col.DeleteDocumentWithOptions(ctx, myDocumentKey, &options)
if err != nil {
    // handle error 
}
```

### Update a document

```go
ctx := context.Background()
patch := map[string]interface{}{
    "name": "Frank",
}
meta, err := col.UpdateDocument(ctx, myDocumentKey, patch)
if err != nil {
    // handle error 
}
```

## Working with AQL

### Query documents, one document at a time

```go
ctx := context.Background()
query := "FOR d IN myCollection LIMIT 10 RETURN d"
cursor, err := db.Query(ctx, query, nil)
if err != nil {
    // handle error 
}
defer cursor.Close()
for {
    var doc MyDocument 
    meta, err := cursor.ReadDocument(ctx, &doc)
    if driver.IsNoMoreDocuments(err) {
        break
    } else if err != nil {
        // handle other errors
    }
    fmt.Printf("Got doc with key '%s' from query\n", meta.Key)
}
```

### Query documents, fetching the total count

```go
ctx := context.Background()
options := arangodb.QueryOptions{
    Count: true,
}

query := "FOR d IN myCollection RETURN d"
cursor, err := db.Query(ctx, query, &options)
if err != nil {
    // handle error 
}
defer cursor.Close()
fmt.Printf("Query yields %d documents\n", cursor.Count())
```

### Query documents, with bind variables

```go
ctx := driver.WithQueryCount(context.Background())

ctx := context.Background()
options := arangodb.QueryOptions{
    Count: true,
    BindVars: map[string]interface{}{
        "myVar": "Some name",
    }
}

query := "FOR d IN myCollection FILTER d.name == @myVar RETURN d"

cursor, err := db.Query(ctx, query, &options)
if err != nil {
    // handle error 
}
defer cursor.Close()
fmt.Printf("Query yields %d documents\n", cursor.Count())
```

## Full example

```go
package main

import (
  "context"
  "flag"
  "fmt"
  "log"
  "strings"

  "github.com/arangodb/go-driver/v2/arangodb"
  "github.com/arangodb/go-driver/v2/arangodb/shared"
  "github.com/arangodb/go-driver/v2/connection"
)

type User struct {
  Name string `json:"name"`
  Age  int    `json:"age"`
}

func main() {
  // Create an HTTP connection to the database
  endpoint := connection.NewRoundRobinEndpoints([]string{"http://localhost:8529"})
  conn := connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, true))

  auth := connection.NewBasicAuth("root", "")
  err := conn.SetAuthentication(auth)
  if err != nil {
    log.Fatalf("Failed to set authentication: %v", err)
  }

  // Create a client
  client := arangodb.NewClient(conn)
  ctx := context.Background()

  flag.Parse()

  var db arangodb.Database
  var dbExists, collExists bool

  dbExists, err = client.DatabaseExists(ctx, "example")

  if dbExists {
    fmt.Println("That db exists already")

    db, err = client.Database(ctx, "example")

    if err != nil {
      log.Fatalf("Failed to open existing database: %v", err)
    }
  } else {
    db, err = client.CreateDatabase(ctx, "example", nil)

    if err != nil {
      log.Fatalf("Failed to create database: %v", err)
    }
  }

  // Create collection
  collExists, err = db.CollectionExists(ctx, "users")

  if collExists {
    fmt.Println("That collection exists already")
  } else {
    var col arangodb.Collection
    col, err = db.CreateCollection(ctx, "users", nil)

    if err != nil {
      log.Fatalf("Failed to create collection: %v", err)
    }

    // Create documents
    users := []User{
      {
        Name: "John",
        Age:  65,
      },
      {
        Name: "Tina",
        Age:  25,
      },
      {
        Name: "George",
        Age:  31,
      },
    }
    reader, err := col.CreateDocuments(ctx, users)
    if err != nil {
      log.Fatalf("Failed to create documents: %v", err)
    }

    meta1, err := reader.Read()
    if err != nil {
      log.Fatalf("Failed to read document: %v", err)
    }
    meta2, err := reader.Read()
    if err != nil {
      log.Fatalf("Failed to read document: %v", err)
    }
    meta3, err := reader.Read()
    if err != nil {
      log.Fatalf("Failed to read document: %v", err)
    }

    keys := []string{meta1.Key, meta2.Key, meta3.Key}

    fmt.Printf("Created documents with keys '%s' in collection '%s' in database '%s'\n", strings.Join(keys, ","), col.Name(), db.Name())
  }
  PrintCollection(db)
}

func PrintCollection(db arangodb.Database) {
  var err error
  var cursor arangodb.Cursor

  querystring := "FOR doc IN users LIMIT 10 RETURN doc"

  cursor, err = db.Query(nil, querystring, nil)

  if err != nil {
    log.Fatalf("Query failed: %v", err)
  }

  defer cursor.Close()

  for {
    var doc User
    var metadata arangodb.DocumentMeta

    metadata, err = cursor.ReadDocument(nil, &doc)

    if shared.IsNoMoreDocuments(err) {
      break
    } else if err != nil {
      log.Fatalf("Doc returned: %v", err)
    } else {
      fmt.Print("Dot doc ", metadata, doc, "\n")
    }
  }
}
```

## API Design

### Concurrency

All functions of the driver are strictly synchronous. They operate and only
return a value (or error) when they are done.

If you want to run operations concurrently, use a `go` routine. All objects in
the driver are designed to be used from multiple concurrent go routines,
except `Cursor`.

All database objects (except `Cursor`) are considered static. After their
creation, they don't change. For example, after creating a `Collection` instance,
you can remove the collection, but the (Go) instance will still be there. Calling
functions on such a removed collection will, of course, fail.

### Structured error handling & wrapping

All functions of the driver that can fail return an error value. If that value
is not `nil`, the function call is considered to have failed. In that case, all
other return values are set to their zero values.

All errors are structured using error-checking functions named
`Is<SomeErrorCategory>`. For example, `IsNotFound(error)` returns true if the
given error is of the category "not found". There can be multiple internal error
codes that all map onto the same category.

All errors returned from any function of the driver (either internal or exposed)
wrap errors using the `WithStack` function. This can be used to provide detailed
stack traces in case of an error. All error-checking functions use the `Cause`
function to get the cause of an error instead of the error wrapper.

### Context-aware

All functions of the driver that involve some kind of long-running operation or
support additional options are not given as function arguments have a
`context.Context` argument. This enables you to cancel running requests, pass
timeouts/deadlines, and pass additional options.

## Connection management

### Secure connections (TLS)

The driver supports endpoints that use TLS using the `https` URL scheme.
You can specify a TLS configuration when creating a connection configuration.

```go
import (
    /*...*/
  "github.com/arangodb/go-driver/v2/connection"
)

/*...*/

endpoint := connection.NewRoundRobinEndpoints([]string{"https://localhost:8529"})
conn := connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, false))
```

If you want to connect to a server that has a secure endpoint using a
self-signed certificate, use `TLSConfig: &tls.Config{InsecureSkipVerify: true},`.

### Connection Pooling

The driver has a built-in connection pooling, and the connection limit
(`connLimit`) defaults to `32`.

```go
conn, err := http.NewConnection(http.ConnectionConfig{
    Endpoints: []string{"https://localhost:8529"},
    connLimit: 32,
})
```

Opening and closing connections very frequently can exhaust the number of
connections allowed by the operating system. TCP connections enter a special
state `WAIT_TIME` after close and typically remain in this state for two minutes
(maximum segment life \* 2). These connections count towards the global limit,
which depends on the operating system but is usually around 28,000. Connections
should thus be reused as much as possible.

You may run into this problem if you bypass the driver's safeguards by setting a
very high connection limit or by using multiple connection objects and thus pools.

### Endpoints management

The driver supports multiple endpoints to connect to.
Currently Maglev and RoundRobin approaches are supported.

The following example shows how to connect to a cluster of three servers using RoundRobin:

```go
endpoint := connection.NewRoundRobinEndpoints([]string{"http://server1:8529", "http://server2:8529", "http://server3:8529"})
conn := connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, true))
client := arangodb.NewClient(conn)

```

Note that a valid endpoint is a URL to either a standalone server or a URL to a
Coordinator in a cluster.

#### Exact behavior

The driver monitors the request being sent to a specific server (endpoint).
As soon as the request has been completely written, failover will no longer
happen. The reason for that is that several operations cannot be (safely) retried.
For example, when a request to create a document has been sent to a server, and
a timeout occurs, the driver has no way of knowing if the server did or did not
create the document in the database.

If the driver detects that a request has been completely written but still gets
an error (other than an error response from ArangoDB itself), it wraps the error
in a `ResponseError`. The client can test for such an error using `IsResponseError`.

If a client receives a `ResponseError`, it can do one of the following:

- Retry the operation and be prepared for some kind of duplicate record or
  unique constraint violation.
- Perform a test operation to see if the "failed" operation did succeed after all.
- Simply consider the operation failed. This is risky since it can still be the
  case that the operation did succeed.

#### Timeouts

To control the timeout of any function in the driver, you must pass it a context
configured with `context.WithTimeout` (or `context.WithDeadline`).

In the case of multiple endpoints, the actual timeout used for requests is
shorter than the timeout given in the context. The driver divides the timeout by
the number of endpoints with a maximum of `3`. This ensures that the driver can
try up to 3 different endpoints (in case of failover) without being canceled due
to the timeout given by the client. Examples:

- With 1 endpoint and a given timeout of 1 minute, the actual request timeout is 1 minute.
- With 3 endpoints and a given timeout of 1 minute, the actual request timeout is 20 seconds.
- With 8 endpoints and a given timeout of 1 minute, the actual request timeout is 20 seconds.

For most requests, you want an actual request timeout of at least 30 seconds.
