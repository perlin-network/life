package main

import (
	"os"
	"io/ioutil"
	"github.com/perlin-network/life/compiler"
)

func main() {
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	m, err := compiler.LoadModule(input)
	if err != nil {
		panic(err)
	}
	m.Compile()
}