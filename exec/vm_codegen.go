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
		panic(fmt.Errorf("local out of bounds: id = %d, n = %d", id, c.code.NumParams + c.code.NumLocals))
	}
}

func (c *jitContext) checkReg(id int) {
	if id < 0 || id >= c.code.NumRegs {
		panic(fmt.Errorf("reg out of bounds: id = %d, n = %d", id, c.code.NumRegs))
	}
}

func (c *jitContext) checkGlobal(id int) {
	if id < 0 || id >= len(c.vm.Globals) {
		panic(fmt.Errorf("global out of bounds: id = %d, n = %d", id, len(c.vm.Globals)))
	}
}

func (c *jitContext) writeSI32Op(valueID int, op string) {
	a := int(LE.Uint32(c.code.Bytes[c.ip : c.ip + 4]))
	c.checkReg(a)
	b := int(LE.Uint32(c.code.Bytes[c.ip + 4 : c.ip + 8]))
	c.checkReg(b)

	c.ip += 8
	c.program += fmt.Sprintf("regs[%d] = (i64)((i32) regs[%d] %s (i32) regs[%d]);\n", valueID, a, op, b)
}

func (c *jitContext) writeUI32Op(valueID int, op string) {
	a := int(LE.Uint32(c.code.Bytes[c.ip : c.ip + 4]))
	c.checkReg(a)
	b := int(LE.Uint32(c.code.Bytes[c.ip + 4 : c.ip + 8]))
	c.checkReg(b)

	c.ip += 8
	c.program += fmt.Sprintf("regs[%d] = (i64)((u32) regs[%d] %s (u32) regs[%d]);\n", valueID, a, op, b)
}

func (c *jitContext) writeSI64Op(valueID int, op string) {
	a := int(LE.Uint32(c.code.Bytes[c.ip : c.ip + 4]))
	c.checkReg(a)
	b := int(LE.Uint32(c.code.Bytes[c.ip + 4 : c.ip + 8]))
	c.checkReg(b)

	c.ip += 8
	c.program += fmt.Sprintf("regs[%d] = regs[%d] %s regs[%d];\n", valueID, a, op, b)
}

func (c *jitContext) writeUI64Op(valueID int, op string) {
	a := int(LE.Uint32(c.code.Bytes[c.ip : c.ip + 4]))
	c.checkReg(a)
	b := int(LE.Uint32(c.code.Bytes[c.ip + 4 : c.ip + 8]))
	c.checkReg(b)

	c.ip += 8
	c.program += fmt.Sprintf("regs[%d] = (u64) regs[%d] %s (u64) regs[%d];\n", valueID, a, op, b)
}

func (c *jitContext) Generate() bool {
	c.program = `
typedef long long i64;
typedef int i32;
typedef unsigned long long u64;
typedef unsigned int u32;
typedef unsigned char u8;
`

	// Returns -1 for done. The return value should have already be written in ret.
	// Return >= 0 for continuation. In this case, the instruction location should be
	// written in `ret` and only the current instruction will get interpreted.
	c.program += `
i32 run(
	i64 *regs,
	i64 *locals,
	i64 *globals,
	u8 *memory,
	i64 memory_len,
	i64 *yielded,
	i32 continuation,
	i64 *ret
) {
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
			c.writeUI32Op(valueID, "+")
		case opcodes.I32Eq:
			c.writeUI32Op(valueID, "==")
		case opcodes.I64Const:
			imm := int64(LE.Uint64(c.code.Bytes[c.ip:c.ip+8]))
			c.ip += 8
			c.program += fmt.Sprintf("regs[%d] = %dLL;\n", valueID, imm)
		case opcodes.I64Add:
			c.writeUI64Op(valueID, "+")
		case opcodes.I64Eq:
			c.writeUI64Op(valueID, "==")
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
		case opcodes.GetGlobal:
			id := int(LE.Uint32(c.code.Bytes[c.ip : c.ip+4]))
			c.checkGlobal(id)

			c.ip += 4
			c.program += fmt.Sprintf("regs[%d] = globals[%d];\n", valueID, id)
		case opcodes.SetGlobal:
			id := int(LE.Uint32(c.code.Bytes[c.ip:c.ip+4]))
			c.checkGlobal(id)

			val := int(LE.Uint32(c.code.Bytes[c.ip+4:c.ip+8]))
			c.checkReg(val)

			c.ip += 8
			c.program += fmt.Sprintf("globals[%d] = regs[%d];\n", id, val)
		case opcodes.ReturnVoid:
			c.writeFallback()
		case opcodes.ReturnValue:
			c.ip += 4
			c.writeFallback()
		case opcodes.Call, opcodes.CallIndirect:
			c.ip += 4
			argCount := int(LE.Uint32(c.code.Bytes[c.ip:c.ip+4]))
			c.ip += 4
			c.ip += 4 * argCount
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
			fmt.Printf("unsupported op: %s\n", ins.String())
			return false
		}
	}

	c.program += `
	break;
	}
	return -3; // invalid
}
	`
	//fmt.Println(c.program)
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
