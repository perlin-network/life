package compiler

import (
	"bytes"
	"fmt"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/go-interpreter/wagon/disasm"
)

type Module struct {
	Base *wasm.Module
}

func LoadModule(raw []byte) (*Module, error) {
	reader := bytes.NewReader(raw)

	m, err := wasm.ReadModule(reader, nil)
	if err != nil {
		return nil, err
	}
	return &Module {
		Base: m,
	}, nil
}

func (m *Module) CompileForInterpreter() [][]byte {
	ret := make([][]byte, len(m.Base.FunctionIndexSpace))

	for i, f := range m.Base.FunctionIndexSpace {
		d, err := disasm.Disassemble(f, m.Base)
		if err != nil {
			panic(err)
		}
		compiler := &SSAFunctionCompiler {
			Module: m.Base,
			Source: d,
		}
		compiler.Compile()
		fmt.Println(compiler.Code)
		ret[i] = compiler.Serialize()
	}

	return ret
}
