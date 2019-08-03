package compiler

type GasPolicy interface {
	GetCost(key Instr) int64
}

type SimpleGasPolicy struct {
	GasPerInstruction int64
}

func (p *SimpleGasPolicy) GetCost(key Instr) int64 {
	return p.GasPerInstruction
}
