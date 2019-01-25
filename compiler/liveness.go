// Value liveness analysis & register allocation.

package compiler

import (
	"sort"
)

func (c *SSAFunctionCompiler) RegAlloc() int {
	cfg := c.NewCFGraph()
	//cfg.Print()
	valueLiveness, valueIDUpperBound := cfg.AnalyzeLiveness()

	for _, arr := range valueLiveness {
		sort.Ints(arr)
		/*fmt.Printf("Liveness %d:", i)
		for _, v := range arr {
			fmt.Printf(" %d", v)
		}
		fmt.Println()*/
	}

	sets := make([][]TyValueID, 0)

	for i := TyValueID(1); i < valueIDUpperBound; i++ {
		if valueLiveness[i] == nil {
			continue
		}
		members := []TyValueID{i}
		for j := TyValueID(1); j < valueIDUpperBound; j++ {
			if i == j {
				continue
			}
			if valueLiveness[j] == nil {
				continue
			}
			a, b := 0, 0
			left := valueLiveness[i]
			right := valueLiveness[j]

			conflicting := false
			for a < len(left) && b < len(right) {
				if left[a] < right[b] {
					a++
				} else if left[a] > right[b] {
					b++
				} else {
					conflicting = true
					break
				}
			}
			if !conflicting {
				valueLiveness[i] = append(left, right...)
				sort.Ints(valueLiveness[i])
				valueLiveness[j] = nil
				members = append(members, j)
			}
		}
		if len(members) > 0 {
			sets = append(sets, members)
		}
	}

	/*fmt.Println("-----BEGIN-----")

	for _, s := range sets {
		fmt.Printf("Non-conflicting set:")
		for _, v := range s {
			fmt.Printf(" %d", v)
		}
		fmt.Println()
	}

	fmt.Println("-----END-----")*/

	valueRelocs := make(map[TyValueID]TyValueID)
	for i, set := range sets {
		for _, old := range set {
			valueRelocs[old] = TyValueID(i + 1)
		}
	}
	for i := range c.Code {
		ins := &c.Code[i]

		if ins.Target != 0 {
			if reg, ok := valueRelocs[ins.Target]; ok {
				ins.Target = reg
			} else {
				ins.Target = 0
				panic("Register not found for target")
			}
		}

		for j, v := range ins.Values {
			if v != 0 {
				if reg, ok := valueRelocs[v]; ok {
					ins.Values[j] = reg
				} else {
					ins.Values[j] = 0
					panic("Register not found for value")
				}
			}
		}
	}

	return len(sets) + 1
}

func (ins *Instr) BranchTargets() []int {
	switch ins.Op {
	case "jmp", "jmp_if", "jmp_table":
		ret := make([]int, len(ins.Immediates))
		for i, t := range ins.Immediates {
			ret[i] = int(t)
		}
		return ret

	default:
		return []int{}
	}
}
