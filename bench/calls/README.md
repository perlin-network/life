Benchmarking calls to/from VM

# Run bencmarks

- go test -run none -benchmem -bench Benchmark_callSumAndAdd1

# Profile memory

- go test -run none -benchmem -memprofile=mem.out -bench Benchmark_callSumAndAdd1
- go tool pprof -http=:8088 mem.out

# Profile CPU

- go test -run none -benchmem -cpuprofile=cpu.out -bench Benchmark_callSumAndAdd1
- go tool pprof -http=:8088 cpu.out

