# ArangoDB GO Driver.

NOTE: THIS IS WORK IN PROGRESS.
API and implementation WILL change.

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
