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
	FunctionCode [][]byte
	CallStack []Frame
	CurrentFrame int
}

type Frame struct {
	FunctionID int
	Code []byte
	Regs [32]int64 // Hmm... It should be rare for normal applications to use more than 32 regs.
	Locals []int64
	IP int
	RecvReturnValue int64
}

func NewVirtualMachine(code []byte) *VirtualMachine {
	m, err := compiler.LoadModule(code)
	if err != nil {
		panic(err)
	}

	return &VirtualMachine{
		Module: m,
		FunctionCode: m.CompileForInterpreter(),
		CallStack: make([]Frame, DefaultCallStackSize),
		CurrentFrame: -1,
	}
}

func (vm *VirtualMachine) GetCurrentFrame() *Frame {
	if vm.CurrentFrame >= len(vm.CallStack) {
		vm.CallStack = append(vm.CallStack, make([]Frame, DefaultCallStackSize / 2)...)
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
	frame.FunctionID = functionID
	frame.Locals = make([]int64, len(functionInfo.Body.Locals))
	frame.Code = vm.FunctionCode[frame.FunctionID]
	frame.IP = 0

	var yielded int64

	for {
		valueID := int(LE.Uint32(frame.Code[frame.IP:frame.IP + 4]))
		ins := opcodes.Opcode(frame.Code[frame.IP + 4])
		frame.IP += 5

		switch ins {
		case opcodes.Nop:
		case opcodes.I32Const:
			val := LE.Uint32(frame.Code[frame.IP:frame.IP + 4])
			frame.IP += 4
			frame.Regs[valueID] = int64(val)
		case opcodes.I32Add:
			a := LE.Uint32(frame.Code[frame.IP:frame.IP + 4])
			b := LE.Uint32(frame.Code[frame.IP + 4 : frame.IP + 8])
			frame.IP += 8
			frame.Regs[valueID] = int64(a + b)
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
				frame.RecvReturnValue = val
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
		case opcodes.Phi:
			frame.Regs[valueID] = yielded
		default:
			panic("unknown instruction")
		}
	}
}
