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

// Resolver are called in each block to resolve phiUses and phiDefs
type livenessResolver interface {
	phiDefs(block BasicBlock) []TyValueID
	phiUses(block BasicBlock) []TyValueID

	getLiveIn(blockID) []TyValueID
	setLiveIn(blockID, []TyValueID)
	hasLiveIn(blockID) bool
}

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
	/*
		WebAssembly's loops are just a block and a jmp, for our use-case
		we don't need to implement this, since the CFG already handles it.
	*/
	return false
}

type Liveness struct {
	nodes map[blockID]*livenessBasicBlock

	localResolver *livenessLocalResolver
	valueResolver *livenessValueResolver
}

// Process the liveness ranges
// Traverse the CFG in DFS
func (c *SSAFunctionCompiler) NewLiveness(funcLocals []wasm.LocalEntry) *Liveness {
	cfg := c.NewCFGraph()

	nodes := make(map[blockID]*livenessBasicBlock)

	traversalStack := NewLivenessTraversalStack()
	nextTraversalStack := NewLivenessTraversalStack()

	id := 0

	for _, block := range cfg.Blocks {
		b := &livenessBasicBlock{
			id:      blockID(id),
			block:   block,
			visited: false,
		}

		traversalStack.Push(b)
		nodes[blockID(id)] = b

		id++

		for traversalStack.Len() > 0 {
			node := traversalStack.Pop()

			if node.visited == true {
				continue
			}

			nextTraversalStack.Push(node)
			node.visited = true

			for _, edge := range node.block.JmpTargets {
				nextBlock := cfg.Blocks[blockID(edge)]

				b := &livenessBasicBlock{
					id:      blockID(id),
					block:   nextBlock,
					visited: false,
				}

				nextTraversalStack.Push(b)
				nodes[blockID(id)] = b

				id++
			}
		}

		traversalStack.Push(b)
	}

	liveness := &Liveness{
		nodes: nodes,

		localResolver: newLivenessLocalResolver(nodes, c, funcLocals),
		valueResolver: newLivenessValueResolver(nodes, c),
	}

	// DFS
	for nextTraversalStack.Len() > 0 {
		node := nextTraversalStack.Pop()
		liveness.visitBlock(node)
	}

	return liveness
}

// Get Local resolver
func (liveness *Liveness) Local() *livenessLocalResolver {
	return liveness.localResolver
}

// Get Value resolver
func (liveness *Liveness) Value() *livenessValueResolver {
	return liveness.valueResolver
}

// BasicBlock visitor
func (liveness *Liveness) visitBlock(node *livenessBasicBlock) {
	block := node.block

	if len(block.Code) == 0 {
		return
	}

	blockLocalPhiDefs := liveness.localResolver.phiDefs(block)
	blockLocalPhiUses := liveness.localResolver.phiUses(block)

	localLive := blockLocalPhiUses

	blockValuePhiDefs := liveness.valueResolver.phiDefs(block)
	blockValuePhiUses := liveness.valueResolver.phiUses(block)

	valueLive := blockValuePhiUses

	// UpwardExposed(B) = PhiUses(B) \ PhiDefs(B)
	// upwardExposed := setDiff(blockLocalPhiUses, blockLocalPhiDefs)

	/*
		Unused(B) = PhiDefs(B) \ PhiUses(B)
	*/
	localUnused := setDiff(blockLocalPhiDefs, blockLocalPhiUses)
	// ValueUnused := setDiff(blockValuePhiDefs, blockValuePhiUses)

	// if !liveness.localResolver.hasLiveIn(node.id) {
	// 	fmt.Printf(
	// 		"------ %d \nPhiUses(B) = %s, PhiDefs(B) = %s, UpwardExposed(B) = %s, Unused(B) = %s\n",
	// 		node.id, phiUses(block), blockLocalPhiDefs, upwardExposed, unused,
	// 	)

	// 	for _, c := range block.Code {
	// 		fmt.Printf("%s %d\n", c.Op, c.Immediates)
	// 	}

	// 	fmt.Printf("\n")
	// }

	for _, target := range block.JmpTargets {
		// S ∈ successor(B)
		successor := liveness.nodes[blockID(target)]

		if isLoopEdge(block, successor.block) == false {
			localLiveInSuccessor := liveness.localResolver.getLiveIn(successor.id)
			valueLiveInSuccessor := liveness.valueResolver.getLiveIn(successor.id)

			/*
				Live = Live ∪ (LiveIn(S) \ PhiDefs(S))
			*/
			localLive = setUnion(
				localLive,
				setDiff(localLiveInSuccessor, liveness.localResolver.phiDefs(successor.block)),
			)

			valueLive = setUnion(
				localLive,
				setDiff(valueLiveInSuccessor, liveness.valueResolver.phiDefs(successor.block)),
			)
		}
	}

	// LiveOut(B) = Live
	// liveOut := live

	/*
		LiveIn(B) = Live ∪ PhiDefs(B)
	*/
	// for locals
	localLiveIn := setUnion(localLive, blockLocalPhiDefs)

	if liveness.localResolver.hasLiveIn(node.id) {
		liveness.localResolver.setLiveIn(node.id, setUnion(liveness.localResolver.getLiveIn(node.id), localLiveIn))
	} else {
		liveness.localResolver.setLiveIn(node.id, localLiveIn)
	}

	liveness.localResolver.unusedLocals = localUnused

	// for values
	// register needed: LiveIn(B) \ Unused(B)
	valueLiveIn := setUnion(valueLive, blockValuePhiDefs)

	if liveness.valueResolver.hasLiveIn(node.id) {
		liveness.valueResolver.setLiveIn(node.id, setUnion(liveness.valueResolver.getLiveIn(node.id), valueLiveIn))
	} else {
		liveness.valueResolver.setLiveIn(node.id, valueLiveIn)
	}
}
