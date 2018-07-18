package compiler

import (
	"github.com/go-interpreter/wagon/wasm"
	"github.com/go-interpreter/wagon/disasm"
)

type TyValueID uint64

type SSAFunctionCompiler struct {
	Module *wasm.Module
	Source *disasm.Disassembly

	Code []Instr
	Stack []TyValueID
	Locations []*Location

	ValueID TyValueID
}

type Location struct {
	CodePos int
	StackDepth int
	BrHead bool // true for loops
	PreserveTop bool
	FixupList []FixupInfo
}

type FixupInfo struct {
	CodePos int
	TablePos int
	ValueID TyValueID
}

type Instr struct {
	Target TyValueID // the value id we are assigning to

	Op string
	Immediates []int64
	Values []TyValueID
}

func (c *SSAFunctionCompiler) NextValueID() TyValueID {
	c.ValueID++
	return c.ValueID
}

func (c *SSAFunctionCompiler) PopStack(n int) []TyValueID {
	if len(c.Stack) < n {
		panic("stack underflow")
	}
	ret := make([]TyValueID, n)
	pos := len(c.Stack) - n
	copy(ret, c.Stack[pos:])
	c.Stack = c.Stack[:pos]
	return ret
}

func (c *SSAFunctionCompiler) PushStack(values... TyValueID) {
	c.Stack = append(c.Stack, values...)
}

func (c *SSAFunctionCompiler) FixupLocationRef(loc *Location) {
	if loc.BrHead {
		c.Code = append(c.Code, buildInstr(0, "jmp", []int64{int64(loc.CodePos)}, nil))
		// TODO: Finish lazy fixup of internal branches.
		for _, info := range loc.FixupList {
			c.Code[info.CodePos].Immediates[info.TablePos] = int64(loc.CodePos)
		}
	} else {
		// TODO: Finish lazy fixup of internal branches.
		for _, info := range loc.FixupList {
			c.Code[info.CodePos].Immediates[info.TablePos] = int64(len(c.Code))
		}

		if loc.PreserveTop {
			phiInput := make([]TyValueID, len(loc.FixupList) + 1)
			for i, info := range loc.FixupList {
				if info.ValueID == 0 {
					panic("expected info.ValueID != 0")
				}
				phiInput[i] = info.ValueID
			}

			last := c.PopStack(1)[0]
			phiInput[len(phiInput) - 1] = last

			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, "phi", nil, phiInput))
			c.PushStack(retID)
		}
	}
}

func (c *SSAFunctionCompiler) Compile() {
	c.Locations = append(c.Locations, &Location {
		CodePos: 0,
		StackDepth: 0,
	})

	for _, ins := range c.Source.Code {
		switch ins.Op.Name {
		case "i32.const":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, []int64{int64(ins.Immediates[0].(int32))}, nil))
			c.PushStack(retID)

		case "i32.add":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, nil, c.PopStack(2)))
			c.PushStack(retID)

		case "drop":
			c.PopStack(1)

		case "block":
			c.Locations = append(c.Locations, &Location {
				CodePos: len(c.Code),
				StackDepth: len(c.Stack),
				PreserveTop: ins.Block.Signature != wasm.BlockTypeEmpty,
			})

		case "loop":
			c.Locations = append(c.Locations, &Location {
				CodePos: len(c.Code),
				StackDepth: len(c.Stack),
				BrHead: true,
			})

		case "end":
			loc := c.Locations[len(c.Locations) - 1]
			c.Locations = c.Locations[:len(c.Locations) - 1]
			if (loc.PreserveTop && len(c.Stack) == loc.StackDepth + 1) ||
				(!loc.PreserveTop && len(c.Stack) == loc.StackDepth) {
			} else {
				panic("inconsistent stack pattern")
			}
			c.FixupLocationRef(loc)

		case "br":
			label := int(ins.Immediates[0].(uint32))
			loc := c.Locations[len(c.Locations) - 1 - label]
			fixupInfo := FixupInfo {
				CodePos: len(c.Code),
			}
			if loc.PreserveTop {
				fixupInfo.ValueID = c.Stack[len(c.Stack) - 1]
			}
			loc.FixupList = append(loc.FixupList, fixupInfo)
			c.Code = append(c.Code, buildInstr(0, "jmp", []int64{-1}, nil))

		case "br_if":
			brCondition := c.PopStack(1)
			label := int(ins.Immediates[0].(uint32))
			loc := c.Locations[len(c.Locations) - 1 - label]
			fixupInfo := FixupInfo {
				CodePos: len(c.Code),
			}
			if loc.PreserveTop {
				fixupInfo.ValueID = c.Stack[len(c.Stack) - 1]
			}
			loc.FixupList = append(loc.FixupList, fixupInfo)
			c.Code = append(c.Code, buildInstr(0, "jmp_if", []int64{-1}, brCondition))

		case "br_table":
			brCount := int(ins.Immediates[0].(uint32)) + 1
			brTargets := make([]int64, brCount)
			brCondition := c.PopStack(1)

			for i := 0; i < brCount; i++ {
				label := int(ins.Immediates[i + 1].(uint32))
				loc := c.Locations[len(c.Locations) - 1 - label]

				fixupInfo := FixupInfo {
					CodePos: len(c.Code),
					TablePos: i,
				}
				if loc.PreserveTop {
					fixupInfo.ValueID = c.Stack[len(c.Stack) - 1]
				}
				loc.FixupList = append(loc.FixupList, fixupInfo)
				brTargets[i] = -1
			}
			c.Code = append(c.Code, buildInstr(0, "jmp_table", brTargets, brCondition))

		case "return":
			if len(c.Stack) == 1 {
				c.Code = append(c.Code, buildInstr(0, "return", nil, []TyValueID{c.Stack[0]}))
			} else if len(c.Stack) == 0 {
				c.Code = append(c.Code, buildInstr(0, "return", nil, nil))
			} else {
				panic("incorrect stack state at return")
			}

		default:
			panic(ins.Op.Name)
		}
	}

	c.FixupLocationRef(c.Locations[0])
	c.Code = append(c.Code, buildInstr(0, "return", nil, nil))
}

func buildInstr(target TyValueID, op string, immediates []int64, values []TyValueID) Instr {
	return Instr {
		Target: target,
		Op: op,
		Immediates: immediates,
		Values: values,
	}
}
