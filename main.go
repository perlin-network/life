package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"

	"encoding/binary"

	"github.com/perlin-network/life/exec"
)

const (
	// [0, 1024]: Static memory (globals).
	// [1024, 3744]: Stack.
	// [3744, 3760]: Pointer to end of dynamic memory.
	// [3760, 5246640]: Dynamic memory allocated via. sbrk.
	GLOBAL_BASE = 1024

	TOTAL_STACK = 5242880

	STATIC_BASE = GLOBAL_BASE
	STATIC_BUMP = 2704

	tempDoublePtr = STATIC_BASE + STATIC_BUMP
	STATICTOP     = STATIC_BASE + STATIC_BUMP + 16

	// Have 4 bytes wedged between the end of static memory, and the stack
	// to declare a pointer to the end of the dynamic memory.
	STACKTOP   = (STATICTOP + 4 + 15) & -16
	STACK_BASE = (STATICTOP + 4 + 15) & -16

	STACK_MAX = STACK_BASE + TOTAL_STACK

	DYNAMIC_BASE = STACK_MAX
)

var (
	DYNAMICTOP_PTR = STATICTOP
)

type Exception struct {
	ptr        uint32
	adjusted   uint32
	typ        int
	destructor int
	refcount   int
	caught     bool
	rethrown   bool
}

type Resolver struct {
	Exceptions ExceptionsManager
	tempRet0   int64
}

type ExceptionsManager struct {
	Last           uint32
	Uncaught       int
	Caught         []uint32
	Infos          map[uint32]Exception
	CatchBufferPtr uint32
}

func (e *ExceptionsManager) deAdjust(adjusted uint32) uint32 {
	if _, exists := e.Infos[adjusted]; exists {
		return adjusted
	}

	for ptr, info := range e.Infos {
		if info.adjusted == adjusted {
			return ptr
		}
	}

	return adjusted
}

func (e *ExceptionsManager) addRef(ptr uint32) {
	if info, exists := e.Infos[ptr]; exists {
		info.refcount++
	}
}

func (e *ExceptionsManager) decRef(ptr uint32) {
	if info, exists := e.Infos[ptr]; exists {
		if info.refcount <= 0 {
			panic("refcount for info <= 0")
		}
		info.refcount--
	}
}

func (e *ExceptionsManager) clearRef(ptr uint32) {
	if info, exists := e.Infos[ptr]; exists {
		info.refcount = 0
	}
}

func NewResolver() *Resolver {
	return &Resolver{
		Exceptions: ExceptionsManager{
			Last:           0,
			Caught:         []uint32{},
			Infos:          make(map[uint32]Exception),
			CatchBufferPtr: 0,
		},
	}
}

// malloc allocates `n` bytes of dynamic memory.
func (r *Resolver) malloc(vm *exec.VirtualMachine, n int) int64 {
	current := len(vm.Memory)
	if vm.Config.MaxMemoryPages == 0 || (current+n >= current && current+n <= vm.Config.MaxMemoryPages*exec.DefaultPageSize) {
		vm.Memory = append(vm.Memory, make([]byte, n)...)
		return int64(uint32(current))
	} else {
		return -1
	}
}

