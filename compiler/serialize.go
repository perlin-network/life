package compiler

import (
	"bytes"
	"encoding/binary"

	"github.com/perlin-network/life/compiler/opcodes"
)

// Instruction encoding:
// Value ID (4 bytes) | Opcode (1 byte) | Operands
func (c *SSAFunctionCompiler) Serialize() []byte {
	buf := &bytes.Buffer{}
	insRelocs := make([]int, len(c.Code))
	reloc32Targets := make([]int, 0)

	for i, ins := range c.Code {
		insRelocs[i] = buf.Len()
		binary.Write(buf, binary.LittleEndian, uint32(ins.Target))

		switch ins.Op {
		case "unreachable":
			binary.Write(buf, binary.LittleEndian, opcodes.Unreachable)

		// Int 32-bit
		case "i32.const":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Const)
			binary.Write(buf, binary.LittleEndian, int32(ins.Immediates[0]))
		case "i32.add":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Add)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.sub":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Sub)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.mul":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Mul)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.div_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I32DivS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.div_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I32DivU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.rem_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I32RemS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.rem_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I32RemU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.eq":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Eq)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))

		// Int 64-bit
		case "i64.const":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Const)
			binary.Write(buf, binary.LittleEndian, int64(ins.Immediates[0]))
		case "i64.add":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Add)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.sub":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Sub)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.mul":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Mul)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.div_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I64DivS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.div_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I64DivU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.rem_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I64RemS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.rem_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I64RemU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.eq":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Eq)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))

		// Float 32-bit
		case "f32.const":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Const)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
		case "f32.add":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Add)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.sub":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Sub)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.mul":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Mul)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.div":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Div)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.eq":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Eq)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))

		// Float 64-bit
		case "f64.const":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Const)
			binary.Write(buf, binary.LittleEndian, uint64(ins.Immediates[0]))
		case "f64.add":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Add)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.sub":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Sub)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.mul":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Mul)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.div":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Div)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.eq":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Eq)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))

		case "jmp":
			binary.Write(buf, binary.LittleEndian, opcodes.Jmp)

			reloc32Targets = append(reloc32Targets, buf.Len())
			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "jmp_if":
			binary.Write(buf, binary.LittleEndian, opcodes.JmpIf)

			reloc32Targets = append(reloc32Targets, buf.Len())
			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))

			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "jmp_table":
			binary.Write(buf, binary.LittleEndian, opcodes.JmpTable)
			binary.Write(buf, binary.LittleEndian, uint32(len(ins.Immediates)-1))

			for _, v := range ins.Immediates {
				reloc32Targets = append(reloc32Targets, buf.Len())
				binary.Write(buf, binary.LittleEndian, uint32(v))
			}

			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "phi":
			binary.Write(buf, binary.LittleEndian, opcodes.Phi)
		case "return":
			if len(ins.Values) != 0 {
				binary.Write(buf, binary.LittleEndian, opcodes.ReturnValue)
				binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			} else {
				binary.Write(buf, binary.LittleEndian, opcodes.ReturnVoid)
			}
		case "get_local":
			binary.Write(buf, binary.LittleEndian, opcodes.GetLocal)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))

		case "set_local":
			binary.Write(buf, binary.LittleEndian, opcodes.SetLocal)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "call":
			binary.Write(buf, binary.LittleEndian, opcodes.Call)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
			binary.Write(buf, binary.LittleEndian, uint32(len(ins.Values)))
			for _, v := range ins.Values {
				binary.Write(buf, binary.LittleEndian, uint32(v))
			}

		case "call_indirect":
			binary.Write(buf, binary.LittleEndian, opcodes.CallIndirect)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
			binary.Write(buf, binary.LittleEndian, uint32(len(ins.Values)))
			for _, v := range ins.Values {
				binary.Write(buf, binary.LittleEndian, uint32(v))
			}

		default:
			panic(ins.Op)
		}
	}

	ret := buf.Bytes()

	for _, t := range reloc32Targets {
		insPos := binary.LittleEndian.Uint32(ret[t : t+4])
		binary.LittleEndian.PutUint32(ret[t:t+4], uint32(insRelocs[insPos]))
	}

	return ret
}
