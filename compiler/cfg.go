package compiler

type CFGraph struct {
	Blocks []BasicBlock
}

type BasicBlock struct {
	Code       []Instr
	JmpKind    TyJmpKind
	JmpTargets []int

	JmpCond    TyValueID
	YieldValue TyValueID
}

type TyJmpKind uint8

const (
	JmpUndef TyJmpKind = iota
	JmpUncond
	JmpEither
	JmpTable
	JmpReturn
)

func (g *CFGraph) ToInsSeq() []Instr {
	out := make([]Instr, 0)
	blockRelocs := make([]int, len(g.Blocks))
	blockEnds := make([]int, len(g.Blocks))

	for i, bb := range g.Blocks {
		blockRelocs[i] = len(out)
		for _, op := range bb.Code {
			out = append(out, op)
		}
		out = append(out, Instr{}) // jmp placeholder
		blockEnds[i] = len(out)
	}

	for i, bb := range g.Blocks {
		jmpIns := &out[blockEnds[i]-1]
		jmpIns.Immediates = make([]int64, len(bb.JmpTargets))
		for j, target := range bb.JmpTargets {
			jmpIns.Immediates[j] = int64(blockRelocs[target])
		}
		switch bb.JmpKind {
		case JmpUndef:
			panic("got JmpUndef")
		case JmpUncond:
			jmpIns.Op = "jmp"
			jmpIns.Values = []TyValueID{bb.YieldValue}
		case JmpEither:
			jmpIns.Op = "jmp_either"
			jmpIns.Values = []TyValueID{bb.JmpCond, bb.YieldValue}
		case JmpTable:
			jmpIns.Op = "jmp_table"
			jmpIns.Values = []TyValueID{bb.JmpCond, bb.YieldValue}
		case JmpReturn:
			jmpIns.Op = "return"
			if bb.YieldValue != 0 {
				jmpIns.Values = []TyValueID{bb.YieldValue}
			}
		default:
			panic("unreachable")
		}
	}

	return out
}

func (c *SSAFunctionCompiler) NewCFGraph() *CFGraph {
	g := &CFGraph{}
	insLabels := make(map[int]int)

	insLabels[0] = 0
	nextLabel := 1

	for i, ins := range c.Code {
		switch ins.Op {
		case "jmp", "jmp_if", "jmp_either", "jmp_table":
			for _, target := range ins.Immediates {
				if _, ok := insLabels[int(target)]; !ok {
					insLabels[int(target)] = nextLabel
					nextLabel++
				}
			}
			if _, ok := insLabels[i+1]; !ok {
				insLabels[i+1] = nextLabel
				nextLabel++
			}
		case "return":
			if _, ok := insLabels[i+1]; !ok {
				insLabels[i+1] = nextLabel
				nextLabel++
			}
		}
	}

	g.Blocks = make([]BasicBlock, nextLabel)
	var currentBlock *BasicBlock

	for i, ins := range c.Code {
		if label, ok := insLabels[i]; ok {
			if currentBlock != nil {
				currentBlock.JmpKind = JmpUncond
				currentBlock.JmpTargets = []int{label}
			}
			currentBlock = &g.Blocks[label]
		}
		switch ins.Op {
		case "jmp":
			currentBlock.JmpKind = JmpUncond
			currentBlock.JmpTargets = []int{insLabels[int(ins.Immediates[0])]}
			currentBlock.YieldValue = ins.Values[0]
			currentBlock = nil
		case "jmp_if":
			currentBlock.JmpKind = JmpEither
			currentBlock.JmpTargets = []int{insLabels[int(ins.Immediates[0])], insLabels[int(i+1)]}
			currentBlock.JmpCond = ins.Values[0]
			currentBlock.YieldValue = ins.Values[1]
			currentBlock = nil
		case "jmp_either":
			currentBlock.JmpKind = JmpEither
			currentBlock.JmpTargets = []int{insLabels[int(ins.Immediates[0])], insLabels[int(ins.Immediates[1])]}
			currentBlock.JmpCond = ins.Values[0]
			currentBlock.YieldValue = ins.Values[1]
			currentBlock = nil
		case "jmp_table":
			currentBlock.JmpKind = JmpTable
			currentBlock.JmpTargets = make([]int, len(ins.Immediates))
			for j, imm := range ins.Immediates {
				currentBlock.JmpTargets[j] = insLabels[int(imm)]
			}
			currentBlock.JmpCond = ins.Values[0]
			currentBlock.YieldValue = ins.Values[1]
			currentBlock = nil
		case "return":
			currentBlock.JmpKind = JmpReturn
			if len(ins.Values) > 0 {
				currentBlock.YieldValue = ins.Values[0]
			}
			currentBlock = nil
		default:
			currentBlock.Code = append(currentBlock.Code, ins)
		}
	}

	if label, ok := insLabels[len(c.Code)]; ok {
		lastBlock := &g.Blocks[label]
		if lastBlock.JmpKind != JmpUndef {
			panic("last block should always have an undefined jump target")
		}
		lastBlock.JmpKind = JmpReturn
	}

	return g
}
