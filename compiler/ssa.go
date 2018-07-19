package compiler

import (
	"fmt"
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

	StackValueSets map[int][]TyValueID
	UsedValueIDs map[TyValueID]struct{}

	ValueID TyValueID
}

type Location struct {
	CodePos int
	StackDepth int
	BrHead bool // true for loops
	PreserveTop bool
	FixupList []FixupInfo

	IfBlock bool
}

type FixupInfo struct {
	CodePos int
	TablePos int
}

type Instr struct {
	Target TyValueID // the value id we are assigning to

	Op string
	Immediates []int64
	Values []TyValueID
}

func NewSSAFunctionCompiler(m *wasm.Module, d *disasm.Disassembly) *SSAFunctionCompiler {
	return &SSAFunctionCompiler {
		Module: m,
		Source: d,
		StackValueSets: make(map[int][]TyValueID),
		UsedValueIDs: make(map[TyValueID]struct{}),
	}
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
	for i, id := range values {
		if _, ok := c.UsedValueIDs[id]; ok {
			panic("pushing a value ID twice is not supported yet")
		}
		c.UsedValueIDs[id] = struct{}{}
		c.StackValueSets[len(c.Stack) + i] = append(c.StackValueSets[len(c.Stack) + i], id)
	}

	c.Stack = append(c.Stack, values...)
}

func (c *SSAFunctionCompiler) FixupLocationRef(loc *Location) {
	if loc.BrHead {
		c.Code = append(c.Code, buildInstr(0, "jmp", []int64{int64(loc.CodePos)}, []TyValueID{0}))
		// TODO: Finish lazy fixup of internal branches.
		for _, info := range loc.FixupList {
			c.Code[info.CodePos].Immediates[info.TablePos] = int64(loc.CodePos)
		}
	} else {
		if loc.PreserveTop {
			// TODO: This might be inefficient.
			c.Code = append(
				c.Code,
				buildInstr(0, "jmp", []int64{int64(len(c.Code) + 1)}, []TyValueID{c.PopStack(1)[0]}),
			)
		}

		// TODO: Finish lazy fixup of internal branches.
		for _, info := range loc.FixupList {
			c.Code[info.CodePos].Immediates[info.TablePos] = int64(len(c.Code))
		}

		if loc.PreserveTop {
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, "phi", nil, nil))
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
		fmt.Printf("%s %d\n", ins.Op.Name, len(c.Stack))
		switch ins.Op.Name {
		case "nop":

		case "unreachable":
			c.Code = append(c.Code, buildInstr(0, ins.Op.Name, nil, nil))

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

		case "get_local":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, []int64{int64(ins.Immediates[0].(uint32))}, nil))
			c.PushStack(retID)

		case "set_local":
			c.Code = append(c.Code, buildInstr(0, ins.Op.Name, []int64{int64(ins.Immediates[0].(uint32))}, c.PopStack(1)))

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

		case "if":
			cond := c.PopStack(1)[0]

			c.Locations = append(c.Locations, &Location {
				CodePos: len(c.Code),
				StackDepth: len(c.Stack),
				PreserveTop: ins.Block.Signature != wasm.BlockTypeEmpty,
				IfBlock: true,
			})

			c.Code = append(c.Code, buildInstr(0, "jmp_if", []int64{int64(len(c.Code) + 2)}, []TyValueID{cond, 0}))
			c.Code = append(c.Code, buildInstr(0, "jmp", []int64{-1}, []TyValueID{0}))

		case "else":
			loc := c.Locations[len(c.Locations) - 1]
			if !loc.IfBlock {
				panic("expected if block")
			}

			loc.FixupList = append(loc.FixupList, FixupInfo {
				CodePos: len(c.Code),
			})

			if loc.PreserveTop {
				c.Code = append(c.Code, buildInstr(0, "jmp", []int64{-1}, c.PopStack(1)))
			} else {
				c.Code = append(c.Code, buildInstr(0, "jmp", []int64{-1}, []TyValueID{0}))
			}

			c.Code[loc.CodePos + 1].Immediates[0] = int64(len(c.Code))
			loc.IfBlock = false

		case "end":
			loc := c.Locations[len(c.Locations) - 1]
			c.Locations = c.Locations[:len(c.Locations) - 1]

			if loc.IfBlock {
				if loc.PreserveTop {
					panic("if block without an else cannot yield values")
				}
				loc.FixupList = append(loc.FixupList, FixupInfo {
					CodePos: loc.CodePos + 1,
				})
			}

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

			brValues := []TyValueID{0}
			if loc.PreserveTop {
				brValues[0] = c.Stack[len(c.Stack) - 1]
			}
			loc.FixupList = append(loc.FixupList, fixupInfo)
			c.Code = append(c.Code, buildInstr(0, "jmp", []int64{-1}, brValues))

		case "br_if":
			brValues := []TyValueID{c.PopStack(1)[0], 0}
			label := int(ins.Immediates[0].(uint32))
			loc := c.Locations[len(c.Locations) - 1 - label]
			fixupInfo := FixupInfo {
				CodePos: len(c.Code),
			}
			if loc.PreserveTop {
				brValues[1] = c.Stack[len(c.Stack) - 1]
			}
			loc.FixupList = append(loc.FixupList, fixupInfo)
			c.Code = append(c.Code, buildInstr(0, "jmp_if", []int64{-1}, brValues))

		case "br_table":
			brCount := int(ins.Immediates[0].(uint32)) + 1
			brTargets := make([]int64, brCount)
			brValues := []TyValueID{c.PopStack(1)[0], 0}

			preserveTop := false

			for i := 0; i < brCount; i++ {
				label := int(ins.Immediates[i + 1].(uint32))
				loc := c.Locations[len(c.Locations) - 1 - label]

				if loc.PreserveTop {
					preserveTop = true
				}

				fixupInfo := FixupInfo {
					CodePos: len(c.Code),
					TablePos: i,
				}
				loc.FixupList = append(loc.FixupList, fixupInfo)
				brTargets[i] = -1
			}

			if preserveTop {
				brValues[1] = c.Stack[len(c.Stack) - 1]
			}

			c.Code = append(c.Code, buildInstr(0, "jmp_table", brTargets, brValues))

		case "return":
			if len(c.Stack) == 1 {
				c.Code = append(c.Code, buildInstr(0, "return", nil, []TyValueID{c.Stack[0]}))
			} else if len(c.Stack) == 0 {
				c.Code = append(c.Code, buildInstr(0, "return", nil, nil))
			} else {
				panic(fmt.Errorf("incorrect stack state at return: depth = %d", len(c.Stack)))
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
