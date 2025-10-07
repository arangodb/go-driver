# ArangoDB Go Driver

This project contains the official Go driver for the [ArangoDB database system](https://arangodb.com).

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/arangodb/go-driver/tree/master.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/arangodb/go-driver/tree/master)
[![GoDoc](https://godoc.org/github.com/arangodb/go-driver?status.svg)](http://godoc.org/github.com/arangodb/go-driver)

Version 2:
- [Tutorial](https://docs.arangodb.com/stable/develop/drivers/go/)
- [Code examples](v2/examples/)
- [Reference documentation](https://godoc.org/github.com/arangodb/go-driver/v2)

Version 1:
- ⚠️ This version is deprecated and will not receive any new features.
  Please use version 2 ([v2/](v2/)) instead.
- [Tutorial](Tutorial_v1.md)
- [Code examples](examples/)
- [Reference documentation](https://godoc.org/github.com/arangodb/go-driver)

## Supported Go Versions

| Driver        | Go 1.19 | Go 1.20 | Go 1.21 |
|---------------|---------|---------|---------|
| `1.5.0-1.6.1` | ✓       | -       | -       |
| `1.6.2`       | ✓       | ✓       | ✓       |
| `2.1.0`       | ✓       | ✓       | ✓       |
| `master`      | ✓       | ✓       | ✓       |

## Supported ArangoDB Versions

| Driver   | ArangoDB 3.10 | ArangoDB 3.11 | ArangoDB 3.12 |
|----------|---------------|---------------|---------------|
| `1.5.0`  | ✓             | -             | -             |
| `1.6.0`  | ✓             | ✓             | -             |
| `2.1.0`  | ✓             | ✓             | ✓             |
| `master` | +             | +             | +             |

Key:

* `✓` Exactly the same features in both the driver and the ArangoDB version.
* `+` Features included in the driver may be not present in the ArangoDB API.
  Calls to ArangoDB may result in unexpected responses (404).
* `-` The ArangoDB version has features that are not supported by the driver.

## Benchmark Testing

This project includes comprehensive benchmark tests to measure and compare performance between V1 and V2 APIs across all major ArangoDB operations.

### Prerequisites

Before running benchmarks, ensure you have:

1. **Docker** installed and running
2. **Make** installed
3. **Go 1.21+** installed
4. **ArangoDB Enterprise** Docker image available (will be pulled automatically)

### Benchmark Result Files

When you run benchmark comparisons, results are automatically saved to:

- **V1 Results**: `test/v1_benchmarks.txt`
- **V2 Results**: `v2/tests/v2_benchmarks.txt`

These files contain the complete benchmark output and can be used for:
- **Detailed analysis** of individual benchmark results
- **Historical comparison** between different runs
- **Debugging** benchmark failures or performance regressions

### Available Benchmark Names

#### V1 Benchmarks
- `BenchmarkConnectionInitialization`
- `BenchmarkCreateCollection`
- `BenchmarkInsertSingleDocument`
- `BenchmarkInsertBatchDocuments`
- `BenchmarkSimpleQuery`
- `BenchmarkAQLWithBindParameters`
- `BenchmarkCursorIteration`
- `BenchmarkUpdateDocument`
- `BenchmarkDeleteDocument`
- `BenchmarkBatchUpdateDocuments`
- `BenchmarkBatchDeleteDocuments`
- `BenchmarkCreateDocument`
- `BenchmarkReadDocument`
- `BenchmarkCollectionExists`
- `BenchmarkCollections`

#### V2 Benchmarks
- `BenchmarkV2ConnectionInitialization`
- `BenchmarkV2CreateCollection`
- `BenchmarkV2InsertSingleDocument`
- `BenchmarkV2InsertBatchDocuments`
- `BenchmarkV2SimpleQuery`
- `BenchmarkV2AQLWithBindParameters`
- `BenchmarkV2CursorIteration`
- `BenchmarkV2UpdateDocument`
- `BenchmarkV2DeleteDocument`
- `BenchmarkV2BatchUpdateDocuments`
- `BenchmarkV2BatchDeleteDocuments`
- `BenchmarkV2ReadDocument`
- `BenchmarkV2CollectionExists`
- `BenchmarkV2ListCollections`
- `BenchmarkV2DatabaseExists`


### Quick Start

```bash
# Run V1 benchmarks (JSON, no authentication)
make run-benchmarks-single-json-no-auth

# Run V2 benchmarks (no authentication)
make run-v2-benchmarks-single-no-auth

# Compare V1 vs V2 performance
make benchmark-compare

# Run all benchmarks
make benchmark-all
```

### Available Benchmark Commands

#### V1 Benchmarks
```bash
make run-benchmarks-single-json-no-auth     # Single server, JSON, no auth
make run-benchmarks-single-vpack-no-auth    # Single server, Velocypack, no auth
```

#### V2 Benchmarks
```bash
make run-v2-benchmarks-single-no-auth       # Single server, no authentication
make run-v2-benchmarks-single-with-auth     # Single server, with authentication
```

#### Combined Commands
```bash
make run-v2-benchmarks                      # All V2 benchmarks
make run-all-benchmarks                     # V1 + V2 comparison
```

#### Convenient Shortcuts
```bash
make benchmark                              # Quick V2 benchmark (no auth)
make benchmark-v2                           # All V2 benchmarks
make benchmark-all                          # V1 + V2 comparison
make benchmark-compare                      # Side-by-side V1 vs V2 comparison
```

### Benchmark Categories

The benchmark suite covers all major ArangoDB operations:

#### 1. Connection Initialization
- Client connection setup and initialization

#### 2. Collection Operations
- Creating collections
- Checking collection existence
- Listing collections
- Database existence checks

#### 3. Document Operations
- Single document creation, reading, updating, deletion
- Batch document operations (create, read, update, delete)
- Parallel document operations

#### 4. Query Operations
- Simple AQL queries
- AQL queries with bind parameters
- Query validation and explanation

#### 5. Cursor Operations
- Iterating over query results
- Cursor-based data fetching

### Understanding Benchmark Results

#### Key Metrics
- **ns/op** (nanoseconds per operation) - Lower is better
- **B/op** (bytes allocated per operation) - Lower is better
- **allocs/op** (memory allocations per operation) - Lower is better

#### Example Output
```
BenchmarkConnectionInitialization     534    1963648 ns/op
BenchmarkCreateCollection             193    5377251 ns/op
BenchmarkInsertSingleDocument         1044   1178407 ns/op
```

#### Performance Comparison
```bash
make benchmark-compare
```
This command provides side-by-side comparison:
```
=== BENCHMARK COMPARISON RESULTS ===
Connection Initialization:
V1: 1963648 ns/op
V2: 38056 ns/op

Create Collection:
V1: 5377251 ns/op
V2: 3085702 ns/op
```

### Benchmark Files Structure

```
test/
├── benchmark_collection_test.go    # V1 collection and query benchmarks
├── benchmark_document_test.go      # V1 document operation benchmarks
└── v1_benchmarks.txt              # V1 benchmark results (generated)

v2/tests/
├── benchmark_v2_test.go            # V2 comprehensive benchmarks
└── v2_benchmarks.txt              # V2 benchmark results (generated)
```

### Benchmark Result Files

When you run benchmark comparisons, results are automatically saved to:

- **V1 Results**: `test/v1_benchmarks.txt`
- **V2 Results**: `v2/tests/v2_benchmarks.txt`

These files contain the complete benchmark output and can be used for:
- **Detailed analysis** of individual benchmark results
- **Historical comparison** between different runs
- **CI/CD artifact collection** for performance tracking
- **Debugging** benchmark failures or performance regressions

### Running Benchmarks Manually

If you prefer to run benchmarks directly with Go:

```bash
# V1 benchmarks
cd test
go test -bench=. -benchmem -run=^$ -timeout 60m

# V2 benchmarks  
cd v2/tests
go test -bench=. -benchmem -run=^$ -timeout 60m

```

### Troubleshooting

#### Common Issues

1. **Docker not running**
   ```
   Error: Cannot connect to the Docker daemon
   ```
   **Solution**: Start Docker service

2. **Port conflicts**
   ```
   Error: Port 7001 already in use
   ```
   **Solution**: Stop other ArangoDB instances or change ports

3. **Timeout errors**
   ```
   Error: context deadline exceeded
   ```
   **Solution**: Increase timeout or check system resources

4. **Environment variable errors in V2**
   ```
   Error: No endpoints found in environment variable TEST_ENDPOINTS
   ```
   **Solution**: Use Makefile targets instead of direct `go test` commands

#### Performance Tips

1. **Run multiple times** - Benchmark results can vary, run 2-3 times for consistency
2. **Check system resources** - Ensure adequate CPU and memory
3. **Close other applications** - Minimize background processes
4. **Use consistent environment** - Run on the same machine for fair comparison

### Benchmark Development

To add new benchmarks:

1. **V1**: Add to `test/benchmark_*_test.go` files
2. **V2**: Add to `v2/tests/benchmark_v2_test.go`
3. Follow the naming convention: `Benchmark[V2]OperationName`
4. Use appropriate helper functions for setup/cleanup
5. Include memory allocation reporting with `b.ReportAllocs()`

### Example Benchmark Function

```go
func BenchmarkMyOperation(b *testing.B) {
    // Setup
    client := createClient(b, nil)
    db := ensureDatabase(nil, client, "test_db", nil, b)
    defer db.Remove(nil)
    
    col := ensureCollection(nil, db, "test_col", nil, b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Your operation here
        _, err := col.CreateDocument(nil, MyDocument{})
        if err != nil {
            b.Errorf("Operation failed: %s", err)
        }
    }
    b.ReportAllocs()
}
```