func (r *Resolver) ResolveFunc(module, field string) exec.FunctionImport {
	fmt.Printf("Resolve func: %s %s\n", module, field)
	if module != "env" {
		panic("module != env")
	}

	switch field {
	case "__life_ping":
		return func(vm *exec.VirtualMachine) int64 {
			return vm.GetCurrentFrame().Locals[0] + 1
		}
	case "enlargeMemory":
		return func(vm *exec.VirtualMachine) int64 {
			panic("enlargeMemory not implemented")
		}

	case "getTotalMemory":
		return func(vm *exec.VirtualMachine) int64 {
			return int64(len(vm.Memory))
		}

	case "abortOnCannotGrowMemory":
		return func(vm *exec.VirtualMachine) int64 {
			panic("Cannot grow memory")
		}

	case "abortStackOverflow":
		return func(vm *exec.VirtualMachine) int64 {
			panic("Emscripten stack overflow")
		}

	case "___lock", "___unlock", "___gxx_personality_v0":
		return func(vm *exec.VirtualMachine) int64 {
			return 0
		}

	case "___setErrNo":
		return func(vm *exec.VirtualMachine) int64 {
			panic("setErrNo not implemented")
		}

	case "_emscripten_memcpy_big":
		return func(vm *exec.VirtualMachine) int64 {
			frame := vm.GetCurrentFrame()
			dest := int(frame.Locals[0])
			src := int(frame.Locals[1])
			num := int(frame.Locals[2])
			copy(vm.Memory[dest:], vm.Memory[src:src+num])
			return int64(dest)
		}

	case "_llvm_eh_typeid_for":
		return func(vm *exec.VirtualMachine) int64 {
			return vm.GetCurrentFrame().Locals[0]
		}

	case "abort":
		return func(vm *exec.VirtualMachine) int64 {
			panic("Emscripten abort")
		}

	case "setTempRet0":
		return func(vm *exec.VirtualMachine) int64 {
			r.tempRet0 = vm.GetCurrentFrame().Locals[0]
			return 0
		}

	case "getTempRet0":
		return func(vm *exec.VirtualMachine) int64 {
			return r.tempRet0
		}

	case "___cxa_allocate_exception":
		return func(vm *exec.VirtualMachine) int64 {
			return r.malloc(vm, int(vm.GetCurrentFrame().Locals[0]))
		}

	case "___cxa_free_exception":
		return func(vm *exec.VirtualMachine) int64 {
			// TODO: Free memory denoted by pointer vm.GetCurrentFrame().Locals[0].
			return 0
		}

	case "___cxa_begin_catch":
		return func(vm *exec.VirtualMachine) int64 {
			frame := vm.GetCurrentFrame()
			ptr := uint32(frame.Locals[0])

			if info, exists := r.Exceptions.Infos[ptr]; exists {
				if !info.caught {
					info.caught = true
					r.Exceptions.Uncaught--
				} else {
					info.rethrown = false
				}
			}

			r.Exceptions.Caught = append(r.Exceptions.Caught, ptr)
			r.Exceptions.addRef(r.Exceptions.deAdjust(ptr))

			return int64(ptr)
		}

	case "___cxa_end_catch":
		return func(vm *exec.VirtualMachine) int64 {
			if len(r.Exceptions.Caught) > 0 {
				// Pop pointer to info about most recently caught exception.
				ptr := r.Exceptions.Caught[len(r.Exceptions.Caught)-1]
				r.Exceptions.Caught = r.Exceptions.Caught[:len(r.Exceptions.Caught)-2]

				r.Exceptions.decRef(r.Exceptions.deAdjust(ptr))
				r.Exceptions.Last = 0

			}

			return 0
		}

	case "___cxa_find_matching_catch_3":
		return func(vm *exec.VirtualMachine) int64 {
			thrown := r.Exceptions.Last
			if thrown == 0 {
				// Return nil pointer.
				r.tempRet0 = 0
				return r.tempRet0
			}

			info, exists := r.Exceptions.Infos[thrown]
			if !exists || info.typ == 0 {
				r.tempRet0 = int64(thrown)
				return r.tempRet0
			}

			// Initialize 32-bit pointer cache if not exist.
			if r.Exceptions.CatchBufferPtr == 0 {
				ptr := r.malloc(vm, 4)
				if ptr == -1 {
					panic("unable to allocate pointer cache")
				}

				r.Exceptions.CatchBufferPtr = uint32(ptr)
			}
			bufferPtr := r.Exceptions.CatchBufferPtr

			// Write current buffer pointer to cache.
			binary.LittleEndian.PutUint32(vm.Memory[bufferPtr>>2:bufferPtr>>2+4], uint32(bufferPtr))

			// TODO: Check if thrown exception type really can match one of the catches type. `___cxa_can_catch()`
			for range vm.GetCurrentFrame().Locals {
				if true {
					thrown = binary.LittleEndian.Uint32(vm.Memory[thrown>>2 : thrown>>2+4])
					info.adjusted = thrown

					r.tempRet0 = int64(thrown)
					return r.tempRet0
				}
			}

			thrown = binary.LittleEndian.Uint32(vm.Memory[thrown>>2 : thrown>>2+4])
			r.tempRet0 = int64(thrown)
			return r.tempRet0
		}

	case "___cxa_throw":
		return func(vm *exec.VirtualMachine) int64 {
			frame := vm.GetCurrentFrame()
			ptr := uint32(frame.Locals[0])
			typ := int(frame.Locals[1])
			destructor := int(frame.Locals[2])

			r.Exceptions.Infos[ptr] = Exception{
				ptr:        ptr,
				adjusted:   ptr,
				typ:        typ,
				destructor: destructor,
				refcount:   0,
				caught:     false,
				rethrown:   false,
			}
			r.Exceptions.Last = ptr

			return 0
		}

	case "___resumeException":
		return func(vm *exec.VirtualMachine) int64 {
			frame := vm.GetCurrentFrame()
			ptr := uint32(frame.Locals[0])

			if r.Exceptions.Last == 0 {
				r.Exceptions.Last = ptr
				return 0
			}

			panic(ptr)
		}

	default:
		if strings.HasPrefix(field, "nullFunc_") {
			return func(vm *exec.VirtualMachine) int64 {
				panic("nullFunc called")
			}
		}
		if strings.HasPrefix(field, "invoke_") {
			return func(vm *exec.VirtualMachine) int64 {
				panic("dynamic invoke not supported temporarily")
			}
		}
		if strings.HasPrefix(field, "___syscall") {
			return func(vm *exec.VirtualMachine) int64 {
				panic(fmt.Errorf("syscall %s not supported", field))
			}
		}
		if strings.HasPrefix(field, "jsCall_") {
			return func(vm *exec.VirtualMachine) int64 {
				panic(fmt.Errorf("jsCall %s not supported", field))
			}
		}
		panic(fmt.Errorf("unknown field: %s", field))
	}
}

