package compiler

import (
	"github.com/go-interpreter/wagon/wasm"
)

type Module struct {
	base *wasm.Module
}

type FunctionBody struct {
	Code []Instr
}
