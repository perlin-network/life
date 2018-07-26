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

	vm, err := exec.NewVirtualMachine(input, &Resolver{})
	if err != nil {
		panic(err)
	}

	entryID, ok := vm.GetFunctionExport(*entryFunctionFlag)
	if !ok {
		fmt.Printf("Entry function %s not found; starting from 0.\n", *entryFunctionFlag)
		entryID = 0
	}

	ret, err := vm.Run(entryID)
	if err != nil {
		panic(err)
	}
	fmt.Println(ret)
}
