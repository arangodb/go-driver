# ArangoDB GO Driver.

[![Build Status](https://travis-ci.org/arangodb/go-driver.svg?branch=master)](https://travis-ci.org/arangodb/go-driver)
[![GoDoc](https://godoc.org/github.com/arangodb/g-driver?status.svg)](http://godoc.org/github.com/arangodb/go-driver)

API and implementation is considered stable, more protocols (Velocystream) are being added within the existing API.

This project contains a Go driver for the [ArangoDB database](https://arangodb.com).

## Supported versions

- ArangoDB versions 3.1 and up.
    - Single server & cluster setups
    - With or without authentication
- Go 1.7 and up.

## Go dependencies 

- None (Additional error libraries are supported).

## Getting started 

To use the driver, first fetch the sources into your GOPATH.

```sh
go get github.com/arangodb/go-driver
```

Using the driver, you always need to create a `Client`.
The following example shows how to create a `Client` for a single server 
running on localhost.

```go
import (
	"fmt"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

...

conn, err := http.NewConnection(http.ConnectionConfig{
    Endpoints: []string{"http://localhost:8529"},
})
if err != nil {
    // Handle error
}
c, err := driver.NewClient(driver.ClientConfig{
    Connection: conn,
})
if err != nil {
    // Handle error
}
```

Once you have a `Client` you can access/create databases on the server, 
access/create collections, graphs, documents and so on.

The following example shows how to open an existing collection in an existing database 
and create a new document in that collection.

```go
// Open "examples_books" database
db, err := c.Database(nil, "examples_books")
if err != nil {
    // Handle error
}

// Open "books" collection
col, err := db.Collection(nil, "books", nil)
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
fmt.Printf("Created document in collection '%s' in database '%s'\n", col.Name(), db.Name())
```

## API design 

### Concurrency

All functions of the driver are stricly synchronous. They operate and only return a value (or error)
when they're done. 

If you want to run operations concurrently, use a go routine. All objects in the driver are designed 
to be used from multiple concurrent go routines, except `Cursor`.

All database objects (except `Cursor`) are considered static. After their creation they won't change.
E.g. after creating a `Collection` instance you can remove the collection, but the (Go) instance 
will still be there. Calling functions on such a removed collection will of course fail.

### Structured error handling & wrapping

All functions of the driver that can fail return an `error` value. If that value is not `nil`, the 
function call is considered to be failed. In that case all other return values are set to their `zero` 
values.

All errors are structured using error checking functions named `Is<SomeErrorCategory>`.
E.g. `IsNotFound(error)` return true if the given error is of the category "not found". 
There can be multiple internal error codes that all map onto the same category.

All errors returned from any function of the driver (either internal or exposed) wrap errors 
using the `WithStack` function. This can be used to provide detail stack trackes in case of an error.
All error checking functions use the `Cause` function to get the cause of an error instead of the error wrapper.

Note that `WithStack` and `Cause` are actually variables to you can implement it using your own error 
wrapper library. 

If you for example use https://github.com/pkg/errors, you want to initialize to go driver like this:
```go
import (
    driver "github.com/arangodb/go-driver"
    "github.com/pkg/errors"
)

func init() {
    driver.WithStack = errors.WithStack 
    driver.Cause = errors.Cause
}
```

### Context aware 

All functions of the driver that involve some kind of long running operation or 
support additional options not given as function arguments, have a `context.Context` argument. 
This enables you cancel running requests, pass timeouts/deadlines and pass additional options.

In all methods that take a `context.Context` argument you can pass `nil` as value. 
This is equivalent to passing `context.Background()`.

Many functions support 1 or more optional (and infrequently used) additional options.
These can be used with a `With<OptionName>` function.
E.g. to force a create document call to wait until the data is synchronized to disk, 
use a prepared context like this:
```go
ctx := driver.WithWaitForSync(parentContext)
collection.CreateDocument(ctx, yourDocument)
```

### Failover 

The driver supports multiple endpoints to connect to. All request are in principle 
send to the same endpoint until that endpoint fails to respond. 
In that case a new endpoint is chosen and the operation is retried.

The following example shows how to connect to a cluster of 3 servers.

```go
conn, err := http.NewConnection(http.ConnectionConfig{
    Endpoints: []string{"http://server1:8529", "http://server2:8529", "http://server3:8529"},
})
if err != nil {
    // Handle error
}
c, err := driver.NewClient(driver.ClientConfig{
    Connection: conn,
})
if err != nil {
    // Handle error
}
```

Note that a valid endpoint is an URL to either a standalone server, or a URL to a coordinator 
in a cluster.

### Failover: Exact behavior

The driver monitors the request being send to a specific server (endpoint). 
As soon as the request has been completely written, failover will no longer happen.
The reason for that is that several operations cannot be (safely) retried.
E.g. when a request to create a document has been send to a server and a timeout 
occurs, the driver has no way of knowing if the server did or did not create
the document in the database.

If the driver detects that a request has been completely written, but still gets 
an error (other than an error response from Arango itself), it will wrap the 
error in a `ResponseError`. The client can test for such an error using `IsResponseError`.

If a client received a `ResponseError`, it can do one of the following:
- Retry the operation and be prepared for some kind of duplicate record / unique constraint violation.
- Perform a test operation to see if the "failed" operation did succeed after all.
- Simply consider the operation failed. This is risky, since it can still be the case that the operation did succeed.

### Failover: Timeouts

To control the timeout of any function in the driver, you must pass it a context 
configured with `context.WithTimeout` (or `context.WithDeadline`).

In the case of multiple endpoints, the actual timeout used for requests will be shorter than 
the timeout given in the context.
The driver will divide the timeout by the number of endpoints with a maximum of 3.
This ensures that the driver can try up to 3 different endpoints (in case of failover) without 
being canceled due to the timeout given by the client.
E.g.
- With 1 endpoint and a given timeout of 1 minute, the actual request timeout will be 1 minute.
- With 3 endpoints and a given timeout of 1 minute, the actual request timeout will be 20 seconds.
- With 8 endpoints and a given timeout of 1 minute, the actual request timeout will be 20 seconds.

For most requests you want a actual request timeout of at least 30 seconds.

### Secure connections (SSL)

The driver supports endpoints that use SSL using the `https` URL scheme.

The following example shows how to connect to a server that has a secure endpoint using 
a self-signed certificate.

```go
conn, err := http.NewConnection(http.ConnectionConfig{
    Endpoints: []string{"https://localhost:8529"},
    TLSConfig: &tls.Config{InsecureSkipVerify: true},
})
if err != nil {
    // Handle error
}
c, err := driver.NewClient(driver.ClientConfig{
    Connection: conn,
})
if err != nil {
    // Handle error
}
```

# Sample requests 

## Connecting to ArangoDB

```go
conn, err := http.NewConnection(http.ConnectionConfig{
    Endpoints: []string{"http://localhost:8529"},
    TLSConfig: &tls.Config{ /*...*/ },
})
if err != nil {
    // Handle error
}
c, err := driver.NewClient(driver.ClientConfig{
    Connection: conn,
    Authentication: driver.BasicAuthentication("user", "password"),
})
if err != nil {
    // Handle error
}
```

## Opening a database 

```go
ctx := context.Background()
db, err := client.Database(ctx, "myDB")
if err != nil {
    // handle error 
}
```

## Opening a collection

```go
ctx := context.Background()
col, err := db.Collection(ctx, "myCollection")
if err != nil {
    // handle error 
}
```

## Checking if a collection exists

```go
ctx := context.Background()
found, err := db.CollectionExists(ctx, "myCollection")
if err != nil {
    // handle error 
}
```

## Creating a collection

```go
ctx := context.Background()
options := &driver.CreateCollectionOptions{ /* ... */ }
col, err := db.CreateCollection(ctx, "myCollection", options)
if err != nil {
    // handle error 
}
```

## Reading a document from a collection 

```go
var doc MyDocument 
ctx := context.Background()
meta, err := col.ReadDocument(ctx, myDocumentKey, &doc)
if err != nil {
    // handle error 
}
```

## Reading a document from a collection with an explicit revision

```go
var doc MyDocument 
revCtx := driver.WithRevision(ctx, "mySpecificRevision")
meta, err := col.ReadDocument(revCtx, myDocumentKey, &doc)
if err != nil {
    // handle error 
}
```

## Creating a document 

```go
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

## Removing a document 

```go
ctx := context.Background()
err := col.RemoveDocument(revCtx, myDocumentKey)
if err != nil {
    // handle error 
}
```

## Removing a document with an explicit revision

```go
revCtx := driver.WithRevision(ctx, "mySpecificRevision")
err := col.RemoveDocument(revCtx, myDocumentKey)
if err != nil {
    // handle error 
}
```

## Updating a document 

```go
ctx := context.Background()
patch := map[string]interface{}{
    "Name": "Frank",
}
meta, err := col.UpdateDocument(ctx, myDocumentKey, patch)
if err != nil {
    // handle error 
}
```

## Querying documents, one document at a time 

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

## Querying documents, fetching total count

```go
ctx := driver.WithQueryCount(context.Background())
query := "FOR d IN myCollection RETURN d"
cursor, err := db.Query(ctx, query, nil)
if err != nil {
    // handle error 
}
defer cursor.Close()
fmt.Printf("Query yields %d documents\n", cursor.Count())
```

## Querying documents, with bind variables

```go
ctx := context.Background()
query := "FOR d IN myCollection FILTER d.Name == @name RETURN d"
bindVars := map[string]interface{}{
    "name": "Some name",
}
cursor, err := db.Query(ctx, query, bindVars)
if err != nil {
    // handle error 
}
defer cursor.Close()
...
```

