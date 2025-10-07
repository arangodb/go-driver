# 🚀 Benchmark Quick Reference

## 📋 **Essential Commands**

### **Quick Start**
```bash
make benchmark-compare    # Compare V1 vs V2 performance
make benchmark-all        # Run all benchmarks
```

### **Individual Benchmarks**
```bash
# V1 Benchmarks
make run-benchmarks-single-json-no-auth     # JSON, no auth
make run-benchmarks-single-vpack-no-auth    # Velocypack, no auth

# V2 Benchmarks  
make run-v2-benchmarks-single-no-auth       # No authentication
make run-v2-benchmarks-single-with-auth     # With authentication
```

### **Shortcuts**
```bash
make benchmark           # Quick V2 benchmark
make benchmark-v2        # All V2 benchmarks
make benchmark-all       # V1 + V2 comparison
```

## 📊 **Understanding Results**

### **Key Metrics**
- **ns/op** = Nanoseconds per operation (lower = faster)
- **B/op** = Bytes allocated per operation (lower = less memory)
- **allocs/op** = Memory allocations per operation (lower = better)

### **Example Output**
```
BenchmarkConnectionInitialization     534    1963648 ns/op    1234 B/op    5 allocs/op
BenchmarkCreateCollection             193    5377251 ns/op    5678 B/op    12 allocs/op
```

## 🔧 **Prerequisites**
- Docker running
- Make installed
- Go 1.21+

## 📁 **Benchmark Files**
- `test/benchmark_collection_test.go` - V1 collection/query benchmarks
- `test/benchmark_document_test.go` - V1 document benchmarks  
- `v2/tests/benchmark_v2_test.go` - V2 comprehensive benchmarks

## 🎯 **Benchmark Categories**
1. **Connection Initialization** - Client setup
2. **Collection Operations** - Create, exists, list
3. **Document Operations** - CRUD operations
4. **Query Operations** - AQL queries
5. **Cursor Operations** - Result iteration

## 🚨 **Troubleshooting**
- **Docker not running**: Start Docker service
- **Port conflicts**: Stop other ArangoDB instances
- **Timeouts**: Check system resources

## 💡 **Pro Tips**
- Run multiple times for consistency
- Close other applications during benchmarks
- Use same environment for fair comparison
