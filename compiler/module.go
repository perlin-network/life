package compiler

import (
	"bytes"
	"fmt"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/go-interpreter/wagon/disasm"
)

type Module struct {
	base *wasm.Module
}

func LoadModule(raw []byte) (*Module, error) {
	reader := bytes.NewReader(raw)

	m, err := wasm.ReadModule(reader, nil)
	if err != nil {
		return nil, err
	}
	return &Module {
		base: m,
	}, nil
}

func (m *Module) Compile() {
	for _, f := range m.base.FunctionIndexSpace {
		d, err := disasm.Disassemble(f, m.base)
		if err != nil {
			panic(err)
		}
		compiler := &SSAFunctionCompiler {
			Module: m.base,
			Source: d,
		}
		compiler.Compile()
		fmt.Println(compiler.Code)
	}
}