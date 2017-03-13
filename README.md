# ArangoDB GO Driver.

[![GoDoc](https://godoc.org/github.com/arangodb/g-driver?status.svg)](http://godoc.org/github.com/arangodb/go-driver)

NOTE: THIS IS WORK IN PROGRESS.
API and implementation WILL change.

This project contains a Go driver for the [ArangoDB database](https://arangodb.com).

## Supported versions

- ArangoDB versions 3.1 and up.
    - Single server & cluster setups
    - With or without authentication
- Go 1.7 and up.

## Go dependencies 

- None (Additional error libraries are supported).

## Getting started 

Using the driver, you always need to create a `Client`.
The following example shows how to create a `Client` for a single server 
running on localhost.

```
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

```
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
```
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
```
ctx := driver.WithWaitForSync(parentContext)
collection.CreateDocument(ctx, yourDocument)
```

# Sample requests 

## Connecting to ArangoDB

```
client, err := arangodb.NewHTTPClient(arangodb.HTTPClientOptions{
    Endpoints: []string{"https://localhost:8529"},
    Authentication: arangodb.PasswordAuthentication("user", "password"),
    TLSConfig: &tls.Config{ /*...*/ },
})
if err != nil {
    // handle error 
}
```

## Opening a database 

```
ctx := context.Background()
db, err := client.Database(ctx, "myDB")
if err != nil {
    // handle error 
}
```

## Opening a collection

```
ctx := context.Background()
col, err := db.Collection(ctx, "myCollection")
if err != nil {
    // handle error 
}
```

## Checking if a collection exists

```
ctx := context.Background()
found, err := db.CollectionExists(ctx, "myCollection")
if err != nil {
    // handle error 
}
```

## Creating a collection

```
ctx := context.Background()
options := &arangodb.CollectionOptions{ /* ... */ }
col, err := db.CreateCollection(ctx, "myCollection", options)
if err != nil {
    // handle error 
}
```

## Reading a document from a collection 

```
var doc MyDocument 
ctx := context.Background()
meta, err := col.Read(ctx, myDocumentKey, &doc)
if err != nil {
    // handle error 
}
```

## Reading a document from a collection with an explicit revision

```
var doc MyDocument 
revCtx := arangodb.WithRevision(ctx, "mySpecificRevision")
meta, err := col.Read(revCtx, myDocumentKey, &doc)
if err != nil {
    // handle error 
}
```

## Creating a document 

```
doc := MyDocument{
    Name: "jan",
    Counter: 23,
}
ctx := context.Background()
meta, err := col.Create(ctx, doc)
if err != nil {
    // handle error 
}
fmt.Printf("Created document with key '%s', revision '%s'\n", meta.Key, meta.Rev)
```

## Removing a document 

```
ctx := context.Background()
err := col.Remove(revCtx, myDocumentKey)
if err != nil {
    // handle error 
}
```

## Removing a document with an explicit revision

```
revCtx := arangodb.WithRevision(ctx, "mySpecificRevision")
err := col.Remove(revCtx, myDocumentKey)
if err != nil {
    // handle error 
}
```

## Updating a document 

```
ctx := context.Background()
patch := map[string]interface{}{
    "Name": "Frank",
}
meta, err := col.Update(ctx, myDocumentKey, patch)
if err != nil {
    // handle error 
}
```

## Querying documents, one document at a time 

```
ctx := context.Background()
query := "FOR d IN myCollection LIMIT 10 RETURN d"
q, err := col.Query(ctx, query, nil)
if err != nil {
    // handle error 
}
defer q.Close()
for {
    var doc MyDocument 
    meta, err := q.Read(ctx, &doc)
    if arangodb.IsEOF(err) {
        break
    } else if err != nil {
        // handle other errors
    }
    fmt.Printf("Got doc with key '%s' from query\n", meta.Key)
}
```

## Querying documents, multiple document at once

```
ctx := context.Background()
query := "FOR d IN myCollection LIMIT 1000 RETURN d"
q, err := col.Query(ctx, query, nil)
if err != nil {
    // handle error 
}
defer q.Close()
for {
    docs := make([]MyDocument, 10) // read up to 10 at a time
    metas, err := q.ReadMany(ctx, docs)
    if err != nil {
        // handle error 
    }
    if len(metas) == 0 {
        break
    }
    fmt.Printf("Got %d documents from query\n", len(metas))
}
```

## Querying documents, fetching total count

```
ctx := arangodb.WithTotalCount(context.Background())
query := "FOR d IN myCollection RETURN d"
q, err := col.Query(ctx, query, nil)
if err != nil {
    // handle error 
}
defer q.Close()
fmt.Printf("Query yields %d documents\n", q.TotalCount())
```

## Querying documents, with bind variables

```
ctx := context.Background()
query := "FOR d IN myCollection FILTER d.Name == @name RETURN d"
bindVars := map[string]interface{}{
    "name": "Some name",
}
q, err := col.Query(ctx, query, bindVars)
if err != nil {
    // handle error 
}
defer q.Close()
...
```

# Use of context 

- All functions get a `context.Context` as first argument. 
  This can be `nil`. Passing `nil` is the same as passing `context.Background()`.
- All often used arguments are passed in the function arguments. 
  Special options are passed using explicitly types fields in the context using 
  an `arangodb.WithXyz` function.

# Failover 

The client supports multiple endpoints to connect to. All request are in principle 
send to the same endpoint until that endpoint fails to respond. 
In that case a new endpoint is chosen and the operation is retried.

# Interfaces 

All API's of the client are Go interfaces. 
This allows them to be implemented using HTTP or Velocypack.

E.g. 

```
type Client interface {
    // Database opens a connection to an existing database.
    Database(ctx context.Context, name string) (Database, error)

    // CreateDatabase creates a new connection with given name and opens a connection to it.
    CreateDatabase(ctx context.Context, name string) (Database, error)

    // CreateUser creates a new user in the system.
    CreateUser(ctx context.Context, name string) (User, error)
}

type Database interface {
    // Collection opens an existing collection with given name.
    Collection(ctx context.Context, name string) (Collection, error)

    // Collections returns a list of all collections in the database.
    Collections(ctx context.Context) ([]Collection, error)

    // Graphs returns a list of all graphs in the database.
    Graphs(ctx context.Context) ([]Graph, error)
}

type Collection interface {
    // Read reads a single document with given key.
    // The resulting document data is stored into result.
    Read(ctx context.Context, key string, result interface{}) (DocumentMeta, error)
}
```

# Errors 

All methods that at some point in their life can fail must return an `error`.
The type of error is always structured and can be query with `IsXyz(err) bool` functions.

E.g. 
```
func IsNotFound(err error) bool
func IsEOF(err error) bool
func IsDuplicateKey(err error) bool
```

If a function returns a non-nil error, all other return values are considered invalid.
