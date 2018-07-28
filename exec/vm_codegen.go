package exec

import (
	"fmt"
	"github.com/perlin-network/life/compiler/opcodes"
)

// Generate C code for the given function.
// Returns true if codegen succeeds, or false if the current function cannot be code-generated.
func (vm *VirtualMachine) GenerateCodeForFunction(functionID int) bool {
	code := vm.FunctionCode[functionID]

	program := `
typedef long long i64;
typedef int i32;
	`

	// Returns -1 for done. The return value should have already be written in ret.
	// Return >= 0 for continuation. In this case, the instruction location should be
	// written in `ret` and only the current instruction will get interpreted.
	program += `
i32 run(i64 *regs, i64 *locals, i64 *yielded, i32 continuation, i64 *ret) {
	switch(continuation) {
	case 0:
	`

	cont := 1
	ip := 0

	for {
		if ip == len(code.Bytes) {
			break
		}
		program += fmt.Sprintf("I%d:\n", ip)
		thisIP := ip

		valueID := int(LE.Uint32(code.Bytes[ip : ip+4]))
		ins := opcodes.Opcode(code.Bytes[ip+4])
		ip += 5

		switch ins {
		case opcodes.Nop:
		case opcodes.Unreachable:
			program += "return -2;\n"
		case opcodes.I32Const:
			val := int64(LE.Uint32(code.Bytes[ip:ip+4]))
			ip += 4
			program += fmt.Sprintf("regs[%d] = %dLL;\n", valueID, val)
		case opcodes.I32Add:
			a := int(LE.Uint32(code.Bytes[ip : ip + 4]))
			b := int(LE.Uint32(code.Bytes[ip + 4 : ip + 8]))
			ip += 8
			program += fmt.Sprintf("regs[%d] = (i64)((i32) regs[%d] + (i32) regs[%d]);\n", valueID, a, b)
		case opcodes.ReturnVoid:
			program += "return -1;\n"
		case opcodes.ReturnValue:
			val := int(LE.Uint32(code.Bytes[ip:ip+4]))
			ip += 4
			program += fmt.Sprintf("*ret = regs[%d];\n", val)
			program += "return -1;\n"
		case opcodes.Call:
			ip += 4
			argCount := int(LE.Uint32(code.Bytes[ip:ip+4]))
			ip += 4
			ip += 4 * argCount
			program += fmt.Sprintf("*ret = %d;\n", thisIP)
			program += fmt.Sprintf("return %d;\n", cont)
			program += fmt.Sprintf("case %d:\n", cont)
			cont++
		case opcodes.CallIndirect:
			ip += 4
			argCount := int(LE.Uint32(code.Bytes[ip:ip+4]))
			ip += 4
			ip += 4 * argCount
			ip += 4 // table item id
			program += fmt.Sprintf("*ret = %d;\n", thisIP)
			program += fmt.Sprintf("return %d;\n", cont)
			program += fmt.Sprintf("case %d:\n", cont)
			cont++
		case opcodes.Jmp:
			target := int(LE.Uint32(code.Bytes[ip : ip+4]))
			yieldReg := int(LE.Uint32(code.Bytes[ip+4 : ip+8]))
			ip += 8

			program += fmt.Sprintf("*yielded = regs[%d];\n", yieldReg)
			program += fmt.Sprintf("goto I%d;\n", target)
		case opcodes.JmpIf:
			target := int(LE.Uint32(code.Bytes[ip : ip+4]))
			cond := int(LE.Uint32(code.Bytes[ip+4 : ip+8]))
			yieldReg := int(LE.Uint32(code.Bytes[ip+8 : ip+12]))
			ip += 12

			program += fmt.Sprintf("if(regs[%d]) {\n", cond)
			program += fmt.Sprintf("*yielded = regs[%d];\n", yieldReg)
			program += fmt.Sprintf("goto I%d;\n", target)
			program += "}\n"
		case opcodes.Phi:
			fmt.Sprintf("regs[%d] = *yielded\n", valueID)
		default:
			return false
		}
	}

	program += `
	}
	return -3; // invalid
}
	`
	fmt.Println(program)
	return true
}
