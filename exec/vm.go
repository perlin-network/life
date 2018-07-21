package exec

import (
	"encoding/binary"
	"github.com/perlin-network/life/compiler"
	"github.com/perlin-network/life/compiler/opcodes"
)

const DefaultCallStackSize = 512

var LE = binary.LittleEndian

type VirtualMachine struct {
	Module *compiler.Module
	FunctionCode []compiler.InterpreterCode
	CallStack []Frame
	CurrentFrame int
	Table []uint32
}

type Frame struct {
	FunctionID int
	Code []byte
	Regs []int64
	Locals []int64
	IP int
	ReturnReg int
}

func NewVirtualMachine(code []byte) *VirtualMachine {
	m, err := compiler.LoadModule(code)
	if err != nil {
		panic(err)
	}

	table := make([]uint32, 0)
	if m.Base.Table != nil && len(m.Base.Table.Entries) > 0{
		t := &m.Base.Table.Entries[0]

		table = make([]uint32, int(t.Limits.Initial))
		for i := 0; i < int(t.Limits.Initial); i++ {
			table[i] = 0xffffffff
		}

		if m.Base.Elements != nil && len(m.Base.Elements.Entries) > 0 {
			for _, e := range m.Base.Elements.Entries {
				maybeOffset, err := m.Base.ExecInitExpr(e.Offset)
				if err != nil {
					panic(err)
				}
				offset := int(maybeOffset.(int32))
				copy(table[offset:], e.Elems)
			}
		}
	}

	return &VirtualMachine{
		Module: m,
		FunctionCode: m.CompileForInterpreter(),
		CallStack: make([]Frame, DefaultCallStackSize),
		CurrentFrame: -1,
		Table: table,
	}
}

func (f *Frame) Init(functionID int, code compiler.InterpreterCode, numTotalLocals int) {
	values := make([]int64, code.NumRegs + numTotalLocals)

	f.FunctionID = functionID
	f.Regs = values[:code.NumRegs]
	f.Locals = values[code.NumRegs:]
	f.Code = code.Bytes
	f.IP = 0
}

func (vm *VirtualMachine) GetCurrentFrame() *Frame {
	if vm.CurrentFrame >= len(vm.CallStack) {
		panic("call stack overflow")
		//vm.CallStack = append(vm.CallStack, make([]Frame, DefaultCallStackSize / 2)...)
	}
	return &vm.CallStack[vm.CurrentFrame]
}

