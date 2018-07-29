package exec

import (
	"fmt"
	"github.com/perlin-network/life/compiler/opcodes"
	"github.com/perlin-network/life/compiler"
)

type jitContext struct {
	vm *VirtualMachine
	functionID int
	code *compiler.InterpreterCode
	program string
	cont int
	ip int
	thisIP int
}

func (c *jitContext) writeFallback() {
	c.program += fmt.Sprintf("*ret = %d;\n", c.thisIP)
	c.program += fmt.Sprintf("return %d;\n", c.cont)
	c.program += fmt.Sprintf("case %d:\n", c.cont)
	c.cont++
}

func (c *jitContext) checkLocal(id int) {
	if id < 0 || id >= c.code.NumParams + c.code.NumLocals {
		panic("local out of bounds")
	}
}

func (c *jitContext) checkReg(id int) {
	if id < 0 || id >= c.code.NumRegs {
		panic("reg out of bounds")
	}
}

func (c *jitContext) Generate() bool {
	c.program = `
typedef long long i64;
typedef int i32;
	`

	// Returns -1 for done. The return value should have already be written in ret.
	// Return >= 0 for continuation. In this case, the instruction location should be
	// written in `ret` and only the current instruction will get interpreted.
	c.program += `
i32 run(i64 *regs, i64 *locals, i64 *yielded, i32 continuation, i64 *ret) {
	switch(continuation) {
	case 0:
	`

	c.cont = 1
	c.ip = 0

	for {
		if c.ip == len(c.code.Bytes) {
			break
		}
		c.program += fmt.Sprintf("I%d:\n", c.ip)
		c.thisIP = c.ip

		valueID := int(LE.Uint32(c.code.Bytes[c.ip : c.ip+4]))
		c.checkReg(valueID)

		ins := opcodes.Opcode(c.code.Bytes[c.ip+4])
		c.ip += 5

		switch ins {
		case opcodes.Nop:
		case opcodes.Unreachable:
			c.program += "return -2;\n"

		case opcodes.I32Const:
			imm := int64(LE.Uint32(c.code.Bytes[c.ip:c.ip+4]))
			c.ip += 4
			c.program += fmt.Sprintf("regs[%d] = %dLL;\n", valueID, imm)
		case opcodes.I32Add:
			a := int(LE.Uint32(c.code.Bytes[c.ip : c.ip + 4]))
			c.checkReg(a)
			b := int(LE.Uint32(c.code.Bytes[c.ip + 4 : c.ip + 8]))
			c.checkReg(b)

			c.ip += 8
			c.program += fmt.Sprintf("regs[%d] = (i64)((i32) regs[%d] + (i32) regs[%d]);\n", valueID, a, b)
		case opcodes.I32Eq:
			a := int(LE.Uint32(c.code.Bytes[c.ip : c.ip + 4]))
			c.checkReg(a)
			b := int(LE.Uint32(c.code.Bytes[c.ip + 4 : c.ip + 8]))
			c.checkReg(b)

			c.ip += 8
			c.program += fmt.Sprintf("regs[%d] = (i64)((i32) regs[%d] == (i32) regs[%d]);", valueID, a, b)
		case opcodes.I64Const:
			imm := int64(LE.Uint64(c.code.Bytes[c.ip:c.ip+8]))
			c.ip += 8
			c.program += fmt.Sprintf("regs[%d] = %dLL;\n", valueID, imm)
		case opcodes.I64Add:
			a := int(LE.Uint32(c.code.Bytes[c.ip : c.ip + 4]))
			c.checkReg(a)
			b := int(LE.Uint32(c.code.Bytes[c.ip + 4 : c.ip + 8]))
			c.checkReg(b)

			c.ip += 8
			c.program += fmt.Sprintf("regs[%d] = regs[%d] + regs[%d];\n", valueID, a, b)
		case opcodes.I64Eq:
			a := int(LE.Uint32(c.code.Bytes[c.ip : c.ip + 4]))
			c.checkReg(a)
			b := int(LE.Uint32(c.code.Bytes[c.ip + 4 : c.ip + 8]))
			c.checkReg(b)

			c.ip += 8
			c.program += fmt.Sprintf("regs[%d] = (i64)(regs[%d] == regs[%d]);", valueID, a, b)
		
		case opcodes.GetLocal:
			id := int(LE.Uint32(c.code.Bytes[c.ip : c.ip+4]))
			c.checkLocal(id)

			c.ip += 4
			c.program += fmt.Sprintf("regs[%d] = locals[%d];\n", valueID, id)
		case opcodes.SetLocal:
			id := int(LE.Uint32(c.code.Bytes[c.ip:c.ip+4]))
			c.checkLocal(id)

			val := int(LE.Uint32(c.code.Bytes[c.ip+4:c.ip+8]))
			c.checkReg(val)

			c.ip += 8
			c.program += fmt.Sprintf("locals[%d] = regs[%d];\n", id, val)
		case opcodes.ReturnVoid:
			c.writeFallback()
		case opcodes.ReturnValue:
			c.ip += 4
			c.writeFallback()
		case opcodes.Call:
			c.ip += 4
			argCount := int(LE.Uint32(c.code.Bytes[c.ip:c.ip+4]))
			c.ip += 4
			c.ip += 4 * argCount
			c.writeFallback()
		case opcodes.CallIndirect:
			c.ip += 4
			argCount := int(LE.Uint32(c.code.Bytes[c.ip:c.ip+4]))
			c.ip += 4
			c.ip += 4 * argCount
			c.ip += 4 // table item id
			c.writeFallback()
		case opcodes.Jmp:
			target := int(LE.Uint32(c.code.Bytes[c.ip : c.ip+4]))
			yieldReg := int(LE.Uint32(c.code.Bytes[c.ip+4 : c.ip+8]))
			c.checkReg(yieldReg)
			c.ip += 8

			c.program += fmt.Sprintf("*yielded = regs[%d];\n", yieldReg)
			c.program += fmt.Sprintf("goto I%d;\n", target)
		case opcodes.JmpIf:
			target := int(LE.Uint32(c.code.Bytes[c.ip : c.ip+4]))

			cond := int(LE.Uint32(c.code.Bytes[c.ip+4 : c.ip+8]))
			c.checkReg(cond)

			yieldReg := int(LE.Uint32(c.code.Bytes[c.ip+8 : c.ip+12]))
			c.checkReg(yieldReg)

			c.ip += 12

			c.program += fmt.Sprintf("if(regs[%d]) {\n", cond)
			c.program += fmt.Sprintf("*yielded = regs[%d];\n", yieldReg)
			c.program += fmt.Sprintf("goto I%d;\n", target)
			c.program += "}\n"
		case opcodes.Phi:
			fmt.Sprintf("regs[%d] = *yielded\n", valueID)
		default:
			return false
		}
	}

	c.program += `
	break;
	}
	return -3; // invalid
}
	`
	fmt.Println(c.program)
	c.code.JITInfo = CompileDynamicModule(c.program)
	return true
}

// Generate C code for the given function.
// Returns true if codegen succeeds, or false if the current function cannot be code-generated.
func (vm *VirtualMachine) GenerateCodeForFunction(functionID int) bool {
	code := &vm.FunctionCode[functionID]
	c := &jitContext {
		vm: vm,
		functionID: functionID,
		code: code,
	}
	return c.Generate()
}
