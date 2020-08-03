Benchmarking calls to/from VM

# Run bencmarks

- go test -run none -benchmem -bench Benchmark_callSumAndAdd1

# Profile memory

- go test -run none -benchmem -memprofile=mem.out -bench Benchmark_callSumAndAdd1
- go tool pprof -http=:8088 mem.out

# Profile CPU

- go test -run none -benchmem -cpuprofile=cpu.out -bench Benchmark_callSumAndAdd1
- go tool pprof -http=:8088 cpu.out

# Links

- https://github.com/perlin-network/life/blob/69f41b0484c346e56a57921aef51f7aa4947f5b2/exec/vm_codegen.go
- GAS https://github.com/perlin-network/life/issues/86

# Misc

goos: linux
goarch: amd64
pkg: github.com/perlin-network/life/bench/calls
Benchmark_callSumAndAdd1_0_NoAOT-2       2394116               504 ns/op               0 B/op          0 allocs/op
Benchmark_callSumAndAdd1_1_NoAOT-2        974137              1245 ns/op               0 B/op          0 allocs/op
Benchmark_callSumAndAdd1_10_NoAOT-2       142293              8406 ns/op               0 B/op          0 allocs/op
Benchmark_callSumAndAdd1_0_AOT-2         1000000              1192 ns/op              24 B/op          2 allocs/op
Benchmark_callSumAndAdd1_1_AOT-2          606807              2096 ns/op              24 B/op          2 allocs/op
Benchmark_callSumAndAdd1_10_AOT-2         129069              9286 ns/op              24 B/op          2 allocs/op

Benchmark_life_callSumAndAdd1_0_NoAOT-2          1726413               696 ns/op              56 B/op          2 allocs/op
Benchmark_life_callSumAndAdd1_1_NoAOT-2           669055              1834 ns/op             144 B/op          5 allocs/op
Benchmark_life_callSumAndAdd1_10_NoAOT-2           93620             12866 ns/op             936 B/op         32 allocs/op
Benchmark_life_callSumAndAdd1_0_AOT-2             854827              1390 ns/op              80 B/op          4 allocs/op
Benchmark_life_callSumAndAdd1_1_AOT-2             463479              2645 ns/op             168 B/op          7 allocs/op
Benchmark_life_callSumAndAdd1_10_AOT-2             87625             13668 ns/op             960 B/op         34 allocs/op
Benchmark_life_callSumAndAdd1_100_AOT-2             9847            141453 ns/op            8880 B/op        304 allocs/op