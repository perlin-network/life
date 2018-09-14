// Value liveness analysis & register allocation.
//
// Liveness implementation inspired by https://hal.inria.fr/inria-00558509v2
//
// - A variable a is live-in at node n if it is used at n (a ∈ PhiUses(B)),
// or if there is a path from n to a node that uses a that does not contain a
// re-definition of a.
//
// - A variable a is live-out at node n if it is live-in at one
// of the successors of n.
package compiler

import (
	"github.com/go-interpreter/wagon/wasm"
)

// Set relative complement: S' = A ∖ B
func setDiff(a, b []TyValueID) []TyValueID {
	exclusion := make(map[TyValueID]bool)

	for _, item := range b {
		exclusion[item] = true
	}

	out := make([]TyValueID, 0)

	for _, item := range a {
		if _, isExcluded := exclusion[item]; !isExcluded {
			out = append(out, item)
		}
	}

	return out
}

func isLocalGetAction(instr Instr) bool {
	// TODO(sven): use op constants
	return instr.Op == "get_local"
}

func isLocalSetAction(instr Instr) bool {
	// TODO(sven): use op constants
	return instr.Op == "set_local" || instr.Op == "tee_local"
}

// Set union: S' = A ∪ B
func setUnion(a, b []TyValueID) []TyValueID {
	m := make(map[TyValueID]bool)

	for _, item := range a {
		m[item] = true
	}

	for _, item := range b {
		if _, ok := m[item]; !ok {
			a = append(a, item)
		}
	}

	return a
}

func isLoopEdge(parentBlock BasicBlock, block BasicBlock) bool {
	// FIXME(sven): implement this
	return false
}

// Variable definitions in a block
func (liveness *LivenessProcessor) phiDefs(block BasicBlock) []TyValueID {
	return liveness.locals
}

// Variable usage in a block
func phiUses(block BasicBlock) []TyValueID {
	values := make([]TyValueID, 0)
	uses := make(map[int64]bool)

	for _, instr := range block.Code {
		if isLocalGetAction(instr) {
			for _, value := range instr.Immediates {
				if _, ok := uses[value]; !ok {
					uses[value] = true
					values = append(values, TyValueID(value))
				}
			}
		}
	}

	return values
}

// Identifies a block
type blockID int

// Represents a BasicBlock in our Graph (a node)
type livenessBasicBlock struct {
	id      blockID
	block   BasicBlock
	visited bool
}

type LivenessProcessor struct {
	ssaFuncCompiler *SSAFunctionCompiler
	locals          []TyValueID
	nodes           map[blockID]*livenessBasicBlock
	liveIn          map[blockID][]TyValueID
	unusedLocals    []TyValueID
}

func (liveness *LivenessProcessor) GetUnusedLocals() []int {
	deadInstrIndices := make([]int, 0)
	unusedLocals := make(map[TyValueID]bool)

	for _, local := range liveness.unusedLocals {
		unusedLocals[local] = true
	}

	// used to track provenance upwards of the value
	followTargets := make(map[TyValueID]bool)

	/*
		While from bottom up the instructions and do:

		Let C be an instruction

		If LocalSetAction(C)
			Eliminate the instruction
			Track the provenance in the Values

		If C Target ∈ FollowTargets
			Eliminate the instruction
			Track the provenance in the Values

		TODO(sven): wrong code will be generated for tee_local, we
		need to ensure that the instruction's Target ∉ LiveIn(B)

	*/
	for i := len(liveness.ssaFuncCompiler.Code) - 1; i >= 0; i-- {
		c := liveness.ssaFuncCompiler.Code[i]

		if _, doFollow := followTargets[c.Target]; doFollow {
			delete(followTargets, c.Target)

			deadInstrIndices = append(deadInstrIndices, i)

			// next targets
			for _, next := range c.Values {
				followTargets[next] = true
			}
		}

		if isLocalSetAction(c) {
			index := TyValueID(c.Immediates[0])

			if _, useUnusedLocal := unusedLocals[index]; useUnusedLocal {
				deadInstrIndices = append(deadInstrIndices, i)
				followTargets[c.Values[0]] = true
			}
		}
	}

	return deadInstrIndices
}

