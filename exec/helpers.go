package exec

import (
	"errors"
	"github.com/perlin-network/life/utils"
)

// Returns an error if any happened during execution of user code.
// Panics on logical errors.
func (vm *VirtualMachine) RunWithGasLimit(entryID, limit int) (int64, error) {
	count := 0

	vm.Ignite(entryID)
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

// Returns an error if any happened during execution of user code.
// Panics on logical errors.
func (vm *VirtualMachine) Run(entryID int) (int64, error) {
	vm.Ignite(entryID)
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
