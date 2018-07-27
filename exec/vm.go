package exec

import (
	"encoding/binary"
	"fmt"

	"math"

	"math/bits"

	"github.com/go-interpreter/wagon/wasm"

	"github.com/perlin-network/life/compiler"
	"github.com/perlin-network/life/compiler/opcodes"
	"github.com/perlin-network/life/utils"
)

type FunctionImport func(vm *VirtualMachine) int64

const DefaultCallStackSize = 512
const DefaultPageSize = 65536

var LE = binary.LittleEndian

type VirtualMachine struct {
	Config          VMConfig
	Module          *compiler.Module
	FunctionCode    []compiler.InterpreterCode
	FunctionImports []FunctionImport
	CallStack       []Frame
	CurrentFrame    int
	Table           []uint32
	Globals         []int64
	Memory          []byte
	NumValueSlots   int
	Yielded         int64
	InsideExecute   bool
	Delegate        func()
	Exited          bool
	ExitError       interface{}
	ReturnValue     int64
}

type VMConfig struct {
	MaxMemoryPages int
	MaxTableSize      int
	MaxValueSlots     int
	MaxCallStackDepth int
}

type Frame struct {
	FunctionID int
	Code       []byte
	Regs       []int64
	Locals     []int64
	IP         int
	ReturnReg  int
}

type ImportResolver interface {
	ResolveFunc(module, field string) FunctionImport
	ResolveGlobal(module, field string) int64
}

func NewVirtualMachine(
	code []byte,
	config VMConfig,
	impResolver ImportResolver,
) (_retVM *VirtualMachine, retErr error) {
	m, err := compiler.LoadModule(code)
	if err != nil {
		return nil, err
	}

	functionCode, err := m.CompileForInterpreter()
	if err != nil {
		return nil, err
	}

	defer utils.CatchPanic(&retErr)

	table := make([]uint32, 0)
	globals := make([]int64, 0)
	funcImports := make([]FunctionImport, 0)

	if m.Base.Import != nil && impResolver != nil {
		for _, imp := range m.Base.Import.Entries {
			switch imp.Kind {
			case wasm.ExternalFunction:
				funcImports = append(funcImports, impResolver.ResolveFunc(imp.ModuleName, imp.FieldName))
			case wasm.ExternalGlobal:
				globals = append(globals, impResolver.ResolveGlobal(imp.ModuleName, imp.FieldName))
			default:
				panic(fmt.Errorf("import kind not supported: %d", imp.Kind))
			}
		}
	}

	// Populate table elements.
	if m.Base.Table != nil && len(m.Base.Table.Entries) > 0 {
		t := &m.Base.Table.Entries[0]

		if config.MaxTableSize != 0 && int(t.Limits.Initial) > config.MaxTableSize {
			panic("max table size exceeded")
		}

		table = make([]uint32, int(t.Limits.Initial))
		for i := 0; i < int(t.Limits.Initial); i++ {
			table[i] = 0xffffffff
		}
		if m.Base.Elements != nil && len(m.Base.Elements.Entries) > 0 {
			for _, e := range m.Base.Elements.Entries {
				maybeOffset, err := m.Base.ExecInitExpr(e.Offset)
				if err != nil {
					panic(err)
				}
				offset := int(maybeOffset.(int32))
				copy(table[offset:], e.Elems)
			}
		}
	}

	// Load global entries.
	for _, entry := range m.Base.GlobalIndexSpace {
		value, err := m.Base.ExecInitExpr(entry.Init)
		if err != nil {
			panic(err)
		}

		switch value := value.(type) {
		case int32:
			globals = append(globals, int64(value))
		case int64:
			globals = append(globals, value)
		case float32:
			globals = append(globals, int64(math.Float32bits(value)))
		case float64:
			globals = append(globals, int64(math.Float64bits(value)))
		default:
			panic("got an impossible global value type")
		}
	}

	// Load linear memory.
	memory := make([]byte, 0)
	if m.Base.Memory != nil && len(m.Base.Memory.Entries) > 0 {
		initialLimit := int(m.Base.Memory.Entries[0].Limits.Initial)
		if config.MaxMemoryPages != 0 && initialLimit > config.MaxMemoryPages {
			panic("max memory exceeded")
		}

		capacity := initialLimit * DefaultPageSize

		// Initialize empty memory.
		memory = make([]byte, capacity)
		for i := 0; i < capacity; i++ {
			memory[i] = 0
		}

		if m.Base.Data != nil && len(m.Base.Data.Entries) > 0 {
			for _, e := range m.Base.Data.Entries {
				_offset, err := m.Base.ExecInitExpr(e.Offset)
				if err != nil {
					panic(err)
				}

				offset, ok := _offset.(int32)
				if !ok {
					panic("linear memory offset is not varuint32")
				}

				copy(memory[int(offset):], e.Data)
			}
		}
	}

	return &VirtualMachine{
		Module:          m,
		Config: config,
		FunctionCode:    functionCode,
		FunctionImports: funcImports,
		CallStack:       make([]Frame, DefaultCallStackSize),
		CurrentFrame:    -1,
		Table:           table,
		Globals:         globals,
		Memory:          memory,
		Exited:          true,
	}, nil
}

