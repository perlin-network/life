package platform

import (
	"github.com/perlin-network/life/exec"
)

type AOTContext struct {
}

func (c *AOTContext) UnsafeInvokeFunction_0(vm *exec.VirtualMachine, name string) uint64 {
	return 0
}

func (c *AOTContext) UnsafeInvokeFunction_1(vm *exec.VirtualMachine, name string, p0 uint64) uint64 {
	return 0
}

func (c *AOTContext) UnsafeInvokeFunction_2(vm *exec.VirtualMachine, name string, p0, p1 uint64) uint64 {
	return 0
}

func FullAOTCompile(vm *exec.VirtualMachine) *AOTContext {
	return nil
}
