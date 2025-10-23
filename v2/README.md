# ArangoDB Go Driver Version 2

The implementation of the v2 driver makes use of runtime JSON serialization,
reducing memory and CPU usage. The combination of JSON serialization and HTTP/2
support makes the driver more efficient and faster.

To get started, see the
[Tutorial](https://docs.arangodb.com/stable/develop/drivers/go/).

## Deprecation Notice

From ArangoDB v3.12 onward, the VelocyStream (VST) protocol is not supported
any longer. The v2 driver does not support VelocyStream. VelocyPack support in
the driver is not developed and maintained anymore and will removed in a
future version.

The v1 driver is deprecated and will not receive any new features.
Please use v2 instead, which uses a new way of handling requests and responses
that is more efficient and easier to use.

## Benchmarks

V2 driver shows significant performance improvements over V1, with 16-44% faster execution times and 89-94% less memory usage across all operations. 

For detailed benchmark results, analysis, and instructions on running benchmarks,
see [BENCHMARKS.md](./BENCHMARKS.md).

### go-driver v2 vs v1 Summary

- **Protocol**: v2 switches from HTTP/1.1 to HTTP/2, enabling multiplexing, header compression, and binary framing for higher efficiency.
- **Performance**: v2 shows major gains in write-heavy workloads; reads improve less since they’re I/O-bound and limited by network latency.
- **Memory**: v2 uses 89–94% less memory and cuts allocations by 13–99%, greatly reducing GC overhead.
- **Overall**: v2 is faster, more memory-efficient, and better suited for high-throughput, long-running applications.

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

### Running Benchmarks

Quick start to run V2 benchmarks:

```bash
# For local testing
export TESTOPTIONS="-bench=. -benchmem -run=^$"
make run-benchmarks-v2-cluster-json-no-auth

# For remote testing with authentication
export TEST_ENDPOINTS_OVERRIDE="https://your-arango-host:8529"
export TEST_AUTHENTICATION="basic:root:your_password"
export TESTOPTIONS="-bench=. -benchmem -run=^$"
make run-benchmarks-v2-remote-with-auth
```