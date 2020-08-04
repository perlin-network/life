package calls

import (
	"testing"

	"github.com/perlin-network/life/compiler/opcodes"
	"github.com/stretchr/testify/require"
)

func Test_fast_callSumAndAdd1(t *testing.T) {
	vm := newFastVM()
	fn := newCallSumAndAdd1_0()
	res, _ := vm.exec(fn, 3, 4, 0)
	require.Equal(t, int64(3), res)

}

func Benchmark_fast_callSumAndAdd1(b *testing.B) {
	vm := newFastVM()
	fn := newCallSumAndAdd1_0()
	for i := 0; i < b.N; i++ {
		vm.exec(fn, 3, 4, 0)
	}
}

func newCallSumAndAdd1_0() (res *function) {
	fn := function{}
	fn.NumParams = 3
	fn.NumRegs = 3
	fn.NumLocals = 0

	fn.inss = append(fn.inss, ins{valueID: 1, opcode: opcodes.GetLocal, v1: 2, v2: 2})
	fn.inss = append(fn.inss, ins{valueID: 2, opcode: opcodes.I32Const, v1: 1, v2: 1})
	fn.inss = append(fn.inss, ins{valueID: 1, opcode: opcodes.I32GeS, v1: 1, v2: 2})
	fn.inss = append(fn.inss, ins{valueID: 0, opcode: opcodes.JmpIf, v1: 61, v2: 1})
	fn.inss = append(fn.inss, ins{valueID: 0, opcode: opcodes.Jmp, v1: 5, v2: 0})
	fn.inss = append(fn.inss, ins{valueID: 1, opcode: opcodes.GetLocal, v1: 0, v2: 0})
	fn.inss = append(fn.inss, ins{valueID: 0, opcode: opcodes.ReturnValue, v1: 1, v2: 0})
	return &fn
}
