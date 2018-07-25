package main

import (
	"flag"
	"os"
	"fmt"
	"io/ioutil"
	"github.com/perlin-network/life/exec"
)

func main() {
	entryFunctionFlag := flag.String("entry", "app_main", "entry function id")
	flag.Parse()

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	vm := exec.NewVirtualMachine(input, func(module, field string) exec.FunctionImport {
		fmt.Printf("Resolve: %s %s\n", module, field)
		if module == "env" && field == "__life_ping" {
			return func(vm *exec.VirtualMachine) int64 {
				return vm.GetCurrentFrame().Locals[0] + 1
			}
		}
		panic("unknown import")
	})
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
