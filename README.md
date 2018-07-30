# Life

**life** is a secure & fast WebAssembly VM built for decentralized applications, written in [Go](https://golang.org/) by Perlin Network.

## Features

- Correct: Fully implements the WebAssembly execution semantics and passes most of the [official test suite](https://github.com/WebAssembly/testsuite) (66/72 passed, none of the failures are related to the execution semantics).
- Fast: **life** uses a range of optimization techniques and is faster than all other WebAssembly implementations tested ([go-interpreter/wagon](https://github.com/go-interpreter/wagon), [paritytech/wasmi](https://github.com/paritytech/wasmi)). Benchmark results [here](https://gist.github.com/losfair/1d3743433fafd8d0a1d1dac3c0db4827). JIT support for x86-64 and ARM is planned.
- Secure: User code is fully sandboxed. Accurate control to resources (instruction cycles, memory usage) is allowed.

## Get started

```bash
# install vgo tooling
go get -u golang.org/x/vgo

# download the dependencies to vendor folder
vgo mod -vendor

# build test suite runner
vgo build github.com/perlin-network/life/spec/test_runner

# run official test suite
python3 run_spec_tests.py /path/to/testsuite

# build main program
vgo build

# run your wasm program
./life < /path/to/your/wasm/program # entry point is `app_main` with no arguments by default
```

## Integrating into your application

Suppose we have already read in the wasm bytecode to `input`.

Set up the virtual machine:
```go
vm, err := exec.NewVirtualMachine(input, exec.VMConfig{}, &exec.NopResolver{})
if err != nil { // if the wasm bytecode is invalid
    panic(err)
}
```

Lookup the entry function:
```go
entryID, ok := vm.GetFunctionExport("app_main") // can change to whatever exported function name you want
if !ok {
    panic("entry function not found")
}
```

Run the VM:
```go
ret, err := vm.Run(entryID)
if err != nil {
    vm.PrintStackTrace()
    panic(err)
}
fmt.Printf("return value = %d\n", ret)
```

## Benchmark

We benchmarked **life** and a few other WebAssembly implementation in different languages.

### Pollard Rho Factoring (128-bit)

Test case: `pollard_rho_128`

Life (JIT enabled):

```
return value = 2635722126511989555, duration = 1.423329989s

real	0m1.672s
user	0m1.549s
sys	0m0.234s
```

Life (Interpreter only):

```
return value = 2635722126511989555, duration = 1.944211748s

real	0m2.178s
user	0m2.263s
sys	0m0.012s
```

wagon:

```
app_main() i64 => 2635722126511989555 (uint64)

real	0m4.349s
user	0m4.462s
sys	0m0.027s
```

wasmi:

```
Result: Ok(Some(I64(2635722126511989555)))

real	0m4.852s
user	0m4.837s
sys	0m0.006s
```

### Snappy Compress

Test case: `snappy_compress`

Life (JIT enabled):

```
return value = 393476, duration = 811.148389ms

real	0m1.072s
user	0m0.973s
sys	0m0.178s
```

Life (Interpreter only):

```
return value = 393476, duration = 2.710492104s

real	0m2.977s
user	0m3.006s
sys	0m0.031s
```

wagon:

```
app_main() i32 => 393476 (uint32)

real	0m5.443s
user	0m5.448s
sys	0m0.043s
```

wasmi:

```
Result: Ok(Some(I32(393476)))

real	0m7.017s
user	0m6.983s
sys	0m0.020s
```

### Fibonacci generator (recursive)

Test case: `fib_recursive`

Life (JIT enabled):

```
return value = 9227465, duration = 3.649676997s

real	0m3.898s
user	0m4.041s
sys	0m0.057s
```

Life (Interpreter only):

```
return value = 9227465, duration = 2.831716826s

real	0m3.071s
user	0m3.199s
sys	0m0.030s
```

wagon:

```
app_main() i32 => 9227465 (uint32)

real	0m6.204s
user	0m6.361s
sys	0m0.039s
```

wasmi:

```
Result: Ok(Some(I32(9227465)))

real	0m5.811s
user	0m5.789s
sys	0m0.013s
```

## Contributions

We at Perlin love reaching out to the open-source community and are open to accepting issues and pull-requests.

For all code contributions, please ensure they adhere as close as possible to the following guidelines:

1. **Strictly** follows the formatting and styling rules denoted [here](https://github.com/golang/go/wiki/CodeReviewComments).
2. Commit messages are in the format `module_name: Change typed down as a sentence.` This allows our maintainers and everyone else to know what specific code changes you wish to address.
    - `compiler/liveness: Implemented full liveness analysis.`
    - `exec/helpers: Added function to run the VM with time limit.`
3. Consider backwards compatibility. New methods are perfectly fine, though changing the existing public API should only be done should there be a good reason.

If you...

1. love the work we are doing,
2. want to work full-time with us,
3. or are interested in getting paid for working on open-source projects

... **we're hiring**.

To grab our attention, just make a PR and start contributing.

