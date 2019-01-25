package compiler

import "fmt"

type CFGraph struct {
	Blocks []*BasicBlock
}

type BasicBlock struct {
	Code       []Instr
	JmpKind    TyJmpKind
	JmpTargets []int

	JmpCond    TyValueID
	YieldValue TyValueID

	UsedValues map[TyValueID]struct{}
}

type TyJmpKind uint8

const (
	JmpUndef TyJmpKind = iota
	JmpUncond
	JmpEither
	JmpTable
	JmpReturn
)

func (g *CFGraph) doLivenessBFS(blkID int, valueID TyValueID, valueLiveness map[TyValueID]map[int]struct{}) {
	var queue []int

	queue = append(queue, blkID)
	visited := make(map[int]struct{})
	visited[blkID] = struct{}{}

	beginnerCounter := make(map[int]int)

	counter := make(map[int]int)

	maxCounter := 0

	for len(queue) > 0 {
		poppedBlockID := queue[0]
		queue = queue[1:]

		for _, targetBlockID := range g.Blocks[poppedBlockID].JmpTargets {
			if _, seen := visited[targetBlockID]; !seen {
				if _, isBlack := g.Blocks[targetBlockID].UsedValues[valueID]; isBlack {
					counter[targetBlockID] += counter[poppedBlockID] + 1
				} else {
					counter[targetBlockID] += counter[poppedBlockID]
				}

				// Track the first counter value that has begun counting && counter value is not 0.
				if _, begun := beginnerCounter[counter[targetBlockID]]; !begun && counter[targetBlockID] != 0 {
					beginnerCounter[counter[targetBlockID]] = targetBlockID
				}

				// Get the max count
				if maxCounter < counter[targetBlockID] {
					maxCounter = counter[targetBlockID]
				}

				queue = append(queue, targetBlockID)
				visited[targetBlockID] = struct{}{}
			}
		}
	}

	if valueID == 5 {
		fmt.Println(beginnerCounter)
		fmt.Println(counter)
	}

	// Backtrackk!

}

//func (g *CFGraph) doLivenessBFS(blkID int, valueID TyValueID, valueLiveness map[TyValueID]map[int]struct{}) {
//	var queue []int
//
//	queue = append(queue, blkID)
//	visited := make(map[int]struct{}) // already-visited blocks
//
//	greyBlockIDs := make(map[int]struct{})
//	blackBlockIDs := make(map[int]struct{})
//
//	blackBlockIDs[blkID] = struct{}{}
//
//outer:
//	for len(queue) > 0 {
//		poppedBlockID := queue[0]
//
//		block := g.Blocks[poppedBlockID]
//
//		for _, targetBlockID := range block.JmpTargets {
//			if _, seen := visited[targetBlockID]; !seen {
//				if _, colored := g.Blocks[targetBlockID].UsedValues[valueID]; !colored {
//					queue = append(queue, targetBlockID)
//
//					greyBlockIDs[targetBlockID] = struct{}{}
//					visited[targetBlockID] = struct{}{}
//				} else {
//					blackBlockIDs[targetBlockID] = struct{}{}
//				}
//			}
//		}
//
//		queue = queue[1:]
//
//		if len(queue) == 0 && len(blackBlockIDs) > 0 {
//			for blackBlockID := range blackBlockIDs {
//				if _, seen := visited[blackBlockID]; !seen {
//					queue = append(queue, blackBlockID)
//					valueLiveness[valueID][blackBlockID] = struct{}{}
//					visited[blackBlockID] = struct{}{}
//				}
//			}
//
//			for greyBlockID := range greyBlockIDs {
//				valueLiveness[valueID][greyBlockID] = struct{}{}
//			}
//
//			greyBlockIDs = make(map[int]struct{})
//			blackBlockIDs = make(map[int]struct{})
//
//			goto outer
//		}
//	}
//}

func (g *CFGraph) AnalyzeLiveness() (map[TyValueID][]int, TyValueID) {
	valueIDUpperBound := TyValueID(0)

	for _, blk := range g.Blocks {
		blk.UsedValues = make(map[TyValueID]struct{})

		for _, ins := range blk.Code {
			if ins.Target != 0 {
				blk.UsedValues[ins.Target] = struct{}{}
			}
			for _, x := range ins.Values {
				if x != 0 {
					blk.UsedValues[x] = struct{}{}
				}
			}
		}
		if blk.JmpCond != 0 {
			blk.UsedValues[blk.JmpCond] = struct{}{}
		}
		if blk.YieldValue != 0 {
			blk.UsedValues[blk.YieldValue] = struct{}{}
		}

		for id, _ := range blk.UsedValues {
			if id+1 > valueIDUpperBound {
				valueIDUpperBound = id + 1
			}
		}
	}

	out := make(map[TyValueID][]int)
	tmp := make(map[TyValueID]map[int]struct{})

	for i := TyValueID(1); i < valueIDUpperBound; i++ {
		tmp[i] = make(map[int]struct{})
		g.doLivenessBFS(0, i, tmp)
	}

	for id, x := range tmp {
		for k, _ := range x {
			out[id] = append(out[id], k)
		}
	}

	return out, valueIDUpperBound
}

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

func (g *CFGraph) Print() {
	for i, bb := range g.Blocks {
		fmt.Printf("Basic block #%d -> %+v (%d) @ %d\n", i, bb.JmpTargets, bb.YieldValue, bb.JmpCond)
		for _, ins := range bb.Code {
			fmt.Printf("%s %d %+v\n", ins.Op, ins.Target, ins.Values)
		}
	}
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

	g.Blocks = make([]*BasicBlock, nextLabel)
	for i, _ := range g.Blocks {
		g.Blocks[i] = &BasicBlock{}
	}

	var currentBlock *BasicBlock

	for i, ins := range c.Code {
		if label, ok := insLabels[i]; ok {
			if currentBlock != nil {
				currentBlock.JmpKind = JmpUncond
				currentBlock.JmpTargets = []int{label}
			}
			currentBlock = g.Blocks[label]
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
		lastBlock := g.Blocks[label]
		if lastBlock.JmpKind != JmpUndef {
			panic("last block should always have an undefined jump target")
		}
		lastBlock.JmpKind = JmpReturn
	}

	return g
}
