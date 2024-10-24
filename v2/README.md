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
