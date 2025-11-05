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

- **Protocol**: v2 supports both HTTP/1.1 and HTTP/2, with HTTP/2 providing multiplexing, header compression, and binary framing for higher efficiency.
- **Performance**: v2 shows major gains in write-heavy workloads; reads improve less since they're I/O-bound and limited by network latency.
- **Memory**: v2 uses 89–94% less memory and cuts allocations by 13–99%, greatly reducing GC overhead.
- **Overall**: v2 is faster, more memory-efficient, and better suited for high-throughput, long-running applications.