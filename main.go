package main

import (
	"flag"
	"fmt"
	"github.com/perlin-network/life/exec"
	"github.com/perlin-network/life/platform"
	"github.com/perlin-network/life/wasm-validation"
	"io/ioutil"
	"strconv"
	"time"
)

// Resolver defines imports for WebAssembly modules ran in Life.
type Resolver struct {
	tempRet0 int64
}

// ResolveFunc defines a set of import functions that may be called within a WebAssembly module.
func (r *Resolver) ResolveFunc(module, field string) exec.FunctionImport {
	fmt.Printf("Resolve func: %s %s\n", module, field)
	switch module {
	case "env":
		switch field {
		case "__life_ping":
			return func(vm *exec.VirtualMachine) int64 {
				return vm.GetCurrentFrame().Locals[0] + 1
			}
		case "__life_log":
			return func(vm *exec.VirtualMachine) int64 {
				ptr := int(uint32(vm.GetCurrentFrame().Locals[0]))
				msgLen := int(uint32(vm.GetCurrentFrame().Locals[1]))
				msg := vm.Memory[ptr : ptr+msgLen]
				fmt.Printf("[app] %s\n", string(msg))
				return 0
			}
		case "print_i64":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("[app] print_i64: %d\n", vm.GetCurrentFrame().Locals[0])
				return 0
			}

		default:
			panic(fmt.Errorf("unknown field: %s", field))
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
}

// ResolveGlobal defines a set of global variables for use within a WebAssembly module.
func (r *Resolver) ResolveGlobal(module, field string) int64 {
	fmt.Printf("Resolve global: %s %s\n", module, field)
	switch module {
	case "env":
		switch field {
		case "__life_magic":
			return 424
		default:
			panic(fmt.Errorf("unknown field: %s", field))
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
}

func main() {
	entryFunctionFlag := flag.String("entry", "app_main", "entry function name")
	pmFlag := flag.Bool("polymerase", false, "enable the Polymerase engine")
	noFloatingPointFlag := flag.Bool("no-fp", false, "disable floating point")
	flag.Parse()

	// Read WebAssembly *.wasm file.
	input, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		panic(err)
	}

	validator, err := wasm_validation.NewValidator()
	if err != nil {
		panic(err)
	}

	err = validator.ValidateWasm(input)
	if err != nil {
		panic(err)
	}

	// Instantiate a new WebAssembly VM with a few resolved imports.
	vm, err := exec.NewVirtualMachine(input, exec.VMConfig{
		DefaultMemoryPages:   128,
		DefaultTableSize:     65536,
		DisableFloatingPoint: *noFloatingPointFlag,
	}, new(Resolver), nil)

	if err != nil {
		panic(err)
	}

	if *pmFlag {
		compileStartTime := time.Now()
		fmt.Println("[Polymerase] Compilation started.")
		aotSvc := platform.FullAOTCompile(vm)
		if aotSvc != nil {
			compileEndTime := time.Now()
			fmt.Printf("[Polymerase] Compilation finished successfully in %+v.\n", compileEndTime.Sub(compileStartTime))
			vm.SetAOTService(aotSvc)
		} else {
			fmt.Println("[Polymerase] The current platform is not yet supported.")
		}
	}

	// Get the function ID of the entry function to be executed.
	entryID, ok := vm.GetFunctionExport(*entryFunctionFlag)
	if !ok {
		fmt.Printf("Entry function %s not found; starting from 0.\n", *entryFunctionFlag)
		entryID = 0
	}

	start := time.Now()

	// If any function prior to the entry function was declared to be
	// called by the module, run it first.
	if vm.Module.Base.Start != nil {
		startID := int(vm.Module.Base.Start.Index)
		_, err := vm.Run(startID)
		if err != nil {
			vm.PrintStackTrace()
			panic(err)
		}
	}
	var args []int64
	for _, arg := range flag.Args()[1:] {
		fmt.Println(arg)
		if ia, err := strconv.Atoi(arg); err != nil {
			panic(err)
		} else {
			args = append(args, int64(ia))
		}
	}

	// Run the WebAssembly module's entry function.
	ret, err := vm.Run(entryID, args...)
	if err != nil {
		vm.PrintStackTrace()
		panic(err)
	}
	end := time.Now()

	fmt.Printf("return value = %d, duration = %v\n", ret, end.Sub(start))
}