func (vm *VirtualMachine) Execute(functionID int) int64 {
	functionInfo := &vm.Module.Base.FunctionIndexSpace[functionID]
	if len(functionInfo.Sig.ParamTypes) != 0 {
		panic("entry function must have no params")
	}

	vm.CurrentFrame++

	frame := vm.GetCurrentFrame()
	frame.Init(functionID, vm.FunctionCode[functionID], len(functionInfo.Body.Locals))

	var yielded int64

	for {
		valueID := int(LE.Uint32(frame.Code[frame.IP:frame.IP + 4]))
		ins := opcodes.Opcode(frame.Code[frame.IP + 4])
		frame.IP += 5

		switch ins {
		case opcodes.Nop:
		case opcodes.Unreachable:
			panic("wasm: unreachable executed")
		case opcodes.I32Const:
			val := LE.Uint32(frame.Code[frame.IP:frame.IP + 4])
			frame.IP += 4
			frame.Regs[valueID] = int64(val)
		case opcodes.I32Add:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP + 4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP + 4 : frame.IP + 8]))])
			frame.IP += 8
			frame.Regs[valueID] = int64(a + b)
		case opcodes.I32Eq:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP + 4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP + 4 : frame.IP + 8]))])
			frame.IP += 8
			if a == b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.Jmp:
			target := int(LE.Uint32(frame.Code[frame.IP:frame.IP + 4]))
			yielded = frame.Regs[int(LE.Uint32(frame.Code[frame.IP + 4 : frame.IP + 8]))]
			frame.IP = target
		case opcodes.JmpIf:
			target := int(LE.Uint32(frame.Code[frame.IP:frame.IP + 4]))
			cond := int(LE.Uint32(frame.Code[frame.IP + 4:frame.IP + 8]))
			yieldedReg := int(LE.Uint32(frame.Code[frame.IP + 8 : frame.IP + 12]))
			frame.IP += 12
			if frame.Regs[cond] != 0 {
				yielded = frame.Regs[yieldedReg]
				frame.IP = target
			}
		case opcodes.JmpTable:
			targetCount := int(LE.Uint32(frame.Code[frame.IP:frame.IP + 4]))
			frame.IP += 4

			targetsRaw := frame.Code[frame.IP : frame.IP + 4 * targetCount]
			frame.IP += 4 * targetCount

			defaultTarget := int(LE.Uint32(frame.Code[frame.IP:frame.IP + 4]))
			frame.IP += 4

			cond := int(LE.Uint32(frame.Code[frame.IP:frame.IP + 4]))
			frame.IP += 4

			yielded = frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP + 4]))]
			frame.IP += 4

			val := int(frame.Regs[cond])
			if val >= 0 && val < targetCount {
				frame.IP = int(LE.Uint32(targetsRaw[val * 4 : val * 4 + 4]))
			} else {
				frame.IP = defaultTarget
			}
		case opcodes.ReturnValue:
			val := frame.Regs[int(LE.Uint32(frame.Code[frame.IP : frame.IP + 4]))]
			vm.CurrentFrame--
			if vm.CurrentFrame == -1 {
				return val
			} else {
				frame = vm.GetCurrentFrame()
				frame.Regs[frame.ReturnReg] = val
			}
		case opcodes.ReturnVoid:
			vm.CurrentFrame--
			if vm.CurrentFrame == -1 {
				return 0
			} else {
				frame = vm.GetCurrentFrame()
			}
		case opcodes.GetLocal:
			val := frame.Locals[int(LE.Uint32(frame.Code[frame.IP : frame.IP + 4]))]
			frame.IP += 4
			frame.Regs[valueID] = val
		case opcodes.SetLocal:
			id := int(LE.Uint32(frame.Code[frame.IP : frame.IP + 4]))
			val := frame.Regs[int(LE.Uint32(frame.Code[frame.IP + 4: frame.IP + 8]))]
			frame.IP += 8
			frame.Locals[id] = val
		case opcodes.Call:
			functionID = int(LE.Uint32(frame.Code[frame.IP : frame.IP + 4]))
			frame.IP += 4
			argCount := int(LE.Uint32(frame.Code[frame.IP : frame.IP + 4]))
			frame.IP += 4
			argsRaw := frame.Code[frame.IP : frame.IP + 4 * argCount]
			frame.IP += 4 * argCount

			functionInfo = &vm.Module.Base.FunctionIndexSpace[functionID]

			oldRegs := frame.Regs
			frame.ReturnReg = valueID

			vm.CurrentFrame++
			frame = vm.GetCurrentFrame()
			frame.Init(functionID, vm.FunctionCode[functionID], argCount + len(functionInfo.Body.Locals))
			for i := 0; i < argCount; i++ {
				frame.Locals[i] = oldRegs[int(LE.Uint32(argsRaw[i * 4 : i * 4 + 4]))]
			}

		case opcodes.CallIndirect:
			typeID := int(LE.Uint32(frame.Code[frame.IP : frame.IP + 4]))
			frame.IP += 4
			argCount := int(LE.Uint32(frame.Code[frame.IP : frame.IP + 4])) - 1
			frame.IP += 4
			argsRaw := frame.Code[frame.IP : frame.IP + 4 * argCount]
			frame.IP += 4 * argCount
			tableItemID := frame.Regs[int(LE.Uint32(frame.Code[frame.IP : frame.IP + 4]))]
			frame.IP += 4

			sig := &vm.Module.Base.Types.Entries[typeID]

			functionID = int(vm.Table[tableItemID])
			functionInfo = &vm.Module.Base.FunctionIndexSpace[functionID]

			// TODO: We are only checking CC here; Do we want strict typeck?
			if len(functionInfo.Sig.ParamTypes) != len(sig.ParamTypes) ||
				len(functionInfo.Sig.ReturnTypes) != len(sig.ReturnTypes) {
				panic("type mismatch")
			}

			oldRegs := frame.Regs
			frame.ReturnReg = valueID

			vm.CurrentFrame++
			frame = vm.GetCurrentFrame()
			frame.Init(functionID, vm.FunctionCode[functionID], argCount + len(functionInfo.Body.Locals))
			for i := 0; i < argCount; i++ {
				frame.Locals[i] = oldRegs[int(LE.Uint32(argsRaw[i * 4 : i * 4 + 4]))]
			}

		case opcodes.Phi:
			frame.Regs[valueID] = yielded
		default:
			panic("unknown instruction")
		}
	}
}
