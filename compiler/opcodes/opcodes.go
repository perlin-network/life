package opcodes

type Opcode byte

const (
	Nop Opcode = iota
	Unreachable
	I32Const
	I32Add
	I32Eq
	I64Const
	I64Add
	I64Eq
	F32Const
	F32Add
	F32Eq
	F64Const
	F64Add
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
