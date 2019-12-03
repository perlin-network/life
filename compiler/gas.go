package compiler

func (c *SSAFunctionCompiler) InsertGasCounters(gp GasPolicy) {
	cfg := c.NewCFGraph()

	for i := range cfg.Blocks {
		totalCost := int64(1)

		blk := &cfg.Blocks[i]
		for _, ins := range blk.Code {
			totalCost += gp.GetCost(ins)
			if totalCost < 0 {
				panic("total cost overflow")
			}
		}

		if totalCost != 0 {
			blk.Code = append([]Instr{
				buildInstr(0, "add_gas", []int64{totalCost}, []TyValueID{}),
			}, blk.Code...)
		}
	}

	c.Code = cfg.ToInsSeq()
}
