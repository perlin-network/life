package exec

type DynamicModule struct {
}

func (m *DynamicModule) Run(vm *VirtualMachine, ret *int64) int32 {
	panic("not implemented")
}

func CompileDynamicModule(source string) *DynamicModule {
	panic("not implemented")
}
