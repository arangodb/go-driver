# ArangoDB Go Driver V2

Implementation of Driver V2 makes use of runtime JSON serialization, reducing memory and CPU Driver usage.
The Combination of JSON serialization and HTTP2 support makes the driver more efficient and faster.

## Deprecation Notice

Since ArangoDB 3.12 VST support has been dropped and VPack is not anymore developed and maintained. 
The driver will not support VST from version V2 and VPack support will be removed in the future.

V1 driver is deprecated and will not receive any new features. Please use V2 instead.
In V2 we have introduced a new way of handling requests and responses, which is more efficient and easier to use.
