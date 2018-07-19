package opcodes

type Opcode byte

const (
	Nop Opcode = iota
	Unreachable
	I32Const
	I32Add
	Jmp
	JmpIf
	JmpTable
	ReturnValue
	ReturnVoid
	GetLocal
	SetLocal
	Phi
)
