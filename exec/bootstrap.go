package exec

import (
	"errors"
	"sync"
	"github.com/perlin-network/life/bootstrap"
	"log"
	"github.com/perlin-network/life/bootstrap/protos"
	"github.com/golang/protobuf/proto"
)

type Resolver struct {
}

func (r *Resolver) ResolveFunc(module, field string) FunctionImport {
	panic("not implemented")
}

func (r *Resolver) ResolveGlobal(module, field string) int64 {
	panic("not implemented")
}

type Engine struct {
	mu             sync.Mutex
	vm             *VirtualMachine
	funcGetCodeBuf int
	funcCheckAndParse      int
}

func NewEngine() (*Engine, error) {
	vm, exports, err := NewVirtualMachineFromDumpUnsafe(bootstrap.VMBytecode, VMConfig{
		DefaultMemoryPages: 32,
		DefaultTableSize:   128,
	})
	if err != nil {
		return nil, err
	}

	funcGetCodeBuf, ok := exports["get_code_buf"]
	if !ok {
		return nil, errors.New("cannot find get_code_buf")
	}
	funcCheckAndParse, ok := exports["check_and_parse"]
	if !ok {
		return nil, errors.New("cannot find check_and_parse")
	}

	return &Engine{
		vm:             vm,
		funcGetCodeBuf: funcGetCodeBuf,
		funcCheckAndParse:      funcCheckAndParse,
	}, nil
}

/*func (v *Engine) SelfCompileAOTAsync() {
	go func() {
		log.Println("Compiling validator")

		aotSvc := platform.FullAOTCompile(v.vm)
		if aotSvc != nil {
			log.Println("Polymerase enabled for validator.")
			v.mu.Lock()
			v.vm.SetAOTService(aotSvc)
			v.mu.Unlock()
		}
	}()
}*/

func (v *Engine) ValidateWasm(input []byte) error {
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

	_ret, err = v.vm.Run(v.funcCheckAndParse, int64(ret), int64(len(input)))
	ptrAndLen := uint64(_ret)

	if ptrAndLen == 0 {
		return errors.New("validation failed")
	}

	outPtr := uint32(ptrAndLen)
	outLen := uint32(ptrAndLen >> 32)

	log.Printf("outPtr = %d, outLen = %d\n", outPtr, outLen)

	out := v.vm.Memory[int(outPtr) : int(outPtr + outLen)]
	module := protos.ModuleInfo{}

	if err := proto.Unmarshal(out, &module); err != nil {
		return err
	}

	log.Printf("%+v\n", &module)
	return nil
}
