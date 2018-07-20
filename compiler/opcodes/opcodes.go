package opcodes

type Opcode byte

const (
	Nop Opcode = iota
	Unreachable
	I32Const
	I32Add
	I32Eq
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
