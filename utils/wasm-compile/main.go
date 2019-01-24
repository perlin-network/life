package main

import (
	"flag"
	"fmt"
	"github.com/perlin-network/life/exec"
	"io/ioutil"
)

func main() {
	noFloatingPointFlag := flag.Bool("no-fp", false, "disable floating point")
	noMemBoundCheck := flag.Bool("no-mem-bound-check", false, "disable memory bound check")
	flag.Parse()

	// Read WebAssembly *.wasm file.
	input, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		panic(err)
	}

	vm, err := exec.NewVirtualMachine(input, exec.VMConfig{
		DefaultMemoryPages:   128,
		DefaultTableSize:     65536,
		DisableFloatingPoint: *noFloatingPointFlag,
	}, new(exec.NopResolver), nil)

	if err != nil {
		panic(err)
	}

	code := vm.NCompile(exec.NCompileConfig{
		AliasDef:             true,
		DisableMemBoundCheck: *noMemBoundCheck,
	})
	fmt.Println(code)
}
