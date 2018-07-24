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
		case "i32.and":
			binary.Write(buf, binary.LittleEndian, opcodes.I32And)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.or":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Or)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.xor":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Xor)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.shl":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Shl)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.shr_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I32ShrS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.shr_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I32ShrU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.rotl":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Rotl)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.rotr":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Rotr)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.clz":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Clz)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i32.ctz":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Ctz)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i32.popcnt":
			binary.Write(buf, binary.LittleEndian, opcodes.I32PopCnt)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i32.eqz":
			binary.Write(buf, binary.LittleEndian, opcodes.I32EqZ)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i32.eq":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Eq)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.ne":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Ne)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.lt_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I32LtS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.lt_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I32LtU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.le_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I32LeS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.le_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I32LeU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.gt_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I32GtS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.gt_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I32GtU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.ge_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I32GeS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i32.ge_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I32GeU)
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
		case "i64.and":
			binary.Write(buf, binary.LittleEndian, opcodes.I64And)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.or":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Or)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.xor":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Xor)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.shl":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Shl)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.shr_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I64ShrS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.shr_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I64ShrU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.rotl":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Rotl)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.rotr":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Rotr)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.clz":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Clz)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i64.ctz":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Ctz)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i64.popcnt":
			binary.Write(buf, binary.LittleEndian, opcodes.I64PopCnt)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i64.eqz":
			binary.Write(buf, binary.LittleEndian, opcodes.I64EqZ)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "i64.eq":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Eq)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.ne":
			binary.Write(buf, binary.LittleEndian, opcodes.I64Ne)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.lt_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I64LtS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.lt_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I64LtU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.le_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I64LeS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.le_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I64LeU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.gt_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I64GtS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.gt_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I64GtU)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.ge_s":
			binary.Write(buf, binary.LittleEndian, opcodes.I64GeS)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "i64.ge_u":
			binary.Write(buf, binary.LittleEndian, opcodes.I64GeU)
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
		case "f32.sqrt":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Sqrt)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.min":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Min)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.max":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Max)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.ceil":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Ceil)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.floor":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Floor)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.trunc":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Trunc)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.nearest":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Nearest)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.abs":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Abs)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.neg":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Neg)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f32.copysign":
			binary.Write(buf, binary.LittleEndian, opcodes.F32CopySign)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.eq":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Eq)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.ne":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Ne)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.lt":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Lt)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.le":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Le)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.gt":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Gt)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f32.ge":
			binary.Write(buf, binary.LittleEndian, opcodes.F32Ge)
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
		case "f64.sqrt":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Sqrt)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.min":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Min)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.max":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Max)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.ceil":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Ceil)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.floor":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Floor)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.trunc":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Trunc)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.nearest":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Nearest)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.abs":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Abs)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.neg":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Neg)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
		case "f64.copysign":
			binary.Write(buf, binary.LittleEndian, opcodes.F64CopySign)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.eq":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Eq)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.ne":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Ne)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.lt":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Lt)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.le":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Le)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.gt":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Gt)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))
		case "f64.ge":
			binary.Write(buf, binary.LittleEndian, opcodes.F64Ge)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))

		case "i32.load":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Load)

			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address
		case "i32.store":
			binary.Write(buf, binary.LittleEndian, opcodes.I32Store)

			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0])) // Memory alignment flags
			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[1])) // Memory offset
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[0]))     // Memory base address
			binary.Write(buf, binary.LittleEndian, uint32(ins.Values[1]))     // Address of value to store
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

		case "get_global":
			binary.Write(buf, binary.LittleEndian, opcodes.GetGlobal)
			binary.Write(buf, binary.LittleEndian, uint32(ins.Immediates[0]))
		case "set_global":
			binary.Write(buf, binary.LittleEndian, opcodes.SetGlobal)
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
