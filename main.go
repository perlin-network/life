package main

import (
	"flag"
	"fmt"
	"github.com/perlin-network/life/exec"
	"io/ioutil"
	"os"
	"time"
	"math"
	"strings"
)

type Resolver struct {
	tempRet0 int64
}

func (r *Resolver) ResolveFunc(module, field string) exec.FunctionImport {
	fmt.Printf("Resolve func: %s %s\n", module, field)
	if module != "env" {
		panic("module != env")
	}
	
	switch field {
	case "__life_ping":
		return func(vm *exec.VirtualMachine) int64 {
			return vm.GetCurrentFrame().Locals[0] + 1
		}
	case "enlargeMemory":
		return func(vm *exec.VirtualMachine) int64 {
			panic("enlargeMemory not implemented")
		}

	case "getTotalMemory":
		return func(vm *exec.VirtualMachine) int64 {
			return int64(len(vm.Memory))
		}

	case "abortOnCannotGrowMemory":
		return func(vm *exec.VirtualMachine) int64 {
			panic("Cannot grow memory")
		}

	case "abortStackOverflow":
		return func(vm *exec.VirtualMachine) int64 {
			panic("Emscripten stack overflow")
		}

	case "___lock", "___unlock":
		return func(vm *exec.VirtualMachine) int64 {
			return 0
		}

	case "___setErrNo":
		return func(vm *exec.VirtualMachine) int64 {
			panic("setErrNo not implemented")
		}

	case "_emscripten_memcpy_big":
		return func(vm *exec.VirtualMachine) int64 {
			frame := vm.GetCurrentFrame()
			dest := int(frame.Locals[0])
			src := int(frame.Locals[1])
			num := int(frame.Locals[2])
			copy(vm.Memory[dest:], vm.Memory[src:src + num])
			return int64(dest)
		}

	case "abort":
		return func(vm *exec.VirtualMachine) int64 {
			panic("Emscripten abort")
		}

	case "setTempRet0":
		return func(vm *exec.VirtualMachine) int64 {
			r.tempRet0 = vm.GetCurrentFrame().Locals[0]
			return 0
		}

	case "getTempRet0":
		return func(vm *exec.VirtualMachine) int64 {
			return r.tempRet0
		}
	
	default:
		if strings.HasPrefix(field, "nullFunc_") {
			return func(vm *exec.VirtualMachine) int64 {
				panic("nullFunc called")
			}
		}
		if strings.HasPrefix(field, "___syscall") {
			return func(vm *exec.VirtualMachine) int64 {
				panic(fmt.Errorf("syscall %s not supported", field))
			}
		}
		if strings.HasPrefix(field, "jsCall_") {
			return func(vm *exec.VirtualMachine) int64 {
				panic(fmt.Errorf("jsCall %s not supported", field))
			}
		}
		panic(fmt.Errorf("unknown field: %s", field))
	}
}

func (r *Resolver) ResolveGlobal(module, field string) int64 {
	fmt.Printf("Resolve global: %s %s\n", module, field)
	switch module {
	case "env":
		switch field {
		case "__life_magic":
			return 424
		case "memoryBase":
			return 0
		case "tableBase":
			return 0
		case "DYNAMICTOP_PTR":
			return 16
		case "tempDoublePtr":
			return 64
		case "STACK_BASE":
			return 4096
		case "STACKTOP":
			return 4096
		case "STACK_MAX":
			return 4096 + 1048576
		case "ABORT":
			return 0
		case "gb":
			return 1024
		case "fb":
			return 0
		default:
			panic(fmt.Errorf("unknown field: %s", field))
		}
	case "global":
		switch field {
		case "NaN":
			return int64(math.Float64bits(math.NaN()))
		case "Infinity":
			return int64(math.Float64bits(math.Inf(1)))
		default:
			panic(fmt.Errorf("unknown field: %s", field))
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
}

func main() {
	entryFunctionFlag := flag.String("entry", "app_main", "entry function id")
	flag.Parse()

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	vm, err := exec.NewVirtualMachine(input, exec.VMConfig{
		DefaultMemoryPages: 128,
		DefaultTableSize: 65536,
	}, &Resolver{})
	if err != nil {
		panic(err)
	}

	entryID, ok := vm.GetFunctionExport(*entryFunctionFlag)
	if !ok {
		fmt.Printf("Entry function %s not found; starting from 0.\n", *entryFunctionFlag)
		entryID = 0
	}

	start := time.Now()

	if vm.Module.Base.Start != nil {
		startID := int(vm.Module.Base.Start.Index)
		_, err := vm.Run(startID)
		if err != nil {
			vm.PrintStackTrace()
			panic(err)
		}
	}

	ret, err := vm.Run(entryID)
	if err != nil {
		vm.PrintStackTrace()
		panic(err)
	}
	end := time.Now()
	fmt.Printf("return value = %d, duration = %v\n", ret, end.Sub(start))
}