func (f *Frame) Init(vm *VirtualMachine, functionID int, code compiler.InterpreterCode) {
	numValueSlots := code.NumRegs + code.NumParams + code.NumLocals
	if vm.Config.MaxValueSlots != 0 && vm.NumValueSlots+numValueSlots > vm.Config.MaxValueSlots {
		panic("max value slot count exceeded")
	}
	vm.NumValueSlots += numValueSlots

	values := make([]int64, numValueSlots)

	f.FunctionID = functionID
	f.Regs = values[:code.NumRegs]
	f.Locals = values[code.NumRegs:]
	f.Code = code.Bytes
	f.IP = 0

	//fmt.Printf("Enter function %d\n", functionID)
}

func (f *Frame) Destroy(vm *VirtualMachine) {
	numValueSlots := len(f.Regs) + len(f.Locals)
	vm.NumValueSlots -= numValueSlots

	//fmt.Printf("Leave function %d\n", f.FunctionID)
}

func (vm *VirtualMachine) GetCurrentFrame() *Frame {
	if vm.Config.MaxCallStackDepth != 0 && vm.CurrentFrame >= vm.Config.MaxCallStackDepth {
		panic("max call stack depth exceeded")
	}

	if vm.CurrentFrame >= len(vm.CallStack) {
		panic("call stack overflow")
		//vm.CallStack = append(vm.CallStack, make([]Frame, DefaultCallStackSize / 2)...)
	}
	return &vm.CallStack[vm.CurrentFrame]
}

func (vm *VirtualMachine) getExport(key string, kind wasm.External) (int, bool) {
	if vm.Module.Base.Export == nil {
		return -1, false
	}

	entry, ok := vm.Module.Base.Export.Entries[key]
	if !ok {
		return -1, false
	}

	if entry.Kind != kind {
		return -1, false
	}

	return int(entry.Index), true
}

func (vm *VirtualMachine) GetGlobalExport(key string) (int, bool) {
	return vm.getExport(key, wasm.ExternalGlobal)
}

func (vm *VirtualMachine) GetFunctionExport(key string) (int, bool) {
	return vm.getExport(key, wasm.ExternalFunction)
}

func (vm *VirtualMachine) PrintStackTrace() {
	fmt.Println("--- Begin stack trace ---")
	for i := vm.CurrentFrame; i >= 0; i-- {
		fmt.Printf("<%d> function %d\n", i, vm.CallStack[i].FunctionID)
	}
	fmt.Println("--- End stack trace ---")
}

// Init the first frame.
func (vm *VirtualMachine) Ignite(functionID int, params... int64) {
	if vm.ExitError != nil {
		panic("last execution exited with error; cannot ignite.")
	}

	if vm.CurrentFrame != -1 {
		panic("call stack not empty; cannot ignite.")
	}

	code := vm.FunctionCode[functionID]
	if code.NumParams != len(params) {
		panic("param count mismatch")
	}

	vm.Exited = false

	vm.CurrentFrame++
	frame := vm.GetCurrentFrame()
	frame.Init(
		vm,
		functionID,
		code,
	)
	copy(frame.Locals, params)
}

