# ArangoDB Go Driver Benchmarks Documentation

This document provides step-by-step instructions for running and understanding the benchmark tests in the ArangoDB Go Driver project.

## Table of Contents
- [Overview](#overview)
- [Benchmark Structure](#benchmark-structure)
- [Prerequisites](#prerequisites)
- [Running Local Benchmarks](#running-local-benchmarks)
- [Running Remote Benchmarks](#running-remote-benchmarks)
- [Understanding Results](#understanding-results)
- [Performance Comparison: V1 vs V2](#performance-comparison-v1-vs-v2)
- [Benchmark Operations](#benchmark-operations)
- [Configuration Options](#configuration-options)
- [Troubleshooting](#troubleshooting)
- [Database and Collection Resources](#database-and-collection-resources)
- [Makefile Targets Reference](#makefile-targets-reference)
- [Contributing](#contributing)

---

## Overview

The project contains two sets of benchmarks:

1. **V1 Benchmarks** (`test/benchmarks_test.go`) - For the v1 driver API
2. **V2 Benchmarks** (`v2/tests/benchmarks_test.go`) - For the v2 driver API

Both benchmark suites test the same operations with **100,000 documents**:
- **Bulk Insert**: Creating large numbers of documents
- **Bulk Read**: Reading documents via queries
- **Bulk Update**: Updating large numbers of documents
- **Bulk Delete**: Deleting large numbers of documents

---

## Benchmark Structure

### V1 Benchmarks (`test/benchmarks_test.go`)

```
test/
├── benchmarks_test.go          # V1 benchmark implementations
└── client_test.go              # Client creation and utilities
```

**Benchmark Functions:**
- `BenchmarkV1BulkInsert100KDocs` - Insert 100K documents
- `BenchmarkV1BulkRead100KDocs` - Read 100K documents  
- `BenchmarkV1bulkUpdate100KDocs` - Update 100K documents
- `BenchmarkV1BulkDelete100KDocs` - Delete 100K documents

### V2 Benchmarks (`v2/tests/benchmarks_test.go`)

```
v2/tests/
└── benchmarks_test.go          # V2 benchmark implementations
```

**Benchmark Functions:**
- `BenchmarkV2BulkInsert100KDocs` - Insert 100K documents
- `BenchmarkV2BulkRead100KDocs` - Read 100K documents
- `BenchmarkV2bulkUpdate100KDocs` - Update 100K documents  
- `BenchmarkV2BulkDelete100KDocs` - Delete 100K documents

---

## Prerequisites

### Local Testing
- Docker installed and running
- Make installed
- At least 8GB RAM available
- Port 7001 available (or configure different port)

### Remote Testing
- Access to a remote ArangoDB instance (HTTP or HTTPS)
- Valid authentication credentials
- Network connectivity to the remote instance

---

## Running Local Benchmarks

Local benchmarks automatically spin up a temporary ArangoDB cluster in Docker.

### V1 Local Benchmarks

```bash
# Set benchmark options (optional)
export TESTOPTIONS="-bench=. -benchmem -run=^$"

# Run all V1 benchmarks against local cluster
make run-benchmarks-cluster-json-no-auth
```

### V2 Local Benchmarks

```bash
# Set benchmark options (optional)
export TESTOPTIONS="-bench=. -benchmem -run=^$"

# Run all V2 benchmarks against local cluster
make run-benchmarks-v2-cluster-json-no-auth
```

### Run Specific Benchmarks

```bash
# Run only V1 insert benchmarks
export TESTOPTIONS="-bench=Insert -benchmem -run=^$"
make run-benchmarks-cluster-json-no-auth

# Run only V2 read benchmarks
export TESTOPTIONS="-bench=Read -benchmem -run=^$"
make run-benchmarks-v2-cluster-json-no-auth
```

---

## Running Remote Benchmarks

Remote benchmarks connect to an existing ArangoDB instance (useful for cloud deployments, Kubernetes clusters, etc.).

### Step 1: Set Environment Variables

```bash
# Set the remote endpoint (HTTPS recommended for production)
export TEST_ENDPOINTS_OVERRIDE="https://your-arango-host:8529"
# Set authentication credentials (format: type:username:password)
export TEST_AUTHENTICATION="basic:root:your_password_here"

# Set benchmark options
export TESTOPTIONS="-bench=. -benchmem -run=^$"
```

**Authentication Format:**
- Basic Auth: `basic:username:password`

### Step 2: Run Remote Benchmarks

**V1 Remote Benchmarks:**
```bash
make run-benchmarks-remote-with-auth
```

**V2 Remote Benchmarks:**
```bash
make run-benchmarks-v2-remote-with-auth
```

### Complete Example - Remote Benchmarks

```bash
#!/bin/bash

# Configuration
export TEST_ENDPOINTS_OVERRIDE="https://eabb601f3998.adbdev.cloud:8529"
export TEST_AUTHENTICATION="basic:root:MySecurePassword123"
export TESTOPTIONS="-bench=. -benchmem -run=^$"

# Clean up any existing test containers
docker rm -f go-driver-test go-driver-test-s 2>/dev/null || true

# Run V1 benchmarks
echo "Running V1 Benchmarks..."
make run-benchmarks-remote-with-auth

# Run V2 benchmarks
echo "Running V2 Benchmarks..."
make run-benchmarks-v2-remote-with-auth
```

---

## Understanding Results

### Benchmark Output Format

```
BenchmarkV2BulkInsert100KDocs/Insert-4    1    7359918443 ns/op    16914840 B/op    265 allocs/op
│                                     │    │             │             │               │
│                                     │    │             │             │               └─ Allocations per operation
│                                     │    │             │             └───────────────── Bytes allocated per operation
│                                     │    │             └─────────────────────────────── Nanoseconds per operation
│                                     │    └───────────────────────────────────────────── Number of iterations
│                                     └────────────────────────────────────────────────── CPU cores used
└──────────────────────────────────────────────────────────────────────────────────────── Benchmark name
```

### Key Metrics

- **ns/op** (nanoseconds per operation): Lower is better
- **B/op** (bytes per operation): Memory used per operation
- **allocs/op**: Number of memory allocations per operation
- **Iterations**: How many times the benchmark ran

### Example Output

```
BenchmarkV2BulkInsert100KDocs/Insert-4              1         4322425027 ns/op         16868600 B/op         201 allocs/op
BenchmarkV2BulkRead100KDocs/ReadAllDocsOnce-4       1        21376798806 ns/op        129860728 B/op     1914146 allocs/op
```
---

## Performance Comparison: V1 vs V2

The following table shows real-world benchmark results comparing V1 (HTTP/1.1) and V2 (HTTP/2) drivers with 100K documents:

### Benchmark Results Summary

#### Create Operations - 100K Documents

| Version | Protocol | Operation | Iterations | ns/op | Δ vs V1 (%) | B/op | Δ vs V1 (%) | allocs/op | Δ vs V1 (%) |
|---------|----------|-----------|------------|-------|-------------|------|-------------|-----------|-------------|
| V1 | HTTP/1.1 | Create-4 | 1 | 7,680,955,841 | - | 165,983,096 | - | 3,300,638 | - |
| V2 | HTTP/2 | Create-4 | 1 | 4,322,425,027 | **-43.73%** | 16,868,600 | **-89.84%** | 201 | **-99.99%** |

#### Read Operations - 100K Documents

| Version | Protocol | Operation | Iterations | ns/op | Δ vs V1 (%) | B/op | Δ vs V1 (%) | allocs/op | Δ vs V1 (%) |
|---------|----------|-----------|------------|-------|-------------|------|-------------|-----------|-------------|
| V1 | HTTP/1.1 | Read-4 | 1 | 21,980,673,168 | - | 117,757,824 | - | 2,221,683 | - |
| V2 | HTTP/2 | Read-4 | 1 | 21,376,798,806 | **-2.75%** | 129,860,728 | +10.28% | 1,914,146 | **-13.84%** |

#### Update Operations - 100K Documents

| Version | Protocol | Operation | Iterations | ns/op | Δ vs V1 (%) | B/op | Δ vs V1 (%) | allocs/op | Δ vs V1 (%) |
|---------|----------|-----------|------------|-------|-------------|------|-------------|-----------|-------------|
| V1 | HTTP/1.1 | Update-4 | 1 | 10,888,914,647 | - | 376,446,344 | - | 7,400,544 | - |
| V2 | HTTP/2 | Update-4 | 1 | 6,500,651,813 | **-40.3%** | 23,234,504 | **-93.83%** | 199,920 | **-97.3%** |

#### Delete Operations - 100K Documents

| Version | Protocol | Operation | Iterations | ns/op | Δ vs V1 (%) | B/op | Δ vs V1 (%) | allocs/op | Δ vs V1 (%) |
|---------|----------|-----------|------------|-------|-------------|------|-------------|-----------|-------------|
| V1 | HTTP/1.1 | Delete-4 | 252 | 12,356,330,212 | - | 398,291,240 | - | 7,401,716 | - |
| V2 | HTTP/2 | Delete-4 | 310 | 10,316,694,878 | **-16.51%** | 22,662,968 | **-94.31%** | 327 | **-99.96%** |

#### Performance Improvements (V2 vs V1)

1. **Create Operations:**
   - **43.73% faster** execution time
   - **89.84% less** memory usage
   - **99.99% fewer** allocations
   - **Best improvement across all operations**

2. **Read Operations:**
   - **2.75% faster** execution time
   - 10.28% more memory usage (acceptable tradeoff)
   - **13.84% fewer** allocations
   - **Minimal performance difference, slight edge to V2**

3. **Update Operations:**
   - **40.3% faster** execution time
   - **93.83% less** memory usage
   - **97.3% fewer** allocations
   - **Excellent improvement in all metrics**

4. **Delete Operations:**
   - **16.51% faster** execution time
   - **94.31% less** memory usage
   - **99.96% fewer** allocations
   - **Significant improvement in memory efficiency**

#### Analysis

**V2 HTTP/2 Advantages:**
- **Multiplexing**: Multiple requests over single connection
- **Header compression**: Reduced overhead per request
- **Binary protocol**: More efficient than text-based HTTP/1.1
- **Better resource management**: Drastically fewer allocations

**Why Read operations show less improvement:**
- Read operations are I/O bound more than CPU/memory bound
- Network latency dominates over protocol overhead
- HTTP/2 benefits are more pronounced in write-heavy operations

**Memory Efficiency:**
- V2 shows dramatic memory improvements across all operations (89-94% reduction)
- Allocation counts reduced by 13-99%, reducing GC pressure
- Better for high-throughput, long-running applications

---

## Benchmark Operations

Each benchmark operation may include sub-benchmarks for different scenarios (e.g., batch operations vs single document operations). The results will show all sub-tests in the output.

### 1. Bulk Insert

Tests document creation performance.

**What it does:**
- Creates a fresh collection
- Inserts N documents in a single batch operation
- Measures time to insert all documents

**V1 Implementation:**
```go
col.CreateDocuments(ctx, docs)
```

**V2 Implementation:**
```go
col.CreateDocumentsWithOptions(ctx, docs, opts)
```

**Use Case:** Measuring initial data loading performance

---

### 2. Bulk Read

Tests document retrieval performance.

**What it does:**
- Runs AQL query to fetch all documents: `FOR d IN collection RETURN d`
- Iterates through cursor to read all documents
- Measures time to read entire collection

**Use Case:** Full collection scans, data exports

---

### 3. Bulk Update

Tests document modification performance.

**What it does:**
- Updates all documents in a single batch operation
- Changes document fields (name, value)
- Measures time to update entire collection

**Use Case:** Bulk data modifications, schema migrations

---

### 4. Bulk Delete

Tests document deletion performance.

**What it does:**
- Recreates all documents
- Deletes all documents in a single batch operation
- Measures time to delete entire collection

**Use Case:** Bulk data cleanup, collection purging

---

## Configuration Options

### Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `TEST_ENDPOINTS_OVERRIDE` | Remote ArangoDB endpoint(s) | - | `https://host:8529` |
| `TEST_AUTHENTICATION` | Authentication credentials | - | `basic:root:password` |
| `TEST_MODE` | Test mode (single/cluster) | `cluster` | `cluster` |
| `TEST_MODE_K8S` | Kubernetes mode flag | - | `k8s` |
| `TEST_NOT_WAIT_UNTIL_READY` | Skip readiness check | - | `1` |
| `TESTOPTIONS` | Go test options | - | `-bench=. -benchmem` |
| `TEST_CONTENT_TYPE` | Content type | `json` |

### TESTOPTIONS Examples

```bash
# Run all benchmarks with memory statistics
export TESTOPTIONS="-bench=. -benchmem -run=^$"

# Run only insert benchmarks
export TESTOPTIONS="-bench=Insert -benchmem -run=^$"

# Run benchmarks for 10 seconds minimum
export TESTOPTIONS="-bench=. -benchtime=10s -run=^$"

# Run each benchmark exactly 10 times
export TESTOPTIONS="-bench=. -benchtime=10x -run=^$"

# Verbose output
export TESTOPTIONS="-bench=. -benchmem -v -run=^$"
```

**Important Flags:**
- `-bench=.` - Run all benchmarks (or specify pattern)
- `-benchmem` - Show memory allocation statistics
- `-run=^$` - Skip regular tests, run only benchmarks
- `-benchtime=Nx` - Run exactly N iterations
- `-benchtime=Ts` - Run for minimum T seconds

---

## Troubleshooting

### Issue 1: "not authorized to execute this request"

**Cause:** Authentication not configured or incorrect credentials

**Solution:**
```bash
# Ensure TEST_AUTHENTICATION is set correctly
export TEST_AUTHENTICATION="basic:root:your_actual_password"

# Verify format: type:username:password
echo $TEST_AUTHENTICATION
```

---

### Issue 2: "duplicate database name 'bench_db'"

**Cause:** Previous benchmark run left databases on remote server

**Solution:**

The benchmarks now automatically handle this by:
- V1: Reuses existing `bench_db_v1` database
- V2: Reuses existing `bench_db_v2` database
- Collections are truncated before each benchmark

To manually clean up:
```bash
# Connect to ArangoDB and delete:
# - bench_db_v1
# - bench_db_v2
```

---

### Issue 3: "dial tcp: lookup deployment-coordinator... no such host"

**Cause:** Driver trying to resolve internal Kubernetes DNS names

**Solution:**

This is already fixed in the remote benchmark targets with:
- `TEST_MODE_K8S="k8s"` - Skips endpoint synchronization
- `TEST_NOT_WAIT_UNTIL_READY="1"` - Skips readiness checks

---

### Issue 4: "http2: unencrypted HTTP/2 not enabled"

**Cause:** Using HTTP (not HTTPS) endpoint with HTTP/2

**Solution:**

Already fixed in V2 benchmarks:
- HTTP endpoints use HTTP/2 cleartext (h2c)
- HTTPS endpoints use standard HTTP/2 with TLS

---

### Issue 5: No benchmark results shown

**Cause:** Running regular tests instead of benchmarks

**Solution:**
```bash
# Always include -run=^$ to skip regular tests
export TESTOPTIONS="-bench=. -benchmem -run=^$"
```

---

## Database and Collection Resources

### Created Resources

**V1 Benchmarks:**
- Database: `bench_db_v1`
- Collection: `bench_col_v1`

**V2 Benchmarks:**
- Database: `bench_db_v2`
- Collection: `bench_col_v2`

### Resource Management

**Local Tests:**
- Resources are automatically cleaned up when Docker containers are destroyed
- Fresh environment on each run

**Remote Tests:**
- Resources persist on the remote server
- Resources are reused across multiple runs
- Collections are truncated before each benchmark
- Safe to run multiple times

### Manual Cleanup

If you need to manually clean up test resources:

```javascript
// Connect to ArangoDB Web UI or arangosh

// Delete V1 resources
db._dropDatabase("bench_db_v1");

// Delete V2 resources
db._dropDatabase("bench_db_v2");
```
---

## Makefile Targets Reference

### V1 Benchmarks

| Target | Description |
|--------|-------------|
| `run-benchmarks-cluster-json-no-auth` | Local cluster, no auth |
| `run-benchmarks-remote-with-auth` | Remote cluster, with auth |

### V2 Benchmarks

| Target | Description |
|--------|-------------|
| `run-benchmarks-v2-cluster-json-no-auth` | Local cluster, no auth |
| `run-benchmarks-v2-remote-with-auth` | Remote cluster, with auth |

---

## Contributing

When adding new benchmarks:

1. Follow existing naming conventions
2. Add documentation to this file
3. Ensure benchmarks are deterministic
4. Include both V1 and V2 implementations (if applicable)
5. Test with both local and remote configurations
6. Ensure proper resource cleanup

---