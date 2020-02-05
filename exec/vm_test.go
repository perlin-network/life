package exec

import "testing"

var testCode = []byte{
	0, 97, 115, 109, 1, 0, 0, 0, 1, 14, 3, 96, 2, 127, 127, 0, 96, 0, 0, 96, 2, 127, 127, 0, 2, 12, 1, 3, 109, 111, 100, 4, 116, 101, 115, 116, 0, 2, 3, 2, 1, 1, 5, 3, 1, 0, 17, 7, 17, 2, 6, 109, 101, 109, 111, 114, 121, 2, 0, 4, 109, 97, 105, 110, 0, 1, 10, 11, 1, 9, 0, 65, 0, 65, 247, 0, 16, 0, 11,
}

type MyResolver struct{}

func (mr *MyResolver) ResolveFunc(module, field string) FunctionImport {
	return func(vm *VirtualMachine) int64 {
		panic("panic in order to trigger the VM's termination")
		return 0
	}
}
func (mr *MyResolver) ResolveGlobal(module, field string) int64 {
	panic("not implemented")
}

func TestVMReset(t *testing.T) {
	vm, err := NewVirtualMachine(testCode, VMConfig{}, &MyResolver{}, nil)
	if err != nil {
		t.Fatalf("Error creating VM: %v", err)
	}
	x := func() {
		defer func() {
			if err := recover(); err == nil {
				t.Fatalf("an error should have been triggered")
			}
		}()
		vm.Run(0, 0, 0)
	}
	x()
	vm.Reset()
	y := func() {
		defer func() {
			if err := recover(); err != nil && err != "panic in order to trigger the VM's termination" {
				t.Fatalf("no panic or wrong type: %v", err)
			}
		}()
		vm.Run(0, 1, 0)
	}
	y()
}