func (r *Resolver) ResolveGlobal(module, field string) int64 {
	fmt.Printf("Resolve global: %s %s\n", module, field)
	switch module {
	case "env":
		switch field {
		case "__life_magic":
			return 424
		case "memoryBase":
			return STATIC_BASE
		case "tableBase":
			return 0
		case "DYNAMICTOP_PTR":
			return int64(DYNAMICTOP_PTR)
		case "tempDoublePtr":
			return tempDoublePtr
		case "STACK_BASE":
			return STACK_BASE
		case "STACKTOP":
			return STACKTOP
		case "STACK_MAX":
			return STACK_MAX
		case "ABORT":
			return 0
		case "gb":
			return GLOBAL_BASE
		case "fb":
			return 0
		default:
			panic(fmt.Errorf("unknown field: %s", field))
		}
	case "global":
		switch field {
		case "NaN":
			return int64(math.Float64bits(math.NaN()))
		case "Infinity":
			return int64(math.Float64bits(math.Inf(1)))
		default:
			panic(fmt.Errorf("unknown field: %s", field))
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
}

func main() {
	entryFunctionFlag := flag.String("entry", "app_main", "entry function id")
	flag.Parse()

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	vm, err := exec.NewVirtualMachine(input, exec.VMConfig{
		DefaultMemoryPages: 128,
		DefaultTableSize:   65536,
	}, NewResolver())
	if err != nil {
		panic(err)
	}

	entryID, ok := vm.GetFunctionExport(*entryFunctionFlag)
	if !ok {
		fmt.Printf("Entry function %s not found; starting from 0.\n", *entryFunctionFlag)
		entryID = 0
	}

	start := time.Now()

	if vm.Module.Base.Start != nil {
		startID := int(vm.Module.Base.Start.Index)
		_, err := vm.Run(startID)
		if err != nil {
			vm.PrintStackTrace()
			panic(err)
		}
	}

	ret, err := vm.Run(entryID)
	if err != nil {
		vm.PrintStackTrace()
		panic(err)
	}
	end := time.Now()
	fmt.Printf("return value = %d, duration = %v\n", ret, end.Sub(start))
}
