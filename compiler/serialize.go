package compiler

import (
	"bytes"
	"encoding/binary"

	"github.com/perlin-network/life/compiler/opcodes"
)

// Serialize serializes a set of SSA-form instructions into a byte array
// for execution with an exec.VirtualMachine.
//
// Instruction encoding:
// Value ID (4 bytes) | Opcode (1 byte) | Operands
//
// Types are erased in the generated code.
// Example: float32/float64 are represented as uint32/uint64 respectively.
func (c *SSAFunctionCompiler) Serialize() []byte {
	buf := &bytes.Buffer{}
	insRelocs := make([]int, len(c.Code))
	reloc32Targets := make([]int, 0)

	for i, ins := range c.Code {
		insRelocs[i] = buf.Len()
		_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Target))

		switch ins.Op {
		case "unreachable":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.Unreachable)
		case "select":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.Select)
			for i := 0; i < 3; i++ {
				_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[i]))
			}
			// Int 32-bit
		case "i32.const", "f32.const":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Const)
			_ = binary.Write(buf, binary.LittleEndian, int32(ins.Immediates[0]))
		case "i32.add":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Add)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.sub":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Sub)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.mul":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Mul)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.div_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32DivS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.div_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32DivU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.rem_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32RemS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.rem_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32RemU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.and":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32And)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.or":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Or)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.xor":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Xor)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.shl":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Shl)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.shr_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32ShrS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.shr_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32ShrU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.rotl":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Rotl)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.rotr":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Rotr)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.clz":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Clz)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i32.ctz":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Ctz)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i32.popcnt":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32PopCnt)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i32.eqz":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32EqZ)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i32.eq":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Eq)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.ne":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Ne)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.lt_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32LtS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.lt_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32LtU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.le_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32LeS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.le_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32LeU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.gt_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32GtS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.gt_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32GtU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.ge_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32GeS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.ge_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32GeU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))

			// Int 64-bit
		case "i64.const", "f64.const":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Const)
			_ = binary.Write(buf, binary.LittleEndian, ins.Immediates[0])
		case "i64.add":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Add)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.sub":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Sub)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.mul":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Mul)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.div_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64DivS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.div_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64DivU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.rem_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64RemS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.rem_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64RemU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.and":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64And)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.or":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Or)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.xor":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Xor)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.shl":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Shl)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.shr_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64ShrS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.shr_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64ShrU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.rotl":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Rotl)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.rotr":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Rotr)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.clz":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Clz)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i64.ctz":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Ctz)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i64.popcnt":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64PopCnt)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i64.eqz":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64EqZ)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i64.eq":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Eq)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.ne":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Ne)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.lt_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64LtS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.lt_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64LtU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.le_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64LeS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.le_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64LeU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.gt_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64GtS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.gt_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64GtU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.ge_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64GeS)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.ge_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64GeU)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))

			// Float 32-bit
		case "f32.add":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Add)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.sub":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Sub)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.mul":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Mul)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.div":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Div)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.sqrt":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Sqrt)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.min":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Min)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.max":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Max)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.ceil":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Ceil)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.floor":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Floor)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.trunc":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Trunc)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.nearest":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Nearest)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.abs":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Abs)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.neg":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Neg)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.copysign":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32CopySign)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.eq":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Eq)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.ne":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Ne)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.lt":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Lt)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.le":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Le)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.gt":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Gt)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.ge":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32Ge)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))

			// Float 64-bit
		case "f64.add":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Add)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.sub":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Sub)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.mul":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Mul)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.div":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Div)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.sqrt":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Sqrt)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.min":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Min)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.max":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Max)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.ceil":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Ceil)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.floor":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Floor)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.trunc":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Trunc)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.nearest":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Nearest)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.abs":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Abs)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.neg":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Neg)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.copysign":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64CopySign)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.eq":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Eq)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.ne":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Ne)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.lt":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Lt)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.le":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Le)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.gt":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Gt)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.ge":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64Ge)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))

		case "i32.wrap/i64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32WrapI64)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i32.trunc_s/f32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32TruncSF32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i32.trunc_s/f64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32TruncSF64)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i32.trunc_u/f32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32TruncUF32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i32.trunc_u/f64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32TruncUF64)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i64.trunc_s/f32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64TruncSF32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i64.trunc_s/f64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64TruncSF64)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i64.trunc_u/f32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64TruncUF32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i64.trunc_u/f64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64TruncUF64)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i64.extend_u/i32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64ExtendUI32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i64.extend_s/i32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64ExtendSI32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "f32.demote/f64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32DemoteF64)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "f64.promote/f32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64PromoteF32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "f32.convert_s/i32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32ConvertSI32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "f32.convert_s/i64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32ConvertSI64)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "f32.convert_u/i32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32ConvertUI32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "f32.convert_u/i64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F32ConvertUI64)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "f64.convert_s/i32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64ConvertSI32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "f64.convert_s/i64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64ConvertSI64)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "f64.convert_u/i32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64ConvertUI32)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "f64.convert_u/i64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.F64ConvertUI64)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "i32.reinterpret/f32", "i64.reinterpret/f64", "f32.reinterpret/i32", "f64.reinterpret/i64":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.Nop)

		case "i32.load", "f32.load":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Load)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i32.load8_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Load8S)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i32.load16_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Load16S)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i64.load8_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Load8S)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i64.load16_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Load16S)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i64.load32_s":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Load32S)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i32.load8_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Load8U)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i32.load16_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Load16U)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i64.load8_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Load8U)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i64.load16_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Load16U)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i64.load32_u":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Load32U)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i64.load", "f64.load":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Load)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address

		case "i32.store", "f32.store":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Store)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))     // Address of value to store
		case "i32.store8":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Store8)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))     // Address of value to store

		case "i32.store16":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I32Store16)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))     // Address of value to store

		case "i64.store8":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Store8)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))     // Address of value to store

		case "i64.store16":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Store16)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))     // Address of value to store

		case "i64.store32":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Store32)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))     // Address of value to store

		case "i64.store", "f64.store":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.I64Store)

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))     // Address of value to store

		case "jmp":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.Jmp)

			reloc32Targets = append(reloc32Targets, buf.Len())
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "jmp_if":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.JmpIf)

			reloc32Targets = append(reloc32Targets, buf.Len())
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "jmp_either":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.JmpEither)

			reloc32Targets = append(reloc32Targets, buf.Len())
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
			reloc32Targets = append(reloc32Targets, buf.Len())
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1]))

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "jmp_table":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.JmpTable)
			_ = binary.Write(buf, binary.LittleEndian, uint32(len(ins.Immediates)-1))

			for _, v := range ins.Immediates {
				reloc32Targets = append(reloc32Targets, buf.Len())
				_ = binary.Write(buf, binary.LittleEndian, uint32(v))
			}

			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "phi":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.Phi)
		case "return":
			if len(ins.Values) != 0 {
				_ = binary.Write(buf, binary.LittleEndian, opcodes.ReturnValue)
				_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			} else {
				_ = binary.Write(buf, binary.LittleEndian, opcodes.ReturnVoid)
			}

		case "get_local":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.GetLocal)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
		case "set_local":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.SetLocal)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "get_global":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.GetGlobal)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
		case "set_global":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.SetGlobal)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "call":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.Call)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(len(ins.Values)))
			for _, v := range ins.Values {
				_ = binary.Write(buf, binary.LittleEndian, uint32(v))
			}

		case "call_indirect":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.CallIndirect)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
			_ = binary.Write(buf, binary.LittleEndian, uint32(len(ins.Values)))
			for _, v := range ins.Values {
				_ = binary.Write(buf, binary.LittleEndian, uint32(v))
			}

		case "memory.size":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.CurrentMemory)

		case "memory.grow":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.GrowMemory)
			_ = binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))

		case "add_gas":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.AddGas)
			_ = binary.Write(buf, binary.LittleEndian, uint64(ins.Immediates[0]))

		case "fp_disabled_error":
			_ = binary.Write(buf, binary.LittleEndian, opcodes.FPDisabledError)

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
