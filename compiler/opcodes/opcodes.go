package opcodes

// To execute next command please install a `stringer` tool:
// `go get -u golang.org/x/tools/cmd/stringer`

//go:generate stringer -type=Opcode

type Opcode byte

const (
	Nop Opcode = iota
	Unreachable
	Select
	I32Const
	I32Add
	I32Sub
	I32Mul
	I32DivS
	I32DivU
	I32RemS
	I32RemU
	I32And
	I32Or
	I32Xor
	I32Shl
	I32ShrS
	I32ShrU
	I32Rotl
	I32Rotr
	I32Clz
	I32Ctz
	I32PopCnt
	I32EqZ
	I32Eq
	I32Ne
	I32LtS
	I32LtU
	I32LeS
	I32LeU
	I32GtS
	I32GtU
	I32GeS
	I32GeU

	I64Const
	I64Add
	I64Sub
	I64Mul
	I64DivS
	I64DivU
	I64RemS
	I64RemU
	I64Rotl
	I64Rotr
	I64Clz
	I64Ctz
	I64PopCnt
	I64EqZ
	I64And
	I64Or
	I64Xor
	I64Shl
	I64ShrS
	I64ShrU
	I64Eq
	I64Ne
	I64LtS
	I64LtU
	I64LeS
	I64LeU
	I64GtS
	I64GtU
	I64GeS
	I64GeU

	F32Add
	F32Sub
	F32Mul
	F32Div
	F32Sqrt
	F32Min
	F32Max
	F32Ceil
	F32Floor
	F32Trunc
	F32Nearest
	F32Abs
	F32Neg
	F32CopySign
	F32Eq
	F32Ne
	F32Lt
	F32Le
	F32Gt
	F32Ge

	F64Add
	F64Sub
	F64Mul
	F64Div
	F64Sqrt
	F64Min
	F64Max
	F64Ceil
	F64Floor
	F64Trunc
	F64Nearest
	F64Abs
	F64Neg
	F64CopySign
	F64Eq
	F64Ne
	F64Lt
	F64Le
	F64Gt
	F64Ge

	I32WrapI64
	I32TruncUF32
	I32TruncUF64
	I32TruncSF32
	I32TruncSF64
	I64TruncUF32
	I64TruncUF64
	I64TruncSF32
	I64TruncSF64
	I64ExtendUI32
	I64ExtendSI32

	F32DemoteF64
	F64PromoteF32
	F32ConvertSI32
	F32ConvertSI64
	F32ConvertUI32
	F32ConvertUI64
	F64ConvertSI32
	F64ConvertSI64
	F64ConvertUI32
	F64ConvertUI64

	I32Load
	I64Load

	I32Store
	I64Store

	I32Load8S
	I32Load16S
	I64Load8S
	I64Load16S
	I64Load32S

	I32Load8U
	I32Load16U
	I64Load8U
	I64Load16U
	I64Load32U

	I32Store8
	I32Store16
	I64Store8
	I64Store16
	I64Store32

	Jmp
	JmpIf
	JmpEither
	JmpTable
	ReturnValue
	ReturnVoid

	GetLocal
	SetLocal

	GetGlobal
	SetGlobal

	Call
	CallIndirect
	InvokeImport

	CurrentMemory
	GrowMemory

	Phi

	AddGas

	FPDisabledError

	Unknown
)
