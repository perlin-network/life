package codegen

import (
	"github.com/perlin-network/life/compiler"
)

func TranslateFunction(functionID int, c *compiler.SSAFunctionCompiler) string {
	functionInfo := &c.Module.FunctionIndexSpace[functionID]
	ir := &FunctionIR {
		ID: functionID,
		NumParams: len(functionInfo.Sig.ParamTypes),
		HasReturn: len(functionInfo.Sig.ReturnTypes) != 0,
	}
	for i, ins := range c.Code {
		insSet := make([]Instr, 0)
		switch ins.Op {
		case "i32.const":
			insSet = append(insSet, NewInstr(ins.Target, "add i32 %d, 0", int32(ins.Immediates[0])))
			insSet = append(insSet, NewBr(i + 1))
		case "i32.add":
			insSet = append(insSet, NewInstr(ins.Target, "add i32 %s, %s", RefValue(ins.Values[0]), RefValue(ins.Values[1])))
			insSet = append(insSet, NewBr(i + 1))
		case "jmp":
			insSet = append(insSet, NewBr(int(ins.Immediates[0])))
		case "jmp_if":
			asI1 := ir.RefTempValue()

			insSet = append(insSet, NewInstr(0, "%s = icmp ne %s, 0", asI1, RefValue(ins.Values[0])))
			insSet = append(insSet, NewCondBr(asI1, int(ins.Immediates[0]), i + 1))
		//case "return":
			// TODO
		default:
			panic(ins.Op)
		}
		ir.InstructionSets = append(ir.InstructionSets, insSet)
	}
	return ir.Serialize()
}
