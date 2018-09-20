package compiler

import "github.com/go-interpreter/wagon/wasm"

type livenessLocalResolver struct {
	ssaFuncCompiler *SSAFunctionCompiler

	nodes  map[blockID]*livenessBasicBlock
	liveIn map[blockID][]TyValueID

	unusedLocals []TyValueID
	locals       []TyValueID
}

func isLocalGetAction(instr Instr) bool {
	// TODO(sven): use op constants
	return instr.Op == "get_local"
}

func isLocalSetAction(instr Instr) bool {
	// TODO(sven): use op constants
	return instr.Op == "set_local" || instr.Op == "tee_local"
}

func newLivenessLocalResolver(nodes map[blockID]*livenessBasicBlock, c *SSAFunctionCompiler,
	funcLocals []wasm.LocalEntry) *livenessLocalResolver {
	unusedLocals := make([]TyValueID, 0)

	locals := make([]TyValueID, 0)

	for _, local := range funcLocals {
		// TODO(sven): ignore type for now, doesn't impact the liveness analysis
		for i := 0; i < int(local.Count); i++ {
			locals = append(locals, TyValueID(i))
		}
	}

	return &livenessLocalResolver{
		ssaFuncCompiler: c,
		nodes:           nodes,
		locals:          locals,
		unusedLocals:    unusedLocals,
		liveIn:          make(map[blockID][]TyValueID, 0),
	}
}

// Variable definitions in a block
func (liveness *livenessLocalResolver) phiDefs(block BasicBlock) []TyValueID {
	return liveness.locals
}

// Variable usage in a block
func (liveness *livenessLocalResolver) phiUses(block BasicBlock) []TyValueID {
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

func (liveness *livenessLocalResolver) hasLiveIn(id blockID) bool {
	_, has := liveness.liveIn[id]
	return has
}

func (liveness *livenessLocalResolver) getLiveIn(id blockID) []TyValueID {
	return liveness.liveIn[id]
}

func (liveness *livenessLocalResolver) setLiveIn(id blockID, values []TyValueID) {
	liveness.liveIn[id] = values
}

func (liveness *livenessLocalResolver) GetUnused() []int {
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
