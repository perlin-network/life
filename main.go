package main

import (
	"flag"
	"fmt"
	"github.com/perlin-network/life/exec"
	"io/ioutil"
	"os"
)

type Resolver struct{}

func (r *Resolver) ResolveFunc(module, field string) exec.FunctionImport {
	fmt.Printf("Resolve func: %s %s\n", module, field)
	if module == "env" && field == "__life_ping" {
		return func(vm *exec.VirtualMachine) int64 {
			return vm.GetCurrentFrame().Locals[0] + 1
		}
	}
	panic("unknown func import")
}

func (r *Resolver) ResolveGlobal(module, field string) int64 {
	fmt.Printf("Resolve global: %s %s\n", module, field)
	if module == "env" && field == "__life_magic" {
		return 424
	}
	panic("unknown global import")
}

func main() {
	entryFunctionFlag := flag.String("entry", "app_main", "entry function id")
	flag.Parse()

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	vm := exec.NewVirtualMachine(input, &Resolver{})
	if entryID, ok := vm.GetFunctionExport(*entryFunctionFlag); ok {
		vm.Ignite(entryID)
	} else {
		fmt.Printf("Entry function %s not found; starting from 0.\n", *entryFunctionFlag)
		vm.Ignite(0)
	}

	for !vm.Exited {
		vm.Execute()
		if vm.Delegate != nil {
			vm.Delegate()
			vm.Delegate = nil
		}
	}
	if vm.ExitError != nil {
		panic(vm.ExitError)
	}
	fmt.Println(vm.ReturnValue)
}
