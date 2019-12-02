package compiler

import (
	"bytes"
	"encoding/binary"
	"strings"

	"github.com/go-interpreter/wagon/disasm"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/go-interpreter/wagon/wasm/leb128"

	"github.com/perlin-network/life/compiler/opcodes"
	"github.com/perlin-network/life/utils"
)

type Module struct {
	Base                 *wasm.Module
	FunctionNames        map[int]string
	DisableFloatingPoint bool
}

type InterpreterCode struct {
	NumRegs    int
	NumParams  int
	NumLocals  int
	NumReturns int
	Bytes      []byte
	JITInfo    interface{}
	JITDone    bool
}

func LoadModule(raw []byte) (*Module, error) {
	reader := bytes.NewReader(raw)

	m, err := wasm.ReadModule(reader, nil)
	if err != nil {
		return nil, err
	}

	/*err = validate.VerifyModule(m)
	if err != nil {
		return nil, err
	}*/

	functionNames := make(map[int]string)

	for _, sec := range m.Customs {
		if sec.Name == "name" {
			r := bytes.NewReader(sec.RawSection.Bytes)

			for {
				ty, err := leb128.ReadVarUint32(r)
				if err != nil || ty != 1 {
					break
				}

				payloadLen, err := leb128.ReadVarUint32(r)
				if err != nil {
					panic(err)
				}

				data := make([]byte, int(payloadLen))

				n, err := r.Read(data)
				if err != nil {
					panic(err)
				}

				if n != len(data) {
					panic("len mismatch")
				}

				{
					r := bytes.NewReader(data)
					for {
						count, err := leb128.ReadVarUint32(r)
						if err != nil {
							break
						}

						for i := 0; i < int(count); i++ {
							index, err := leb128.ReadVarUint32(r)
							if err != nil {
								panic(err)
							}

							nameLen, err := leb128.ReadVarUint32(r)
							if err != nil {
								panic(err)
							}

							name := make([]byte, int(nameLen))

							n, err := r.Read(name)
							if err != nil {
								panic(err)
							}

							if n != len(name) {
								panic("len mismatch")
							}

							functionNames[int(index)] = string(name)
						}
					}
				}
			}
		}
	}

	return &Module{
		Base:          m,
		FunctionNames: functionNames,
	}, nil
}

func (m *Module) CompileWithNGen(gp GasPolicy, numGlobals uint64) (string, error) {
	var (
		out    string
		retErr error
	)

	defer utils.CatchPanic(&retErr)

	importStubBuilder := &strings.Builder{}
	importTypeIDs := make([]int, 0)
	numFuncImports := 0

	if m.Base.Import != nil {
		for i := 0; i < len(m.Base.Import.Entries); i++ {
			e := &m.Base.Import.Entries[i]
			if e.Type.Kind() != wasm.ExternalFunction {
				continue
			}

			tyID := e.Type.(wasm.FuncImport).Type
			ty := &m.Base.Types.Entries[int(tyID)]

			bSprintf(importStubBuilder, "uint64_t %s%d(struct VirtualMachine *vm", NGEN_FUNCTION_PREFIX, i)

			for j := 0; j < len(ty.ParamTypes); j++ {
				bSprintf(importStubBuilder, ",uint64_t %s%d", NGEN_LOCAL_PREFIX, j)
			}

			importStubBuilder.WriteString(") {\n")
			importStubBuilder.WriteString("uint64_t params[] = {")

			for j := 0; j < len(ty.ParamTypes); j++ {
				bSprintf(importStubBuilder, "%s%d", NGEN_LOCAL_PREFIX, j)

				if j != len(ty.ParamTypes)-1 {
					importStubBuilder.WriteByte(',')
				}
			}

			importStubBuilder.WriteString("};\n")
			bSprintf(importStubBuilder, "return %sinvoke_import(vm, %d, %d, params);\n", NGEN_ENV_API_PREFIX, numFuncImports, len(ty.ParamTypes))
			importStubBuilder.WriteString("}\n")

			importTypeIDs = append(importTypeIDs, int(tyID))
			numFuncImports++
		}
	}

	out += importStubBuilder.String()

	for i, f := range m.Base.FunctionIndexSpace {
		//fmt.Printf("Compiling function %d (%+v) with %d locals\n", i, f.Sig, len(f.Body.Locals))
		instrs, err := disasm.Disassemble(f.Body.Code)
		if err != nil {
			panic(err)
		}

		d := disasm.Disassembly{
			Code:     instrs,
			MaxDepth: 512,
		}
		compiler := NewSSAFunctionCompiler(m.Base, &d)
		compiler.CallIndexOffset = numFuncImports

		compiler.Compile(importTypeIDs)

		if m.DisableFloatingPoint {
			compiler.FilterFloatingPoint()
		}

		if gp != nil {
			compiler.InsertGasCounters(gp)
		}
		//fmt.Println(compiler.Code)
		//fmt.Printf("%+v\n", compiler.NewCFGraph())
		//numRegs := compiler.RegAlloc()
		//fmt.Println(compiler.Code)
		numLocals := 0

		for _, v := range f.Body.Locals {
			numLocals += int(v.Count)
		}

		out += compiler.NGen(uint64(numFuncImports+i), uint64(len(f.Sig.ParamTypes)), uint64(numLocals), numGlobals)
	}

	return out, retErr
}

