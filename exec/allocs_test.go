package exec

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_callSumAndAdd1(t *testing.T) {

	input, err := ioutil.ReadFile("sum-add.wasm")
	require.Nil(t, err)

	vm, err := NewVirtualMachine(input, VMConfig{}, &lifeResolver{}, nil)
	require.Nil(t, err)

	entryID, ok := vm.GetFunctionExport("callSumAndAdd1")
	require.True(t, ok)

	ret, err := vm.Run(entryID, 3, 4, 0)
	require.Equal(t, int64(3), ret)

	ret, err = vm.Run(entryID, 3, 4, 1)
	require.Nil(t, err)
	require.Equal(t, int64(8), ret)

	ret, err = vm.Run(entryID, 3, 4, 10)
	require.Nil(t, err)
	require.Equal(t, int64(53), ret)

}

func Benchmark_life_callSumAndAdd1_10_NoAOT_DoNotKeepFrameValues(b *testing.B) {
	callSumAndAdd1(b, 10, false)
}

func Benchmark_life_callSumAndAdd1_10_NoAOT_KeepFrameValues(b *testing.B) {
	callSumAndAdd1(b, 10, true)
}

func Benchmark_life_callSumAndAdd1_1_NoAOT_KeepFrameValues(b *testing.B) {
	callSumAndAdd1(b, 1, true)
}

func callSumAndAdd1(t *testing.B, cnt int, keepFrameValues bool) {
	input, err := ioutil.ReadFile("sum-add.wasm")
	require.Nil(t, err)

	vm, err := NewVirtualMachine(input, VMConfig{}, &lifeResolver{}, nil)
	vm.KeepFrameValues = keepFrameValues
	require.Nil(t, err)

	entryID, ok := vm.GetFunctionExport("callSumAndAdd1")
	require.True(t, ok)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		_, err := vm.Run(entryID, 3, 4, int64(cnt))
		if nil != err {
			panic(err)
		}
	}
}

type lifeResolver struct{}

func (r *lifeResolver) ResolveFunc(module, field string) FunctionImport {
	switch module {
	case "env":
		switch field {
		case "sum":
			return func(vm *VirtualMachine) int64 {
				v1 := int32(vm.GetCurrentFrame().Locals[0])
				v2 := int32(vm.GetCurrentFrame().Locals[1])
				return int64(v1 + v2)
			}
		default:
			panic(fmt.Errorf("unknown import resolved: %s", field))
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
}

func (r *lifeResolver) ResolveGlobal(module, field string) int64 {
	panic("we're not resolving global variables for now")
}
