# Examples 

This folder contains various examples about using the ArangoDB go driver.

Most examples assume that you have a single instance ArangoDB server running 
on `http://localhost:8529` without any authentication.

An easy way to run such an instance with docker is:

```
docker run -d -p 8529:8529 -e ARANGO_NO_AUTH=1 arangodb:3.1.11
```

Then you can run the examples with `go run`.

E.g. 

```
go run getting_started.go
```