func (m *Module) CompileForInterpreter(gp GasPolicy) ([]InterpreterCode, error) {
	var (
		ret    []InterpreterCode
		retErr error
	)

	defer utils.CatchPanic(&retErr)

	importTypeIDs := make([]int, 0)

	if m.Base.Import != nil {
		j := 0

		for i := 0; i < len(m.Base.Import.Entries); i++ {
			e := &m.Base.Import.Entries[i]
			if e.Type.Kind() != wasm.ExternalFunction {
				continue
			}

			tyID := e.Type.(wasm.FuncImport).Type
			ty := &m.Base.Types.Entries[int(tyID)]

			buf := &bytes.Buffer{}

			_ = binary.Write(buf, binary.LittleEndian, uint32(1)) // value ID
			_ = binary.Write(buf, binary.LittleEndian, opcodes.InvokeImport)
			_ = binary.Write(buf, binary.LittleEndian, uint32(j))
			_ = binary.Write(buf, binary.LittleEndian, uint32(0))

			if len(ty.ReturnTypes) != 0 {
				_ = binary.Write(buf, binary.LittleEndian, opcodes.ReturnValue)
				_ = binary.Write(buf, binary.LittleEndian, uint32(1))
			} else {
				_ = binary.Write(buf, binary.LittleEndian, opcodes.ReturnVoid)
			}

			code := buf.Bytes()

			ret = append(ret, InterpreterCode{
				NumRegs:    2,
				NumParams:  len(ty.ParamTypes),
				NumLocals:  0,
				NumReturns: len(ty.ReturnTypes),
				Bytes:      code,
			})

			importTypeIDs = append(importTypeIDs, int(tyID))
			j++
		}
	}

	numFuncImports := len(ret)
	ret = append(ret, make([]InterpreterCode, len(m.Base.FunctionIndexSpace))...)

	for i, f := range m.Base.FunctionIndexSpace {
		//fmt.Printf("Compiling function %d (%+v) with %d locals\n", i, f.Sig, len(f.Body.Locals))
		instrs, err := disasm.Disassemble(f.Body.Code)
		if err != nil {
			panic(err)
		}

		d := disasm.Disassembly{
			Code:     instrs,
			MaxDepth: 512,
		}

		compiler := NewSSAFunctionCompiler(m.Base, &d)
		compiler.CallIndexOffset = numFuncImports
		compiler.Compile(importTypeIDs)

		if m.DisableFloatingPoint {
			compiler.FilterFloatingPoint()
		}

		if gp != nil {
			compiler.InsertGasCounters(gp)
		}
		//fmt.Println(compiler.Code)
		//fmt.Printf("%+v\n", compiler.NewCFGraph())
		numRegs := compiler.RegAlloc()
		//fmt.Println(compiler.Code)
		numLocals := 0

		for _, v := range f.Body.Locals {
			numLocals += int(v.Count)
		}

		ret[numFuncImports+i] = InterpreterCode{
			NumRegs:    numRegs,
			NumParams:  len(f.Sig.ParamTypes),
			NumLocals:  numLocals,
			NumReturns: len(f.Sig.ReturnTypes),
			Bytes:      compiler.Serialize(),
		}
	}

	return ret, retErr
}
