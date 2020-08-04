/*
 * Copyright (c) 2020-present Heeus Authors
 */

package calls

import (
	"errors"

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
	gas        uint
	gasLimit   uint
}

func newFastVM() (res *fastvm) {
	res = &fastvm{}
	res.valueSlots = make([]int64, 1000)
	res.gasLimit = ^uint(0)
	return res
}

func (vm *fastvm) exec(fn *function, params ...int64) (res int64, err error) {
	return vm.execinternal(0, fn, params...)
}

func (vm *fastvm) execinternal(slot int, fn *function, params ...int64) (res int64, err error) {

	if fn.NumParams != len(params) {
		panic("param count mismatch")
	}

	numValueSlots := fn.NumRegs + fn.NumParams + fn.NumLocals
	Regs := vm.valueSlots[slot : slot+fn.NumRegs]
	Locals := vm.valueSlots[slot+fn.NumRegs : slot+numValueSlots]
	copy(Locals, params)

	gas := vm.gas
	// gasLimit := vm.gasLimit

	ip := 0
	for {
		ins := fn.inss[ip]
		valueID := ins.valueID
		ip++
		gas++
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
			if vm.gas > vm.gasLimit {
				return 0, errors.New("Gas limit exceeded")
			}

			target := int(ins.v1)
			cond := int(ins.v2)
			// yieldedReg := int(LE.Uint32(frame.Code[frame.IP+8 : frame.IP+12]))
			// frame.IP += 12
			if Regs[cond] != 0 {
				ip = target
			}
		case opcodes.Jmp:
			if vm.gas > vm.gasLimit {
				return 0, errors.New("Gas limit exceeded")
			}

			target := int(ins.v1)
			// vm.Yielded = frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			ip = target
		case opcodes.ReturnValue:
			val := Regs[int(ins.v1)]
			return val, nil
		default:
			panic("Unknown op")
		}
	}
}
