package exec

import (
	"errors"

	"github.com/perlin-network/life/utils"
)

var _ ImportResolver = (*NopResolver)(nil)

// NopResolver is a nil WebAssembly module import resolver.
type NopResolver struct{}

func (r *NopResolver) ResolveFunc(module, field string) FunctionImport {
	panic("func import not allowed")
}

func (r *NopResolver) ResolveGlobal(module, field string) int64 {
	panic("global import not allowed")
}

// RunWithGasLimit runs a WebAssembly modules function denoted by its ID with a specified set
// of parameters for a specified amount of instructions (also known as gas) denoted by `limit`.
// Panics on logical errors.
func (vm *VirtualMachine) RunWithGasLimit(entryID, limit int, params ...int64) (int64, error) {
	count := 0

	vm.Ignite(entryID, params...)
	for !vm.Exited {
		vm.Execute()
		if vm.Delegate != nil {
			vm.Delegate()
			vm.Delegate = nil
		}
		count++
		if count == limit {
			return -1, errors.New("gas limit exceeded")
		}
	}

	if vm.ExitError != nil {
		return -1, utils.UnifyError(vm.ExitError)
	}
	return vm.ReturnValue, nil
}

// Run runs a WebAssembly modules function denoted by its ID with a specified set
// of parameters.
// Panics on logical errors.
func (vm *VirtualMachine) Run(entryID int, params ...int64) (int64, error) {
	vm.Ignite(entryID, params...)
	for !vm.Exited {
		vm.Execute()
		if vm.Delegate != nil {
			vm.Delegate()
			vm.Delegate = nil
		}
	}

	if vm.ExitError != nil {
		return -1, utils.UnifyError(vm.ExitError)
	}
	return vm.ReturnValue, nil
}
