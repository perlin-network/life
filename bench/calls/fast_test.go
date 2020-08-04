package calls

import (
	"testing"

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
