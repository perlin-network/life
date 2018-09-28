package compiler

type livenessValueResolver struct {
	ssaFuncCompiler *SSAFunctionCompiler

	nodes  map[blockID]*livenessBasicBlock
	liveIn map[blockID][]TyValueID
}

func newLivenessValueResolver(nodes map[blockID]*livenessBasicBlock, c *SSAFunctionCompiler) *livenessValueResolver {
	return &livenessValueResolver{
		ssaFuncCompiler: c,
		nodes:           nodes,
		liveIn:          make(map[blockID][]TyValueID, 0),
	}
}

// Variable definitions in a block
func (liveness *livenessValueResolver) phiDefs(block BasicBlock) []TyValueID {
	values := make([]TyValueID, 0)
	uses := make(map[TyValueID]bool)

	for _, instr := range block.Code {
		if _, ok := uses[instr.Target]; !ok {
			uses[instr.Target] = true
			values = append(values, instr.Target)
		}
	}

	return values
}

// Variable usage in a block
func (liveness *livenessValueResolver) phiUses(block BasicBlock) []TyValueID {
	values := make([]TyValueID, 0)
	uses := make(map[TyValueID]bool)

	for _, instr := range block.Code {
		for _, value := range instr.Values {
			if _, ok := uses[value]; !ok {
				uses[value] = true
				values = append(values, TyValueID(value))
			}
		}
	}

	return values
}

func (liveness *livenessValueResolver) hasLiveIn(id blockID) bool {
	_, has := liveness.liveIn[id]
	return has
}

func (liveness *livenessValueResolver) getLiveIn(id blockID) []TyValueID {
	return liveness.liveIn[id]
}

func (liveness *livenessValueResolver) setLiveIn(id blockID, values []TyValueID) {
	liveness.liveIn[id] = values
}

func (liveness *livenessValueResolver) GetNumberOfReg() int {
	totalLiveValue := make([]TyValueID, 0)

	for _, blockLiveness := range liveness.liveIn {
		totalLiveValue = setUnion(totalLiveValue, blockLiveness)
	}

	// add one since register start at one, zero being null
	return len(totalLiveValue) + 1
}
