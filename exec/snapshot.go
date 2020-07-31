package exec

import "github.com/vmihailenco/msgpack"

func (vm *VirtualMachine) ReadSnapshot() *Snapshot {
	frames := make([]frameSnapshot, len(vm.CallStack))
	for i, f := range vm.CallStack {
		frames[i].FunctionID = f.FunctionID
		frames[i].Regs = f.Regs
		frames[i].Locals = f.Locals
		frames[i].IP = f.IP
		frames[i].ReturnReg = f.ReturnReg
		frames[i].Continuation = f.Continuation
	}
	ss := &stateSnapshot{
		CallStack:     frames,
		CurrentFrame:  vm.CurrentFrame,
		Globals:       vm.Globals,
		NumValueSlots: vm.NumValueSlots,
		Gas:           vm.Gas,
		Yielded:       vm.Yielded,
	}

	b, err := msgpack.Marshal(ss)

	if err != nil {
		panic(err)
	}

	return &Snapshot{
		State:  b,
		Memory: vm.Memory,
	}
}

func (vm *VirtualMachine) WriteSnapshot(ss *Snapshot) error {
	var state stateSnapshot
	err := msgpack.Unmarshal(ss.State, &state)
	if err != nil {
		return err
	}

	vm.CallStack = make([]Frame, len(state.CallStack))
	for i, f := range state.CallStack {
		vm.CallStack[i].FunctionID = f.FunctionID
		vm.CallStack[i].Regs = f.Regs
		vm.CallStack[i].Locals = f.Locals
		vm.CallStack[i].IP = f.IP
		vm.CallStack[i].ReturnReg = f.ReturnReg
		vm.CallStack[i].Continuation = f.Continuation
		vm.CallStack[i].Code = vm.FunctionCode[f.FunctionID].Bytes
	}
	vm.CurrentFrame = state.CurrentFrame
	vm.Globals = state.Globals
	vm.NumValueSlots = state.NumValueSlots
	vm.Gas = state.Gas
	vm.Yielded = state.Yielded

	vm.Memory = ss.Memory

	return nil
}

type stateSnapshot struct {
	CallStack     []frameSnapshot
	CurrentFrame  int
	Globals       []int64
	NumValueSlots int
	Gas           uint64
	Yielded       int64
}

type frameSnapshot struct {
	FunctionID   int
	Regs         []int64
	Locals       []int64
	IP           int
	ReturnReg    int
	Continuation int32
}

type Snapshot struct {
	State  []byte
	Memory []byte
}
