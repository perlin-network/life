package compiler

type CFGraph struct {
	Blocks []BasicBlock
}

type BasicBlock struct {
	Code       []Instr
	JmpKind    TyJmpKind
	JmpTargets []int
	JmpValueID TyValueID
}

type TyJmpKind uint8

const (
	JmpUncond TyJmpKind = iota
	JmpEither
	JmpTable
	JmpReturn
)

func (c *SSAFunctionCompiler) NewCFGraph() *CFGraph {
	g := &CFGraph{}
	insLabels := make(map[int]int)

	insLabels[0] = 0
	nextLabel := 1

	for i, ins := range c.Code {
		switch ins.Op {
		case "jmp", "jmp_if":
			insLabels[int(ins.Immediates[0])] = nextLabel
			nextLabel++

			insLabels[i+1] = nextLabel
			nextLabel++
		case "jmp_table":
			for _, target := range ins.Immediates {
				insLabels[int(target)] = nextLabel
				nextLabel++
			}
			insLabels[i+1] = nextLabel
			nextLabel++
		case "return":
			insLabels[i+1] = nextLabel
			nextLabel++
		}
	}

	g.Blocks = make([]BasicBlock, nextLabel)
	var currentBlock *BasicBlock

	for i, ins := range c.Code {
		if label, ok := insLabels[i]; ok {
			currentBlock = &g.Blocks[label]
		}
		switch ins.Op {
		case "jmp":
			currentBlock.JmpKind = JmpUncond
			currentBlock.JmpTargets = []int{insLabels[int(ins.Immediates[0])]}
			if len(ins.Values) > 0 {
				currentBlock.JmpValueID = ins.Values[0]
			}
			currentBlock = nil
		case "jmp_if":
			currentBlock.JmpKind = JmpEither
			currentBlock.JmpTargets = []int{insLabels[int(i+1)], insLabels[int(ins.Immediates[0])]}
			if len(ins.Values) > 0 {
				currentBlock.JmpValueID = ins.Values[0]
			}
			currentBlock = nil
		case "jmp_table":
			currentBlock.JmpKind = JmpTable
			currentBlock.JmpTargets = make([]int, len(ins.Immediates))
			for j, imm := range ins.Immediates {
				currentBlock.JmpTargets[j] = insLabels[int(imm)]
			}
			if len(ins.Values) > 0 {
				currentBlock.JmpValueID = ins.Values[0]
			}
			currentBlock = nil
		case "return":
			currentBlock.JmpKind = JmpReturn
			if len(ins.Values) > 0 {
				currentBlock.JmpValueID = ins.Values[0]
			}
			currentBlock = nil
		default:
			currentBlock.Code = append(currentBlock.Code, ins)
		}
	}

	return g
}
