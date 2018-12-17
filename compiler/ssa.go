package compiler

import (
	"fmt"

	"math"

	"github.com/go-interpreter/wagon/disasm"
	"github.com/go-interpreter/wagon/wasm"
	"strings"
)

type TyValueID uint64

// SSAFunctionCompiler represents a compiler which translates a WebAssembly modules
// intepreted code into a Static-Single-Assignment-based intermediate representation.
type SSAFunctionCompiler struct {
	Module *wasm.Module
	Source *disasm.Disassembly

	Code      []Instr
	Stack     []TyValueID
	Locations []*Location

	CallIndexOffset int

	StackValueSets map[int][]TyValueID
	UsedValueIDs   map[TyValueID]struct{}

	ValueID TyValueID
}

type Location struct {
	CodePos         int
	StackDepth      int
	BrHead          bool // true for loops
	PreserveTop     bool
	LoopPreserveTop bool
	FixupList       []FixupInfo

	IfBlock bool
}

type FixupInfo struct {
	CodePos  int
	TablePos int
}

// Instr denotes a single instrution.
type Instr struct {
	Target TyValueID // the value id we are assigning to

	Op         string
	Immediates []int64
	Values     []TyValueID
}

// NewSSAFunctionCompiler instantiates a compiler which translates a WebAssembly modules
// intepreted code into a Static-Single-Assignment-based intermediate representation.
func NewSSAFunctionCompiler(m *wasm.Module, d *disasm.Disassembly) *SSAFunctionCompiler {
	return &SSAFunctionCompiler{
		Module:         m,
		Source:         d,
		StackValueSets: make(map[int][]TyValueID),
		UsedValueIDs:   make(map[TyValueID]struct{}),
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

func (c *SSAFunctionCompiler) PushStack(values ...TyValueID) {
	for i, id := range values {
		if _, ok := c.UsedValueIDs[id]; ok {
			panic("pushing a value ID twice is not supported yet")
		}
		c.UsedValueIDs[id] = struct{}{}
		c.StackValueSets[len(c.Stack)+i] = append(c.StackValueSets[len(c.Stack)+i], id)
	}

	c.Stack = append(c.Stack, values...)
}

func (c *SSAFunctionCompiler) FixupLocationRef(loc *Location, wasUnreachable bool) {
	if loc.PreserveTop || loc.LoopPreserveTop {
		if wasUnreachable {
			c.Code = append(
				c.Code,
				buildInstr(0, "jmp", []int64{int64(len(c.Code) + 1)}, []TyValueID{0}),
			)
		} else {
			c.Code = append(
				c.Code,
				buildInstr(0, "jmp", []int64{int64(len(c.Code) + 1)}, c.PopStack(1)),
			)
		}
	}

	var innerBrTarget int64
	if loc.BrHead {
		innerBrTarget = int64(loc.CodePos)
	} else {
		innerBrTarget = int64(len(c.Code))
	}

	for _, info := range loc.FixupList {
		c.Code[info.CodePos].Immediates[info.TablePos] = innerBrTarget
	}

	if loc.PreserveTop || loc.LoopPreserveTop /* why? */ {
		retID := c.NextValueID()
		c.Code = append(c.Code, buildInstr(retID, "phi", nil, nil))
		c.PushStack(retID)
	}
}

func (c *SSAFunctionCompiler) FilterFloatingPoint() {
	for i, ins := range c.Code {
		if strings.HasPrefix(ins.Op, "f32.") || strings.HasPrefix(ins.Op, "f64.") ||
			strings.HasSuffix(ins.Op, "/f32") || strings.HasSuffix(ins.Op, "/f64") {
			if strings.Contains(ins.Op, ".reinterpret/") || strings.HasSuffix(ins.Op, ".const") {
				continue // whitelist
			}
			c.Code[i] = buildInstr(ins.Target, "fp_disabled_error", nil, nil)
		}
	}
}

// Compile compiles an interpreted WebAssembly modules source code into
// a Static-Single-Assignment-based intermediate representation.
func (c *SSAFunctionCompiler) Compile(importTypeIDs []int) {
	c.Locations = append(c.Locations, &Location{
		CodePos:    0,
		StackDepth: 0,
	})

	unreachableDepth := 0

	for _, ins := range c.Source.Code {
		//fmt.Printf("%s %d\n", ins.Op.Name, len(c.Stack))
		wasUnreachable := false

		if unreachableDepth != 0 {
			wasUnreachable = true
			switch ins.Op.Name {
			case "block", "loop", "if":
				unreachableDepth++
			case "end":
				unreachableDepth--
			}
			if unreachableDepth == 1 && ins.Op.Name == "else" {
				unreachableDepth--
			}
			if unreachableDepth != 0 {
				continue
			}
		}

		switch ins.Op.Name {
		case "nop":

		case "unreachable":
			c.Code = append(c.Code, buildInstr(0, ins.Op.Name, nil, nil))
			unreachableDepth = 1

		case "select":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, nil, c.PopStack(3)))
			c.PushStack(retID)

		case "i32.const":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, []int64{int64(ins.Immediates[0].(int32))}, nil))
			c.PushStack(retID)

		case "i64.const":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, []int64{ins.Immediates[0].(int64)}, nil))
			c.PushStack(retID)

		case "f32.const":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, []int64{int64(math.Float32bits(ins.Immediates[0].(float32)))}, nil))
			c.PushStack(retID)

		case "f64.const":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, []int64{int64(math.Float64bits(ins.Immediates[0].(float64)))}, nil))
			c.PushStack(retID)

		case "i32.add", "i32.sub", "i32.mul", "i32.div_s", "i32.div_u", "i32.rem_s", "i32.rem_u", "i32.and", "i32.or", "i32.xor", "i32.shl", "i32.shr_s", "i32.shr_u", "i32.rotl", "i32.rotr",
			"i32.eq", "i32.ne", "i32.lt_s", "i32.lt_u", "i32.le_s", "i32.le_u", "i32.gt_s", "i32.gt_u" /*"i32.ge_s", */, "i32.ge_u",
			"i64.add", "i64.sub", "i64.mul", "i64.div_s", "i64.div_u", "i64.rem_s", "i64.rem_u", "i64.and", "i64.or", "i64.xor", "i64.shl", "i64.shr_s", "i64.shr_u", "i64.rotl", "i64.rotr",
			"i64.eq", "i64.ne", "i64.lt_s", "i64.lt_u", "i64.le_s", "i64.le_u", "i64.gt_s", "i64.gt_u", "i64.ge_s", "i64.ge_u",
			"f32.add", "f32.sub", "f32.mul", "f32.div", "f32.min", "f32.max", "f32.copysign",
			"f32.eq", "f32.ne", "f32.lt", "f32.le", "f32.gt", "f32.ge",
			"f64.add", "f64.sub", "f64.mul", "f64.div", "f64.min", "f64.max", "f64.copysign",
			"f64.eq", "f64.ne", "f64.lt", "f64.le", "f64.gt", "f64.ge":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, nil, c.PopStack(2)))
			c.PushStack(retID)

		case "i32.ge_s":
			retID := c.NextValueID()
			if ins.Op.Code == 0x4e { // real ge_s
				c.Code = append(c.Code, buildInstr(retID, "i32.ge_s", nil, c.PopStack(2)))
			} else if ins.Op.Code == 0x4f { // the wagon ge_s
				c.Code = append(c.Code, buildInstr(retID, "i32.ge_u", nil, c.PopStack(2)))
			} else {
				panic("unreachable")
			}
			c.PushStack(retID)
		case "i32.clz", "i32.ctz", "i32.popcnt", "i32.eqz",
			"i64.clz", "i64.ctz", "i64.popcnt", "i64.eqz",
			"f32.sqrt", "f32.ceil", "f32.floor", "f32.trunc", "f32.nearest", "f32.abs", "f32.neg",
			"f64.sqrt", "f64.ceil", "f64.floor", "f64.trunc", "f64.nearest", "f64.abs", "f64.neg",
			"i32.wrap/i64", "i64.extend_u/i32", "i64.extend_s/i32",
			"i32.trunc_u/f32", "i32.trunc_u/f64", "i64.trunc_u/f32", "i64.trunc_u/f64",
			"i32.trunc_s/f32", "i32.trunc_s/f64", "i64.trunc_s/f32", "i64.trunc_s/f64",
			"f32.demote/f64", "f64.promote/f32",
			"f32.convert_u/i32", "f32.convert_u/i64", "f64.convert_u/i32", "f64.convert_u/i64",
			"f32.convert_s/i32", "f32.convert_s/i64", "f64.convert_s/i32", "f64.convert_s/i64",
			"i32.reinterpret/f32", "i64.reinterpret/f64",
			"f32.reinterpret/i32", "f64.reinterpret/i64":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, nil, c.PopStack(1)))
			c.PushStack(retID)
		case "drop":
			c.PopStack(1)

		case "i32.load", "i64.load", "i32.load8_s", "i32.load16_s", "i64.load8_s", "i64.load16_s", "i64.load32_s",
			"i32.load8_u", "i32.load16_u", "i64.load8_u", "i64.load16_u", "i64.load32_u",
			"f32.load", "f64.load":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, []int64{int64(ins.Immediates[0].(uint32)),
				int64(ins.Immediates[1].(uint32))}, c.PopStack(1)))
			c.PushStack(retID)
		case "i32.store", "i32.store8", "i32.store16", "i64.store", "i64.store8", "i64.store16", "i64.store32", "f32.store", "f64.store":
			c.Code = append(c.Code, buildInstr(0, ins.Op.Name, []int64{int64(ins.Immediates[0].(uint32)),
				int64(ins.Immediates[1].(uint32))}, c.PopStack(2)))

		case "get_local", "get_global":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, []int64{int64(ins.Immediates[0].(uint32))}, nil))
			c.PushStack(retID)

		case "set_local", "set_global":
			c.Code = append(c.Code, buildInstr(0, ins.Op.Name, []int64{int64(ins.Immediates[0].(uint32))}, c.PopStack(1)))

		case "tee_local":
			c.Code = append(c.Code, buildInstr(0, "set_local", []int64{int64(ins.Immediates[0].(uint32))}, []TyValueID{c.Stack[len(c.Stack)-1]}))

		case "block":
			c.Locations = append(c.Locations, &Location{
				CodePos:     len(c.Code),
				StackDepth:  len(c.Stack),
				PreserveTop: ins.Block.Signature != wasm.BlockTypeEmpty,
			})

		case "loop":
			c.Locations = append(c.Locations, &Location{
				CodePos:         len(c.Code),
				StackDepth:      len(c.Stack),
				LoopPreserveTop: ins.Block.Signature != wasm.BlockTypeEmpty,
				BrHead:          true,
			})

		case "if":
			cond := c.PopStack(1)[0]

			c.Locations = append(c.Locations, &Location{
				CodePos:     len(c.Code),
				StackDepth:  len(c.Stack),
				PreserveTop: ins.Block.Signature != wasm.BlockTypeEmpty,
				IfBlock:     true,
			})

			c.Code = append(c.Code, buildInstr(0, "jmp_if", []int64{int64(len(c.Code) + 2)}, []TyValueID{cond, 0}))
			c.Code = append(c.Code, buildInstr(0, "jmp", []int64{-1}, []TyValueID{0}))

		case "else":
			loc := c.Locations[len(c.Locations)-1]
			if !loc.IfBlock {
				panic("expected if block")
			}

			loc.FixupList = append(loc.FixupList, FixupInfo{
				CodePos: len(c.Code),
			})

			if loc.PreserveTop {
				if !wasUnreachable {
					c.Code = append(c.Code, buildInstr(0, "jmp", []int64{-1}, c.PopStack(1)))
				} else {
					c.Code = append(c.Code, buildInstr(0, "jmp", []int64{-1}, []TyValueID{0}))
				}
			} else {
				c.Code = append(c.Code, buildInstr(0, "jmp", []int64{-1}, []TyValueID{0}))
			}
			if wasUnreachable {
				c.Stack = c.Stack[:loc.StackDepth] // unwind stack
			}

			c.Code[loc.CodePos+1].Immediates[0] = int64(len(c.Code))
			loc.IfBlock = false

		case "end":
			loc := c.Locations[len(c.Locations)-1]
			c.Locations = c.Locations[:len(c.Locations)-1]

			if loc.IfBlock {
				if loc.PreserveTop {
					panic("if block without an else cannot yield values")
				}
				loc.FixupList = append(loc.FixupList, FixupInfo{
					CodePos: loc.CodePos + 1,
				})
			}

			if !wasUnreachable {
				if ((loc.PreserveTop || loc.LoopPreserveTop) && len(c.Stack) == loc.StackDepth+1) ||
					(!(loc.PreserveTop || loc.LoopPreserveTop) && len(c.Stack) == loc.StackDepth) {
				} else {
					panic(fmt.Errorf("inconsistent stack pattern: pt = %v, lpt = %v, ls = %d, sd = %d", loc.PreserveTop, loc.LoopPreserveTop, len(c.Stack), loc.StackDepth))
				}
			} else {
				c.Stack = c.Stack[:loc.StackDepth]
			}
			c.FixupLocationRef(loc, wasUnreachable)

		case "br":
			label := int(ins.Immediates[0].(uint32))
			loc := c.Locations[len(c.Locations)-1-label]
			fixupInfo := FixupInfo{
				CodePos: len(c.Code),
			}

			brValues := []TyValueID{0}
			if loc.PreserveTop {
				brValues[0] = c.Stack[len(c.Stack)-1]
			}
			loc.FixupList = append(loc.FixupList, fixupInfo)
			c.Code = append(c.Code, buildInstr(0, "jmp", []int64{-1}, brValues))
			unreachableDepth = 1

		case "br_if":
			brValues := []TyValueID{c.PopStack(1)[0], 0}
			label := int(ins.Immediates[0].(uint32))
			loc := c.Locations[len(c.Locations)-1-label]
			fixupInfo := FixupInfo{
				CodePos: len(c.Code),
			}
			if loc.PreserveTop {
				brValues[1] = c.Stack[len(c.Stack)-1]
			}
			loc.FixupList = append(loc.FixupList, fixupInfo)
			c.Code = append(c.Code, buildInstr(0, "jmp_if", []int64{-1}, brValues))

		case "br_table":
			brCount := int(ins.Immediates[0].(uint32)) + 1
			brTargets := make([]int64, brCount)
			brValues := []TyValueID{c.PopStack(1)[0], 0}

			preserveTop := false

			for i := 0; i < brCount; i++ {
				label := int(ins.Immediates[i+1].(uint32))
				loc := c.Locations[len(c.Locations)-1-label]

				if loc.PreserveTop {
					preserveTop = true
				}

				fixupInfo := FixupInfo{
					CodePos:  len(c.Code),
					TablePos: i,
				}
				loc.FixupList = append(loc.FixupList, fixupInfo)
				brTargets[i] = -1
			}

			if preserveTop {
				brValues[1] = c.Stack[len(c.Stack)-1]
			}

			c.Code = append(c.Code, buildInstr(0, "jmp_table", brTargets, brValues))
			unreachableDepth = 1

		case "return":
			if len(c.Stack) != 0 {
				c.Code = append(c.Code, buildInstr(0, "return", nil, c.PopStack(1)))
			} else {
				c.Code = append(c.Code, buildInstr(0, "return", nil, nil))
			}
			unreachableDepth = 1

		case "call":
			targetID := int(ins.Immediates[0].(uint32))
			var targetSig *wasm.FunctionSig

			if targetID-c.CallIndexOffset >= 0 { // virtual function
				targetSig = c.Module.FunctionIndexSpace[targetID-c.CallIndexOffset].Sig
			} else { // import function
				tyID := importTypeIDs[targetID]
				targetSig = &c.Module.Types.Entries[tyID]
			}

			params := c.PopStack(len(targetSig.ParamTypes))
			targetValueID := TyValueID(0)
			if len(targetSig.ReturnTypes) > 0 {
				targetValueID = c.NextValueID()
			}
			c.Code = append(c.Code, buildInstr(targetValueID, "call", []int64{int64(targetID)}, params))
			if targetValueID != 0 {
				c.PushStack(targetValueID)
			}

		case "call_indirect":
			typeID := int(ins.Immediates[0].(uint32))
			sig := &c.Module.Types.Entries[typeID]

			targetWithParams := c.PopStack(len(sig.ParamTypes) + 1)
			targetValueID := TyValueID(0)
			if len(sig.ReturnTypes) > 0 {
				targetValueID = c.NextValueID()
			}
			c.Code = append(c.Code, buildInstr(targetValueID, "call_indirect", []int64{int64(typeID)}, targetWithParams))
			if targetValueID != 0 {
				c.PushStack(targetValueID)
			}

		case "current_memory":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, nil, nil))
			c.PushStack(retID)

		case "grow_memory":
			retID := c.NextValueID()
			c.Code = append(c.Code, buildInstr(retID, ins.Op.Name, nil, c.PopStack(1)))
			c.PushStack(retID)

		default:
			panic(ins.Op.Name)
		}
	}

	c.FixupLocationRef(c.Locations[0], false)
	if len(c.Stack) != 0 {
		c.Code = append(c.Code, buildInstr(0, "return", nil, c.PopStack(1)))
	} else {
		c.Code = append(c.Code, buildInstr(0, "return", nil, nil))
	}
}

func buildInstr(target TyValueID, op string, immediates []int64, values []TyValueID) Instr {
	return Instr{
		Target:     target,
		Op:         op,
		Immediates: immediates,
		Values:     values,
	}
}
