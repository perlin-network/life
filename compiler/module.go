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

type InterpreterCode struct {
	NumRegs int
	Bytes []byte
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

func (m *Module) CompileForInterpreter() []InterpreterCode {
	ret := make([]InterpreterCode, len(m.Base.FunctionIndexSpace))

	for i, f := range m.Base.FunctionIndexSpace {
		d, err := disasm.Disassemble(f, m.Base)
		if err != nil {
			panic(err)
		}
		compiler := NewSSAFunctionCompiler(m.Base, d)
		compiler.Compile()
		fmt.Println(compiler.Code)
		numRegs := compiler.RegAlloc()
		fmt.Println(compiler.Code)
		ret[i] = InterpreterCode {
			NumRegs: numRegs,
			Bytes: compiler.Serialize(),
		}
	}

	return ret
}
