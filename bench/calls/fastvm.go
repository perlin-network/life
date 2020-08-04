/*
 * Copyright (c) 2020-present Heeus Authors
 */

package calls

import (
	"github.com/perlin-network/life/compiler/opcodes"
)

type function struct {
	inss       []ins
	NumRegs    int
	NumParams  int
	NumLocals  int
	NumReturns int
}

type ins struct {
	opcode  opcodes.Opcode
	valueID uint32
	v1      uint32
	v2      uint32
}

type fastvm struct {
	valueSlots []int64
}

func newFastVM() (res *fastvm) {
	res = &fastvm{}
	res.valueSlots = make([]int64, 1000)
	return res
}

func (vm *fastvm) exec(fn *function, params ...int64) int64 {
	return vm.execinternal(0, fn, params...)
}

func (vm *fastvm) execinternal(slot int, fn *function, params ...int64) int64 {

	if fn.NumParams != len(params) {
		panic("param count mismatch")
	}

	numValueSlots := fn.NumRegs + fn.NumParams + fn.NumLocals
	Regs := vm.valueSlots[slot : slot+fn.NumRegs]
	Locals := vm.valueSlots[slot+fn.NumRegs : slot+numValueSlots]
	copy(Locals, params)

	ip := 0
	for {
		ins := fn.inss[ip]
		valueID := ins.valueID
		ip++
		switch ins.opcode {
		case opcodes.GetLocal:
			id := ins.v1
			val := Locals[id]
			Regs[valueID] = val
		case opcodes.I32Const:
			val := ins.v1
			Regs[valueID] = int64(val)
		case opcodes.I32GeS:
			a := int32(ins.v1)
			b := int32(ins.v2)
			if a >= b {
				Regs[valueID] = 1
			} else {
				Regs[valueID] = 0
			}
		case opcodes.JmpIf:
			target := int(ins.v1)
			cond := int(ins.v2)
			// yieldedReg := int(LE.Uint32(frame.Code[frame.IP+8 : frame.IP+12]))
			// frame.IP += 12
			if Regs[cond] != 0 {
				ip = target
			}
		case opcodes.Jmp:
			target := int(ins.v1)
			// vm.Yielded = frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			ip = target
		case opcodes.ReturnValue:
			val := Regs[int(ins.v1)]
			return val
		default:
			panic("Unknown op")
		}
	}
}

func newCallSumAndAdd1_0() (res *function) {
	fn := function{}
	fn.NumParams = 3
	fn.NumRegs = 3
	fn.NumLocals = 0

	fn.inss = append(fn.inss, ins{valueID: 1, opcode: opcodes.GetLocal, v1: 2, v2: 2})
	fn.inss = append(fn.inss, ins{valueID: 2, opcode: opcodes.I32Const, v1: 1, v2: 1})
	fn.inss = append(fn.inss, ins{valueID: 1, opcode: opcodes.I32GeS, v1: 1, v2: 2})
	fn.inss = append(fn.inss, ins{valueID: 0, opcode: opcodes.JmpIf, v1: 61, v2: 1})
	fn.inss = append(fn.inss, ins{valueID: 0, opcode: opcodes.Jmp, v1: 5, v2: 0})
	fn.inss = append(fn.inss, ins{valueID: 1, opcode: opcodes.GetLocal, v1: 0, v2: 0})
	fn.inss = append(fn.inss, ins{valueID: 0, opcode: opcodes.ReturnValue, v1: 1, v2: 0})
	return &fn
}
