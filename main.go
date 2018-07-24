package main

import (
	"os"
	"fmt"
	"io/ioutil"
	"github.com/perlin-network/life/exec"
)

func main() {
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	vm := exec.NewVirtualMachine(input)
	vm.Ignite(0)
	for !vm.Exited {
		vm.Execute()
	}
	if vm.ExitError != nil {
		panic(vm.ExitError)
	}
	fmt.Println(vm.ReturnValue)
}
