package calls

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"

	"github.com/perlin-network/life/exec"
	"github.com/stretchr/testify/require"
)

func Test_callSumAndAdd1(t *testing.T) {

	input, err := ioutil.ReadFile("sum-add.wasm")
	require.Nil(t, err)

	vm := newVM(t, input, &lifeResolver{}, false)
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

func Benchmark_Ignite(t *testing.B) {

	input, err := ioutil.ReadFile("sum-add.wasm")
	require.Nil(t, err)

	vm := newVM(t, input, &lifeResolver{}, false)
	require.Nil(t, err)

	entryID, ok := vm.GetFunctionExport("callSumAndAdd1")
	require.True(t, ok)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		vm.Ignite(entryID, 3, 4, 10)
		vm.CurrentFrame--
		vm.Exited = true
	}
}

func Benchmark_callSumAndAdd1_0_NoAOT(b *testing.B) {
	callSumAndAdd1(b, 0, false)
}

func Benchmark_callSumAndAdd1_1_NoAOT(b *testing.B) {
	callSumAndAdd1(b, 1, false)
}
func Benchmark_callSumAndAdd1_10_NoAOT(b *testing.B) {
	callSumAndAdd1(b, 10, false)
}

func Benchmark_callSumAndAdd1_0_AOT(b *testing.B) {
	callSumAndAdd1(b, 0, true)
}
func Benchmark_callSumAndAdd1_1_AOT(b *testing.B) {
	callSumAndAdd1(b, 1, true)
}
func Benchmark_callSumAndAdd1_10_AOT(b *testing.B) {
	callSumAndAdd1(b, 10, true)
}

func callSumAndAdd1(t *testing.B, cnt int, aot bool) {
	input, err := ioutil.ReadFile("sum-add.wasm")
	require.Nil(t, err)

	vm := newVM(t, input, &lifeResolver{}, aot)

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

func (r *lifeResolver) ResolveFunc(module, field string) exec.FunctionImport {
	switch module {
	case "env":
		switch field {
		case "sum":
			log.Println("Resolver called")
			return func(vm *exec.VirtualMachine) int64 {
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
