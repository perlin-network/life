package opcodes

type Opcode byte

const (
	Nop Opcode = iota
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
