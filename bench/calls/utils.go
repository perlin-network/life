package calls

import (
	"log"
	"runtime"

	"github.com/perlin-network/life/exec"
	"github.com/perlin-network/life/platform"
	"github.com/stretchr/testify/require"
)

func newVM(t require.TestingT, input []byte, impResolver exec.ImportResolver, aot bool) *exec.VirtualMachine {

	vm, err := exec.NewVirtualMachine(input, exec.VMConfig{}, impResolver, nil)
	require.Nil(t, err)

	if aot {
		aotSvc := platform.FullAOTCompile(vm)
		if nil != aotSvc {
			vm.AOTService = aotSvc
		} else {
			log.Println("WARNNING: AOT is not supported on", runtime.GOOS)
		}
	}
	return vm
}
