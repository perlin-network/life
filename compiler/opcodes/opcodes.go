package opcodes

type Opcode byte

const (
	Nop Opcode = iota
	Unreachable
	I32Const
	I32Add
	I32Sub
	I32Mul
	I32Divs
	I32Div
	I32Eq
	I64Const
	I64Add
	I64Sub
	I64Mul
	I64Divs
	I64Div
	I64Eq
	F32Const
	F32Add
	F32Sub
	F32Mul
	F32Div
	F32Eq
	F64Const
	F64Add
	F64Sub
	F64Mul
	F64Div
	F64Eq
	Jmp
	JmpIf
	JmpTable
	ReturnValue
	ReturnVoid
	GetLocal
	SetLocal
	Call
	CallIndirect
	Phi
)
