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
	Locations []Location

	ValueID TyValueID
}

type Location struct {
	CodePos int
	StackDepth int
	BrHead bool // true for loops
	PreserveTop bool
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

	pos := len(c.Stack) - n
	ret := c.Stack[pos:]
	c.Stack = c.Stack[:pos]
	return ret
}

func (c *SSAFunctionCompiler) PushStack(values... TyValueID) {
	c.Stack = append(c.Stack, values...)
}

func (c *SSAFunctionCompiler) Compile() {
	c.Locations = append(c.Locations, Location {
		CodePos: 0,
		StackDepth: 0,
	})

	for _, ins := range c.Source.Code {
		switch ins.Op.Name {
		case "i32.add":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, nil, c.PopStack(2)))
			c.PushStack(retID)
		case "block":
			c.Locations = append(c.Locations, Location {
				CodePos: len(c.Code),
				StackDepth: len(c.Stack),
				PreserveTop: ins.Block.Signature != wasm.BlockTypeEmpty,
			})
		case "end":
			loc := c.Locations[len(c.Locations) - 1]
			c.Locations = c.Locations[:len(c.Locations) - 1]
			if (loc.PreserveTop && len(c.Stack) == loc.StackDepth + 1) ||
				(!loc.PreserveTop && len(c.Stack) == loc.StackDepth) {
			} else {
				panic("inconsistent stack pattern")
			}
			if loc.BrHead {
				c.Code = append(c.Code, buildInstr(0, "jmp", []int64{int64(loc.CodePos)}, nil))
				// TODO: Finish lazy fixup of internal branches.
			} else {
				// TODO: Finish lazy fixup of internal branches.
			}
		default:
			panic(ins.Op.Name)
		}
	}
}

func buildInstr(target TyValueID, op string, immediates []int64, values []TyValueID) Instr {
	return Instr {
		Target: target,
		Op: op,
		Immediates: immediates,
		Values: values,
	}
}
