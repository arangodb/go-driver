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

## Running Benchmark Tests

The go-driver includes comprehensive benchmark tests to measure performance of both V1 and V2 APIs. These benchmarks help compare performance between different driver versions and HTTP protocols.

### Prerequisites

- Docker (for running ArangoDB test instances)
- Go 1.19+ 
- Make

### Available Benchmark Tests

#### V1 API Benchmarks
- **Collection Operations**: `BenchmarkCollectionExists`, `BenchmarkCollection`, `BenchmarkCollections`
- **Document Operations**: `BenchmarkCreateDocument`, `BenchmarkReadDocument`, `BenchmarkRemoveDocument`
- **Parallel Operations**: `BenchmarkCreateDocumentParallel`, `BenchmarkReadDocumentParallel`
- **Comprehensive CRUD**: `BenchmarkComprehensiveDocumentOperations_1K`, `BenchmarkComprehensiveDocumentOperations_10K`
- **Performance Tests**: `Benchmark_Insert`, `Benchmark_BatchInsert`

#### V2 API Benchmarks
- **Collection Operations**: `BenchmarkV2CollectionExists`, `BenchmarkV2Collection`, `BenchmarkV2Collections`
- **Document Operations**: `BenchmarkV2CreateDocument`, `BenchmarkV2ReadDocument`, `BenchmarkV2RemoveDocument`
- **Parallel Operations**: `BenchmarkV2CreateDocumentParallel`, `BenchmarkV2ReadDocumentParallel`
- **Comprehensive CRUD**: `BenchmarkV2ComprehensiveDocumentOperations_1K`, `BenchmarkV2ComprehensiveDocumentOperations_10K`
- **Performance Tests**: `Benchmark_Insert`, `Benchmark_BatchInsert`

### Running Benchmarks

#### Basic Benchmark Execution

```bash
# Run all V1 benchmarks
make run-benchmarks-single-json-no-auth

# Run all V2 benchmarks  
make run-benchmarks-v2-single-json-no-auth
```

#### Running Specific Benchmarks

```bash
# V1 API - Specific benchmark
export TESTOPTIONS="-bench=BenchmarkCreateDocument -test.v"
make run-benchmarks-single-json-no-auth

# V2 API - Specific benchmark
export TESTOPTIONS="-bench=BenchmarkV2CreateDocument -test.v"
make run-benchmarks-v2-single-json-no-auth

# V1 API - Comprehensive CRUD with 10K documents
export TESTOPTIONS="-bench=BenchmarkComprehensiveDocumentOperations_10K -test.v"
make run-benchmarks-single-json-no-auth

# V2 API - Comprehensive CRUD with 10K documents
export TESTOPTIONS="-bench=BenchmarkV2ComprehensiveDocumentOperations_10K -test.v"
make run-benchmarks-v2-single-json-no-auth
```

#### Running Multiple Benchmarks

```bash
# Run multiple V2 benchmarks
export TESTOPTIONS="-bench=BenchmarkV2CreateDocument|BenchmarkV2ReadDocument -test.v"
make run-benchmarks-v2-single-json-no-auth

# Run both 1K and 10K comprehensive benchmarks
export TESTOPTIONS="-bench=BenchmarkV2ComprehensiveDocumentOperations_1K|BenchmarkV2ComprehensiveDocumentOperations_10K -test.v"
make run-benchmarks-v2-single-json-no-auth
```

#### Comparing V1 vs V2 Performance

```bash
# Run V1 10K comprehensive benchmark
export TESTOPTIONS="-bench=BenchmarkComprehensiveDocumentOperations_10K -test.v"
make run-benchmarks-single-json-no-auth

# Run V2 10K comprehensive benchmark
export TESTOPTIONS="-bench=BenchmarkV2ComprehensiveDocumentOperations_10K -test.v"
make run-benchmarks-v2-single-json-no-auth
```

### Benchmark Output Explanation

The benchmark output includes:

- **Operations per second**: How many operations completed in the benchmark duration
- **Time per operation**: Average time per operation (e.g., `720955 ns/op`)
- **Memory allocations**: Bytes allocated per operation (e.g., `9505 B/op`)
- **Allocation count**: Number of memory allocations per operation (e.g., `114 allocs/op`)
- **CPU scaling**: Results for different CPU counts (1, 2, 4 cores)

Example output:
```
BenchmarkV2CreateDocument/HTTP_JSON-4    1197    1055266 ns/op    10713 B/op    128 allocs/op
```

This means:
- 1197 operations completed
- 1,055,266 nanoseconds per operation (~1.06ms)
- 10,713 bytes allocated per operation
- 128 memory allocations per operation
- Tested with 4 CPU cores

### HTTP Protocol Comparison

V2 benchmarks automatically test both HTTP/1.1 and HTTP/2 protocols:

```
BenchmarkV2CreateDocument/HTTP_JSON      # HTTP/1.1 results
BenchmarkV2CreateDocument/HTTP2_JSON     # HTTP/2 results
```

### Benchmark Options

You can customize benchmark execution with additional options:

```bash
# Set benchmark duration
export TESTOPTIONS="-bench=BenchmarkV2CreateDocument -benchtime=10s -test.v"

# Set number of iterations
export TESTOPTIONS="-bench=BenchmarkV2CreateDocument -count=5 -test.v"

# Run with memory profiling
export TESTOPTIONS="-bench=BenchmarkV2CreateDocument -memprofile=mem.prof -test.v"

# Run with CPU profiling  
export TESTOPTIONS="-bench=BenchmarkV2CreateDocument -cpuprofile=cpu.prof -test.v"
```

### Performance Tips

1. **Use specific benchmark names** to avoid running all tests
2. **Compare HTTP/1.1 vs HTTP/2** performance in V2 benchmarks
3. **Test with different document counts** (1K vs 10K) to understand scaling
4. **Run multiple iterations** for more reliable results
5. **Use profiling options** to identify performance bottlenecks

### Troubleshooting

- **"No endpoints found"**: Ensure Docker is running and the test environment is properly set up
- **Benchmark not found**: Check the exact function name (case-sensitive)
- **All benchmarks running**: Use specific benchmark names in `TESTOPTIONS`
- **HTTP/2 errors**: V2 benchmarks automatically handle HTTP/2 stream limits
