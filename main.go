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
	ret := vm.Execute(0)
	fmt.Println(ret)
}
