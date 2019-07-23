package wasm_validation

import (
	"errors"
	"github.com/perlin-network/life/exec"
	"sync"
)

type Resolver struct {
}

func (r *Resolver) ResolveFunc(module, field string) exec.FunctionImport {
	panic("not implemented")
}

func (r *Resolver) ResolveGlobal(module, field string) int64 {
	panic("not implemented")
}

type Validator struct {
	mu             sync.Mutex
	vm             *exec.VirtualMachine
	funcGetCodeBuf int
	funcCheck      int
}

var globalValidator *Validator
var globalValidatorErr error
var globalValidatorInit sync.Once

func NewValidator() (*Validator, error) {
	vm, err := exec.NewVirtualMachine(ValidatorCode, exec.VMConfig{
		DefaultMemoryPages: 32,
		DefaultTableSize:   128,
	}, new(Resolver), nil)

	if err != nil {
		return nil, err
	}

	funcGetCodeBuf, ok := vm.GetFunctionExport("get_code_buf")
	if !ok {
		return nil, errors.New("cannot find get_code_buf")
	}
	funcCheck, ok := vm.GetFunctionExport("check")
	if !ok {
		return nil, errors.New("cannot find check")
	}

	return &Validator{
		vm:             vm,
		funcGetCodeBuf: funcGetCodeBuf,
		funcCheck:      funcCheck,
	}, nil
}

func (v *Validator) ValidateWasm(input []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	_ret, err := v.vm.Run(v.funcGetCodeBuf, int64(len(input)))
	if err != nil {
		return err
	}
	ret := uint32(_ret)
	if ret == 0 {
		return errors.New("input too large")
	}
	copy(v.vm.Memory[int(ret):], input)
	_ret, err = v.vm.Run(v.funcCheck, int64(ret), int64(len(input)))
	ret = uint32(_ret)

	if ret == 0 {
		return errors.New("validation failed")
	} else if ret == 1 {
		return nil
	} else {
		return errors.New("unknown return value")
	}
}

func GetValidator() *Validator {
	globalValidatorInit.Do(func() {
		globalValidator, globalValidatorErr = NewValidator()
	})

	if globalValidatorErr != nil {
		panic(globalValidatorErr) // "poisoning"
	}

	return globalValidator
}