// BasicBlock visitor
func (liveness *LivenessProcessor) visitBlock(node *livenessBasicBlock, parentBlock *livenessBasicBlock) {
	block := node.block

	if len(block.Code) == 0 {
		return
	}

	var live []TyValueID
	live = phiUses(block)

	blockPhiDefs := liveness.phiDefs(block)
	blockPhiUses := phiUses(block)

	// UpwardExposed(B) = PhiUses(B) \ PhiDefs(B)
	// upwardExposed := setDiff(blockPhiUses, blockPhiDefs)

	// Unused(B) = PhiDefs(B) \ PhiUses(B)
	unused := setDiff(blockPhiDefs, blockPhiUses)

	// if _, hasBeenProcessed := liveness.liveIn[node.id]; !hasBeenProcessed {
	// 	fmt.Printf(
	// 		"------ %d \nPhiUses(B) = %s, PhiDefs(B) = %s, UpwardExposed(B) = %s, Unused(B) = %s\n",
	// 		node.id, phiUses(block), blockPhiDefs, upwardExposed, unused,
	// 	)

	// 	for _, c := range block.Code {
	// 		if isLocalGetAction(c) || isLocalSetAction(c) {
	// 			fmt.Printf("%s %d\n", c.Op, c.Immediates)
	// 		}
	// 	}

	// 	fmt.Printf("\n")
	// }

	for _, target := range block.JmpTargets {
		// S ∈ successor(B)
		successor := liveness.nodes[blockID(target)]

		if isLoopEdge(block, successor.block) == false {
			liveInSuccessor := liveness.liveIn[successor.id]

			// Live = Live ∪ (LiveIn(S) \ PhiDefs(S))
			live = setUnion(
				live,
				setDiff(liveInSuccessor, liveness.phiDefs(successor.block)),
			)
		}
	}

	// LiveOut(B) = Live
	// liveOut := live

	// LiveIn(B) = Live ∪ PhiDefs(B)
	liveIn := setUnion(live, blockPhiDefs)

	if _, hasBeenProcessed := liveness.liveIn[node.id]; !hasBeenProcessed {
		liveness.liveIn[node.id] = setUnion(liveness.liveIn[node.id], liveIn)
	} else {
		liveness.liveIn[node.id] = liveIn
	}

	liveness.unusedLocals = unused
}

// Process the liveness ranges
// Traverse the CFG in DFS
func (c *SSAFunctionCompiler) NewLiveness(funcLocals []wasm.LocalEntry) *LivenessProcessor {
	cfg := c.NewCFGraph()

	// fmt.Printf("\n------- liveness ----\n")

	nodes := make(map[blockID]*livenessBasicBlock)
	traversalStack := NewLivenessTraversalStack()

	for index, block := range cfg.Blocks {
		id := blockID(index)

		b := &livenessBasicBlock{
			id:      id,
			block:   block,
			visited: false,
		}

		nodes[id] = b
		traversalStack.Push(b)
	}

	locals := make([]TyValueID, 0)

	for _, local := range funcLocals {
		// TODO(sven): ignore type for now, doesn't impact the liveness analysis
		for i := 0; i < int(local.Count); i++ {
			locals = append(locals, TyValueID(i))
		}
	}

	livenessProcessor := &LivenessProcessor{
		ssaFuncCompiler: c,
		locals:          locals,
		nodes:           nodes,
		liveIn:          make(map[blockID][]TyValueID, 0),
	}

	// DFS

	for traversalStack.Len() > 0 {
		node := traversalStack.Pop()

		if node.visited == false {
			node.visited = true

			for _, target := range node.block.JmpTargets {
				successor := livenessProcessor.nodes[blockID(target)]

				if successor == nil {
					panic("edge is pointing to an unknown node")
				}

				if successor.visited == false {
					livenessProcessor.visitBlock(successor, node)
				}
			}
		}
	}

	for i := range cfg.Blocks {
		livenessProcessor.visitBlock(nodes[blockID(i)], nil)
	}

	return livenessProcessor
}
