// Value liveness analysis & register allocation.

package compiler

func (c *SSAFunctionCompiler) RegAlloc() {
	regID := TyValueID(1)
	valueRelocs := make(map[TyValueID]TyValueID)
	for _, values := range c.StackValueSets {
		for _, v := range values {
			valueRelocs[v] = regID
		}
		regID++
	}
	for i, _ := range c.Code {
		ins := &c.Code[i]

		if ins.Target != 0 {
			if reg, ok := valueRelocs[ins.Target]; ok {
				ins.Target = reg
			} else {
				panic("Register not found for target")
			}
		}

		for j, v := range ins.Values {
			if v != 0 {
				if reg, ok := valueRelocs[v]; ok {
					ins.Values[j] = reg
				} else {
					panic("Register not found for value")
				}
			}
		}
	}
}

func (ins *Instr) BranchTargets() []int {
	switch ins.Op {
	case "jmp":
	case "jmp_if":
	case "jmp_table":
	default:
		return []int{}
	}

	ret := make([]int, len(ins.Immediates))
	for i, t := range ins.Immediates {
		ret[i] = int(t)
	}
	return ret
}
