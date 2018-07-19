package exec

import (
	"encoding/binary"
	"github.com/perlin-network/life/compiler"
	"github.com/perlin-network/life/compiler/opcodes"
)

var LE = binary.LittleEndian

type VirtualMachine struct {
	Module *compiler.Module
	FunctionCode [][]byte
	CallStack []Frame
}

type Frame struct {
	FunctionID int
	Regs [32]int64
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
	}
}

func (vm *VirtualMachine) Execute(functionID int) int64 {
	functionInfo := &vm.Module.Base.FunctionIndexSpace[functionID]
	if len(functionInfo.Sig.ParamTypes) != 0 || len(functionInfo.Sig.ReturnTypes) != 0 {
		panic("entry function must have no params or return values")
	}

	vm.CallStack = append(vm.CallStack, Frame {
		FunctionID: functionID,
		Locals: make([]int64, len(functionInfo.Body.Locals)),
	})

	frame := &vm.CallStack[len(vm.CallStack) - 1]
	code := vm.FunctionCode[frame.FunctionID]
	lastValueID := 0

	for {
		valueID := int(LE.Uint32(code[frame.IP:frame.IP + 4]))
		ins := opcodes.Opcode(code[frame.IP + 4])
		frame.IP += 5

		switch ins {
		case opcodes.Nop:
		case opcodes.I32Const:
			val := LE.Uint32(code[frame.IP:frame.IP + 4])
			frame.IP += 4
			frame.Regs[valueID] = int64(val)
		case opcodes.I32Add:
			a := LE.Uint32(code[frame.IP:frame.IP + 4])
			b := LE.Uint32(code[frame.IP + 4 : frame.IP + 8])
			frame.IP += 8
			frame.Regs[valueID] = int64(a + b)
		case opcodes.Jmp:
			frame.IP = int(LE.Uint32(code[frame.IP:frame.IP + 4]))
		case opcodes.JmpIf:
			target := int(LE.Uint32(code[frame.IP:frame.IP + 4]))
			cond := int(LE.Uint32(code[frame.IP + 4:frame.IP + 8]))
			frame.IP += 8
			if frame.Regs[cond] != 0 {
				frame.IP = target
			}
		case opcodes.JmpTable:
			targetCount := int(LE.Uint32(code[frame.IP:frame.IP + 4]))
			frame.IP += 4

			targetsRaw := code[frame.IP : frame.IP + 4 * targetCount]
			frame.IP += 4 * targetCount

			defaultTarget := int(LE.Uint32(code[frame.IP:frame.IP + 4]))
			frame.IP += 4

			cond := int(LE.Uint32(code[frame.IP:frame.IP + 4]))
			frame.IP += 4

			val := int(frame.Regs[cond])
			if val >= 0 && val < targetCount {
				frame.IP = int(LE.Uint32(targetsRaw[val * 4 : val * 4 + 4]))
			} else {
				frame.IP = defaultTarget
			}
		case opcodes.ReturnValue:
			val := frame.Regs[int(LE.Uint32(code[frame.IP : frame.IP + 4]))]
			vm.CallStack = vm.CallStack[:len(vm.CallStack) - 1]
			if len(vm.CallStack) == 0 {
				return val
			} else {
				frame = &vm.CallStack[len(vm.CallStack) - 1]
				code = vm.FunctionCode[frame.FunctionID]
				frame.RecvReturnValue = val
			}
		case opcodes.ReturnVoid:
			vm.CallStack = vm.CallStack[:len(vm.CallStack) - 1]
			if len(vm.CallStack) == 0 {
				return 0
			} else {
				frame = &vm.CallStack[len(vm.CallStack) - 1]
				code = vm.FunctionCode[frame.FunctionID]
			}
		case opcodes.GetLocal:
			val := frame.Locals[int(LE.Uint32(code[frame.IP : frame.IP + 4]))]
			frame.IP += 4
			frame.Regs[valueID] = val
		case opcodes.SetLocal:
			id := int(LE.Uint32(code[frame.IP : frame.IP + 4]))
			val := frame.Regs[int(LE.Uint32(code[frame.IP + 4: frame.IP + 8]))]
			frame.IP += 8
			frame.Locals[id] = val
		case opcodes.Phi:
			frame.Regs[valueID] = frame.Regs[lastValueID]
		default:
			panic("unknown instruction")
		}

		if valueID != 0 {
			lastValueID = valueID
		}
	}
}