func (vm *VirtualMachine) Execute() {
	if vm.Exited == true {
		panic("attempting to execute an exited vm")
	}

	if vm.Delegate != nil {
		panic("delegate not cleared")
	}

	if vm.InsideExecute {
		panic("vm execution is not re-entrant")
	}
	vm.InsideExecute = true

	defer func() {
		vm.InsideExecute = false
		if err := recover(); err != nil {
			vm.Exited = true
			vm.ExitError = err
		}
	}()

	frame := vm.GetCurrentFrame()

	cycleCount := 0

	for {
		if cycleCount == 10000 {
			return
		}
		cycleCount++

		valueID := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
		ins := opcodes.Opcode(frame.Code[frame.IP+4])
		frame.IP += 5

		//fmt.Printf("INS: [%d] %d\n", valueID, ins)

		switch ins {
		case opcodes.Nop:
		case opcodes.Unreachable:
			panic("wasm: unreachable executed")
		case opcodes.Select:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			c := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])
			frame.IP += 12
			if c != 0 {
				frame.Regs[valueID] = a
			} else {
				frame.Regs[valueID] = b
			}
		case opcodes.I32Const, opcodes.F32Const:
			val := LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			frame.IP += 4
			frame.Regs[valueID] = int64(val)
		case opcodes.I32Add:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			frame.Regs[valueID] = int64(a + b)
		case opcodes.I32Sub:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			frame.Regs[valueID] = int64(a - b)
		case opcodes.I32Mul:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			frame.Regs[valueID] = int64(a * b)
		case opcodes.I32DivS:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			if b == 0 {
				panic("integer division by zero")
			}

			if a == math.MinInt32 && b == -1 {
				panic("signed integer overflow")
			}

			frame.IP += 8
			frame.Regs[valueID] = int64(a / b)
		case opcodes.I32DivU:
			a := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			if b == 0 {
				panic("integer division by zero")
			}

			frame.IP += 8
			frame.Regs[valueID] = int64(a / b)
		case opcodes.I32RemS:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			if b == 0 {
				panic("integer division by zero")
			}

			frame.IP += 8
			frame.Regs[valueID] = int64(a % b)
		case opcodes.I32RemU:
			a := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			if b == 0 {
				panic("integer division by zero")
			}

			frame.IP += 8
			frame.Regs[valueID] = int64(a % b)
		case opcodes.I32And:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(a & b)
		case opcodes.I32Or:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(a | b)
		case opcodes.I32Xor:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(a ^ b)
		case opcodes.I32Shl:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(a << (b % 32))
		case opcodes.I32ShrS:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(a >> (b % 32))
		case opcodes.I32ShrU:
			a := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(a >> (b % 32))
		case opcodes.I32Rotl:
			a := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(bits.RotateLeft32(a, int(b)))
		case opcodes.I32Rotr:
			a := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(bits.RotateLeft32(a, -int(b)))
		case opcodes.I32Clz:
			val := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])

			frame.IP += 4
			frame.Regs[valueID] = int64(bits.LeadingZeros32(val))
		case opcodes.I32Ctz:
			val := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])

			frame.IP += 4
			frame.Regs[valueID] = int64(bits.TrailingZeros32(val))
		case opcodes.I32PopCnt:
			val := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])

			frame.IP += 4
			frame.Regs[valueID] = int64(bits.OnesCount32(val))
		case opcodes.I32EqZ:
			val := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])

			frame.IP += 4
			if val == 0 {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I32Eq:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a == b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I32Ne:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a != b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I32LtS:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a < b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I32LtU:
			a := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a < b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I32LeS:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a <= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I32LeU:
			a := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a <= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I32GtS:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a > b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I32GtU:
			a := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a > b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I32GeS:
			a := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a >= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I32GeU:
			a := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a >= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64Const, opcodes.F64Const:
			val := LE.Uint64(frame.Code[frame.IP : frame.IP+8])
			frame.IP += 8
			frame.Regs[valueID] = int64(val)
		case opcodes.I64Add:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			frame.Regs[valueID] = a + b
		case opcodes.I64Sub:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			frame.Regs[valueID] = a - b
		case opcodes.I64Mul:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			frame.Regs[valueID] = a * b
		case opcodes.I64DivS:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]

			if b == 0 {
				panic("integer division by zero")
			}

			if a == math.MinInt64 && b == -1 {
				panic("signed integer overflow")
			}

			frame.IP += 8
			frame.Regs[valueID] = a / b
		case opcodes.I64DivU:
			a := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			if b == 0 {
				panic("integer division by zero")
			}

			frame.IP += 8
			frame.Regs[valueID] = int64(a / b)
		case opcodes.I64RemS:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]

			if b == 0 {
				panic("integer division by zero")
			}

			frame.IP += 8
			frame.Regs[valueID] = a % b
		case opcodes.I64RemU:
			a := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			if b == 0 {
				panic("integer division by zero")
			}

			frame.IP += 8
			frame.Regs[valueID] = int64(a % b)
		case opcodes.I64And:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]

			frame.IP += 8
			frame.Regs[valueID] = a & b
		case opcodes.I64Or:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]

			frame.IP += 8
			frame.Regs[valueID] = a | b
		case opcodes.I64Xor:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]

			frame.IP += 8
			frame.Regs[valueID] = a ^ b
		case opcodes.I64Shl:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = a << (b % 64)
		case opcodes.I64ShrS:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = a >> (b % 64)
		case opcodes.I64ShrU:
			a := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(a >> (b % 64))
		case opcodes.I64Rotl:
			a := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(bits.RotateLeft64(a, int(b)))
		case opcodes.I64Rotr:
			a := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])

			frame.IP += 8
			frame.Regs[valueID] = int64(bits.RotateLeft64(a, -int(b)))
		case opcodes.I64Clz:
			val := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])

			frame.IP += 4
			frame.Regs[valueID] = int64(bits.LeadingZeros64(val))
		case opcodes.I64Ctz:
			val := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])

			frame.IP += 4
			frame.Regs[valueID] = int64(bits.TrailingZeros64(val))
		case opcodes.I64PopCnt:
			val := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])

			frame.IP += 4
			frame.Regs[valueID] = int64(bits.OnesCount64(val))
		case opcodes.I64EqZ:
			val := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])

			frame.IP += 4
			if val == 0 {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64Eq:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			if a == b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64Ne:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			if a != b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64LtS:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			if a < b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64LtU:
			a := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a < b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64LeS:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			if a <= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64LeU:
			a := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a <= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64GtS:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			if a > b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64GtU:
			a := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a > b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64GeS:
			a := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			b := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			if a >= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.I64GeU:
			a := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			b := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))])
			frame.IP += 8
			if a >= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F32Add:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float32bits(a + b))
		case opcodes.F32Sub:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float32bits(a - b))
		case opcodes.F32Mul:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float32bits(a * b))
		case opcodes.F32Div:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float32bits(a / b))
		case opcodes.F32Sqrt:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(math.Sqrt(float64(val)))))
		case opcodes.F32Min:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float32bits(float32(math.Min(float64(a), float64(b)))))
		case opcodes.F32Max:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float32bits(float32(math.Max(float64(a), float64(b)))))
		case opcodes.F32Ceil:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(math.Ceil(float64(val)))))
		case opcodes.F32Floor:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(math.Floor(float64(val)))))
		case opcodes.F32Trunc:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(math.Trunc(float64(val)))))
		case opcodes.F32Nearest:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(math.RoundToEven(float64(val)))))
		case opcodes.F32Abs:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(math.Abs(float64(val)))))
		case opcodes.F32Neg:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(-val))
		case opcodes.F32CopySign:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float32bits(float32(math.Copysign(float64(a), float64(b)))))
		case opcodes.F32Eq:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a == b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F32Ne:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a != b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F32Lt:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a < b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F32Le:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a <= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F32Gt:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a > b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F32Ge:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a >= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F64Add:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float64bits(a + b))
		case opcodes.F64Sub:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float64bits(a - b))
		case opcodes.F64Mul:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float64bits(a * b))
		case opcodes.F64Div:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float64bits(a / b))
		case opcodes.F64Sqrt:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float64bits(math.Sqrt(val)))
		case opcodes.F64Min:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float64bits(math.Min(a, b)))
		case opcodes.F64Max:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float64bits(math.Max(a, b)))
		case opcodes.F64Ceil:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float64bits(math.Ceil(val)))
		case opcodes.F64Floor:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float64bits(math.Floor(val)))
		case opcodes.F64Trunc:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float64bits(math.Trunc(val)))
		case opcodes.F64Nearest:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float64bits(math.RoundToEven(val)))
		case opcodes.F64Abs:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float64bits(math.Abs(val)))
		case opcodes.F64Neg:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float64bits(-val))
		case opcodes.F64CopySign:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			frame.Regs[valueID] = int64(math.Float64bits(math.Copysign(a, b)))
		case opcodes.F64Eq:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a == b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F64Ne:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a != b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F64Lt:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a < b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F64Le:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a <= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F64Gt:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a > b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}
		case opcodes.F64Ge:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8
			if a >= b {
				frame.Regs[valueID] = 1
			} else {
				frame.Regs[valueID] = 0
			}

		case opcodes.I32WrapI64:
			v := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			frame.IP += 4
			frame.Regs[valueID] = int64(v)

		case opcodes.I64ExtendUI32:
			v := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			frame.IP += 4
			frame.Regs[valueID] = int64(v)

		case opcodes.I64ExtendSI32:
			v := int32(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(v)

		case opcodes.I32Load, opcodes.I64Load32U:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(uint32(LE.Uint32(vm.Memory[effective : effective+4])))
		case opcodes.I64Load32S:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(int32(LE.Uint32(vm.Memory[effective : effective+4])))
		case opcodes.I64Load:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(LE.Uint64(vm.Memory[effective : effective+8]))
		case opcodes.I32Load8S, opcodes.I64Load8S:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(int8(vm.Memory[effective]))
		case opcodes.I32Load8U, opcodes.I64Load8U:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(uint8(vm.Memory[effective]))
		case opcodes.I32Load16S, opcodes.I64Load16S:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(int16(LE.Uint16(vm.Memory[effective : effective+2])))
		case opcodes.I32Load16U, opcodes.I64Load16U:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(uint16(LE.Uint16(vm.Memory[effective : effective+2])))
		case opcodes.I32Store, opcodes.I64Store32:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			value := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+12:frame.IP+16]))]

			frame.IP += 16

			effective := int(uint64(base) + uint64(offset))
			LE.PutUint32(vm.Memory[effective:effective+4], uint32(value))
		case opcodes.I64Store:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			value := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+12:frame.IP+16]))]

			frame.IP += 16

			effective := int(uint64(base) + uint64(offset))
			LE.PutUint64(vm.Memory[effective:effective+8], uint64(value))
		case opcodes.I32Store8, opcodes.I64Store8:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			value := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+12:frame.IP+16]))]

			frame.IP += 16

			effective := int(uint64(base) + uint64(offset))
			vm.Memory[effective] = byte(value)
		case opcodes.I32Store16, opcodes.I64Store16:
			LE.Uint32(frame.Code[frame.IP : frame.IP+4])
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			value := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+12:frame.IP+16]))]

			frame.IP += 16

			effective := int(uint64(base) + uint64(offset))
			LE.PutUint16(vm.Memory[effective:effective+2], uint16(value))

		case opcodes.Jmp:
			target := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			vm.Yielded = frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP = target
		case opcodes.JmpIf:
			target := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			cond := int(LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8]))
			yieldedReg := int(LE.Uint32(frame.Code[frame.IP+8 : frame.IP+12]))
			frame.IP += 12
			if frame.Regs[cond] != 0 {
				vm.Yielded = frame.Regs[yieldedReg]
				frame.IP = target
			}
		case opcodes.JmpTable:
			targetCount := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			frame.IP += 4

			targetsRaw := frame.Code[frame.IP : frame.IP+4*targetCount]
			frame.IP += 4 * targetCount

			defaultTarget := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			frame.IP += 4

			cond := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			frame.IP += 4

			vm.Yielded = frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			frame.IP += 4

			val := int(frame.Regs[cond])
			if val >= 0 && val < targetCount {
				frame.IP = int(LE.Uint32(targetsRaw[val*4 : val*4+4]))
			} else {
				frame.IP = defaultTarget
			}
		case opcodes.ReturnValue:
			val := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			frame.Destroy(vm)
			vm.CurrentFrame--
			if vm.CurrentFrame == -1 {
				vm.Exited = true
				vm.ReturnValue = val
				return
			} else {
				frame = vm.GetCurrentFrame()
				frame.Regs[frame.ReturnReg] = val
			}
		case opcodes.ReturnVoid:
			frame.Destroy(vm)
			vm.CurrentFrame--
			if vm.CurrentFrame == -1 {
				vm.Exited = true
				vm.ReturnValue = 0
				return
			} else {
				frame = vm.GetCurrentFrame()
			}
		case opcodes.GetLocal:
			val := frame.Locals[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			frame.IP += 4
			frame.Regs[valueID] = val
		case opcodes.SetLocal:
			id := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			val := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			frame.Locals[id] = val
		case opcodes.GetGlobal:
			frame.Regs[valueID] = vm.Globals[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			frame.IP += 4
		case opcodes.SetGlobal:
			id := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			val := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8

			vm.Globals[id] = val
		case opcodes.Call:
			functionID := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			frame.IP += 4
			argCount := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			frame.IP += 4
			argsRaw := frame.Code[frame.IP : frame.IP+4*argCount]
			frame.IP += 4 * argCount

			oldRegs := frame.Regs
			frame.ReturnReg = valueID

			vm.CurrentFrame++
			frame = vm.GetCurrentFrame()
			frame.Init(vm, functionID, vm.FunctionCode[functionID])
			for i := 0; i < argCount; i++ {
				frame.Locals[i] = oldRegs[int(LE.Uint32(argsRaw[i*4:i*4+4]))]
			}

		case opcodes.CallIndirect:
			typeID := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			frame.IP += 4
			argCount := int(LE.Uint32(frame.Code[frame.IP:frame.IP+4])) - 1
			frame.IP += 4
			argsRaw := frame.Code[frame.IP : frame.IP+4*argCount]
			frame.IP += 4 * argCount
			tableItemID := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			frame.IP += 4

			sig := &vm.Module.Base.Types.Entries[typeID]

			functionID := int(vm.Table[tableItemID])
			code := vm.FunctionCode[functionID]

			// TODO: We are only checking CC here; Do we want strict typeck?
			if code.NumParams != len(sig.ParamTypes) || code.NumReturns != len(sig.ReturnTypes) {
				panic("type mismatch")
			}

			oldRegs := frame.Regs
			frame.ReturnReg = valueID

			vm.CurrentFrame++
			frame = vm.GetCurrentFrame()
			frame.Init(vm, functionID, code)
			for i := 0; i < argCount; i++ {
				frame.Locals[i] = oldRegs[int(LE.Uint32(argsRaw[i*4:i*4+4]))]
			}

		case opcodes.InvokeImport:
			importID := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			frame.IP += 4
			vm.Delegate = func() {
				frame.Regs[valueID] = vm.FunctionImports[importID](vm)
			}
			return

		case opcodes.CurrentMemory:
			frame.Regs[valueID] = int64(len(vm.Memory) / DefaultPageSize)

		case opcodes.GrowMemory:
			n := int(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))]))
			frame.IP += 4

			current := len(vm.Memory) / DefaultPageSize
			if vm.Config.MaxMemoryPages == 0 || (current + n >= current && current + n <= vm.Config.MaxMemoryPages) {
				frame.Regs[valueID] = int64(current)
				vm.Memory = append(vm.Memory, make([]byte, n * DefaultPageSize)...)
			} else {
				frame.Regs[valueID] = -1
			}

		case opcodes.Phi:
			frame.Regs[valueID] = vm.Yielded
		default:
			panic("unknown instruction")
		}
	}
}
