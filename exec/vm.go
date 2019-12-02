package exec

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
	"runtime/debug"
	"strings"

	"github.com/go-interpreter/wagon/wasm"

	"github.com/perlin-network/life/compiler"
	"github.com/perlin-network/life/compiler/opcodes"
	"github.com/perlin-network/life/utils"
)

type FunctionImport func(vm *VirtualMachine) int64

const (
	// DefaultCallStackSize is the default call stack size.
	DefaultCallStackSize = 512

	// DefaultPageSize is the linear memory page size.
	DefaultPageSize = 65536

	// JITCodeSizeThreshold is the lower-bound code size threshold for the JIT compiler.
	JITCodeSizeThreshold = 30
)

// LE is a simple alias to `binary.LittleEndian`.
var LE = binary.LittleEndian

type FunctionImportInfo struct {
	ModuleName string
	FieldName  string
	F          FunctionImport
}

type NCompileConfig struct {
	AliasDef             bool
	DisableMemBoundCheck bool
}

type AOTService interface {
	Initialize(vm *VirtualMachine)
	UnsafeInvokeFunction_0(vm *VirtualMachine, name string) uint64
	UnsafeInvokeFunction_1(vm *VirtualMachine, name string, p0 uint64) uint64
	UnsafeInvokeFunction_2(vm *VirtualMachine, name string, p0, p1 uint64) uint64
}

// VirtualMachine is a WebAssembly execution environment.
type VirtualMachine struct {
	Config           VMConfig
	Module           *compiler.Module
	FunctionCode     []compiler.InterpreterCode
	FunctionImports  []FunctionImportInfo
	CallStack        []Frame
	CurrentFrame     int
	Table            []uint32
	Globals          []int64
	Memory           []byte
	NumValueSlots    int
	Yielded          int64
	InsideExecute    bool
	Delegate         func()
	Exited           bool
	ExitError        interface{}
	ReturnValue      int64
	Gas              uint64
	GasLimitExceeded bool
	GasPolicy        compiler.GasPolicy
	ImportResolver   ImportResolver
	AOTService       AOTService
	StackTrace       string
}

// VMConfig denotes a set of options passed to a single VirtualMachine insta.ce
type VMConfig struct {
	EnableJIT                bool
	MaxMemoryPages           int
	MaxTableSize             int
	MaxValueSlots            int
	MaxCallStackDepth        int
	DefaultMemoryPages       int
	DefaultTableSize         int
	GasLimit                 uint64
	DisableFloatingPoint     bool
	ReturnOnGasLimitExceeded bool
}

// Frame represents a call frame.
type Frame struct {
	FunctionID   int
	Code         []byte
	Regs         []int64
	Locals       []int64
	IP           int
	ReturnReg    int
	Continuation int32
}

// ImportResolver is an interface for allowing one to define imports to WebAssembly modules
// ran under a single VirtualMachine instance.
type ImportResolver interface {
	ResolveFunc(module, field string) FunctionImport
	ResolveGlobal(module, field string) int64
}

type Module struct {
	Config          VMConfig
	Module          *compiler.Module
	FunctionCode    []compiler.InterpreterCode
	FunctionImports []FunctionImportInfo
	Table           []uint32
	Globals         []int64
	GasPolicy       compiler.GasPolicy
	ImportResolver  ImportResolver
}

var (
	emptyGlobals     = []int64{}
	emptyFuncImports = []FunctionImportInfo{}
	emptyMemory      = []byte{}
	emptyTable       = []uint32{}
)

// NewModule instantiates a module for a given WebAssembly module, with
// specific execution options specified under a VMConfig, and a WebAssembly module import
// resolver.
func NewModule(
	code []byte,
	config VMConfig,
	impResolver ImportResolver,
	gasPolicy compiler.GasPolicy,
) (_retVM *Module, retErr error) {
	if config.EnableJIT {
		fmt.Println("Warning: JIT support is removed.")
	}

	m, err := compiler.LoadModule(code)
	if err != nil {
		return nil, err
	}

	m.DisableFloatingPoint = config.DisableFloatingPoint

	functionCode, err := m.CompileForInterpreter(gasPolicy)
	if err != nil {
		return nil, err
	}

	defer utils.CatchPanic(&retErr)

	table := emptyTable
	globals := emptyGlobals
	funcImports := emptyFuncImports

	if m.Base.Import != nil && impResolver != nil {
		for _, imp := range m.Base.Import.Entries {
			switch imp.Type.Kind() {
			case wasm.ExternalFunction:
				funcImports = append(funcImports, FunctionImportInfo{
					ModuleName: imp.ModuleName,
					FieldName:  imp.FieldName,
					F:          nil, // deferred
				})
			case wasm.ExternalGlobal:
				globals = append(globals, impResolver.ResolveGlobal(imp.ModuleName, imp.FieldName))
			case wasm.ExternalMemory:
				// TODO: Do we want a real import?
				if m.Base.Memory != nil && len(m.Base.Memory.Entries) > 0 {
					panic("cannot import another memory while we already have one")
				}
				m.Base.Memory = &wasm.SectionMemories{
					Entries: []wasm.Memory{
						{
							Limits: wasm.ResizableLimits{
								Initial: uint32(config.DefaultMemoryPages),
							},
						},
					},
				}
			case wasm.ExternalTable:
				// TODO: Do we want a real import?
				if m.Base.Table != nil && len(m.Base.Table.Entries) > 0 {
					panic("cannot import another table while we already have one")
				}
				m.Base.Table = &wasm.SectionTables{
					Entries: []wasm.Table{
						{
							Limits: wasm.ResizableLimits{
								Initial: uint32(config.DefaultTableSize),
							},
						},
					},
				}
			default:
				panic(fmt.Errorf("import kind not supported: %d", imp.Type.Kind()))
			}
		}
	}

	// Load global entries.
	for _, entry := range m.Base.GlobalIndexSpace {
		globals = append(globals, execInitExpr(entry.Init, globals))
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
				offset := int(execInitExpr(e.Offset, globals))
				copy(table[offset:], e.Elems)
			}
		}
	}

	return &Module{
		Module:          m,
		Config:          config,
		FunctionCode:    functionCode,
		FunctionImports: funcImports,
		Table:           table,
		Globals:         globals,
		GasPolicy:       gasPolicy,
		ImportResolver:  impResolver,
	}, nil
}

func (m *Module) getExport(key string, kind wasm.External) (int, bool) {
	if m.Module.Base.Export == nil {
		return -1, false
	}

	entry, ok := m.Module.Base.Export.Entries[key]
	if !ok {
		return -1, false
	}

	if entry.Kind != kind {
		return -1, false
	}

	return int(entry.Index), true
}

// GetGlobalExport returns the global export with the given name.
func (m *Module) GetGlobalExport(key string) (int, bool) {
	return m.getExport(key, wasm.ExternalGlobal)
}

// GetFunctionExport returns the function export with the given name.
func (m *Module) GetFunctionExport(key string) (int, bool) {
	return m.getExport(key, wasm.ExternalFunction)
}

func (m *Module) GenerateNEnv(config NCompileConfig) string {
	builder := &strings.Builder{}

	bSprintf(builder, "#include <stdint.h>\n\n")

	if config.DisableMemBoundCheck {
		builder.WriteString("#define POLYMERASE_NO_MEM_BOUND_CHECK\n")
	}

	builder.WriteString(compiler.NGEN_HEADER)
	if !m.Config.DisableFloatingPoint {
		builder.WriteString(compiler.NGEN_FP_HEADER)
	}

	bSprintf(builder, "static uint64_t globals[] = {")
	for _, v := range m.Globals {
		bSprintf(builder, "%dull,", uint64(v))
	}
	bSprintf(builder, "};\n")

	for i, code := range m.FunctionCode {
		bSprintf(builder, "uint64_t %s%d(struct VirtualMachine *", compiler.NGEN_FUNCTION_PREFIX, i)
		for j := 0; j < code.NumParams; j++ {
			bSprintf(builder, ",uint64_t")
		}
		bSprintf(builder, ");\n")
	}

	// call_indirect dispatcher.
	bSprintf(builder, "struct TableEntry { uint64_t num_params; void *func; };\n")
	bSprintf(builder, "static const uint64_t num_table_entries = %d;\n", len(m.Table))
	bSprintf(builder, "static struct TableEntry table[] = {\n")
	for _, entry := range m.Table {
		if entry == math.MaxUint32 {
			bSprintf(builder, "{ .num_params = 0, .func = 0 },\n")
		} else {
			functionID := int(entry)
			code := m.FunctionCode[functionID]

			bSprintf(builder, "{ .num_params = %d, .func = %s%d },\n", code.NumParams, compiler.NGEN_FUNCTION_PREFIX, functionID)
		}
	}
	bSprintf(builder, "};\n")
	bSprintf(builder, "static void * __attribute__((always_inline)) %sresolve_indirect(struct VirtualMachine *vm, uint64_t entry_id, uint64_t num_params) {\n", compiler.NGEN_ENV_API_PREFIX)
	bSprintf(builder, "if(entry_id >= num_table_entries) { vm->throw_s(vm, \"%s\"); }\n", "table entry out of bounds")
	bSprintf(builder, "if(table[entry_id].func == 0) { vm->throw_s(vm, \"%s\"); }\n", "table entry is null")
	bSprintf(builder, "if(table[entry_id].num_params != num_params) { vm->throw_s(vm, \"%s\"); }\n", "argument count mismatch")
	bSprintf(builder, "return table[entry_id].func;\n")
	bSprintf(builder, "}\n")

	bSprintf(builder, "struct ImportEntry { const char *module_name; const char *field_name; ExternalFunction f; };\n")
	bSprintf(builder, "static const uint64_t num_import_entries = %d;\n", len(m.FunctionImports))
	bSprintf(builder, "static struct ImportEntry imports[] = {\n")
	for _, imp := range m.FunctionImports {
		bSprintf(builder, "{ .module_name = \"%s\", .field_name = \"%s\", .f = 0 },\n", escapeName(imp.ModuleName), escapeName(imp.FieldName))
	}
	bSprintf(builder, "};\n")
	bSprintf(builder,
		"static uint64_t __attribute__((always_inline)) %sinvoke_import(struct VirtualMachine *vm, uint64_t import_id, uint64_t num_params, uint64_t *params) {\n",
		compiler.NGEN_ENV_API_PREFIX,
	)

	bSprintf(builder, "if(import_id >= num_import_entries) { vm->throw_s(vm, \"%s\"); }\n", "import entry out of bounds")
	bSprintf(builder, "if(imports[import_id].f == 0) { imports[import_id].f = vm->resolve_import(vm, imports[import_id].module_name, imports[import_id].field_name); }\n")
	bSprintf(builder, "if(imports[import_id].f == 0) { vm->throw_s(vm, \"%s\"); }\n", "cannot resolve import")
	bSprintf(builder, "return imports[import_id].f(vm, import_id, num_params, params);\n")
	bSprintf(builder, "}\n")

	return builder.String()
}

func (m *Module) NBuildAliasDef() string {
	builder := &strings.Builder{}

	builder.WriteString("// Aliases for exported functions\n")

	if m.Module.Base.Export != nil {
		for name, exp := range m.Module.Base.Export.Entries {
			if exp.Kind == wasm.ExternalFunction {
				bSprintf(builder, "#define %sexport_%s %s%d\n", compiler.NGEN_FUNCTION_PREFIX, filterName(name), compiler.NGEN_FUNCTION_PREFIX, exp.Index)
			}
		}
	}

	return builder.String()
}

func (m *Module) NCompile(config NCompileConfig) string {
	body, err := m.Module.CompileWithNGen(m.GasPolicy, uint64(len(m.Globals)))
	if err != nil {
		panic(err)
	}

	out := m.GenerateNEnv(config) + "\n" + body
	if config.AliasDef {
		out += "\n"
		out += m.NBuildAliasDef()
	}

	return out
}

// NewVirtualMachine instantiates a virtual machine for the module.
func (m *Module) NewVirtualMachine() *VirtualMachine {
	globals := make([]int64, len(m.Globals))
	copy(globals, m.Globals)
	table := make([]uint32, len(m.Table))
	copy(table, m.Table)

	// Load linear memory.
	memory := emptyMemory
	if m.Module.Base.Memory != nil && len(m.Module.Base.Memory.Entries) > 0 {
		initialLimit := int(m.Module.Base.Memory.Entries[0].Limits.Initial)
		if m.Config.MaxMemoryPages != 0 && initialLimit > m.Config.MaxMemoryPages {
			panic("max memory exceeded")
		}

		capacity := initialLimit * DefaultPageSize

		// Initialize empty memory.
		memory = make([]byte, capacity)
		for i := 0; i < capacity; i++ {
			memory[i] = 0
		}

		if m.Module.Base.Data != nil && len(m.Module.Base.Data.Entries) > 0 {
			for _, e := range m.Module.Base.Data.Entries {
				offset := int(execInitExpr(e.Offset, globals))
				copy(memory[offset:], e.Data)
			}
		}
	}

	return &VirtualMachine{
		Module:          m.Module,
		Config:          m.Config,
		FunctionCode:    m.FunctionCode,
		FunctionImports: m.FunctionImports,
		CallStack:       make([]Frame, DefaultCallStackSize),
		CurrentFrame:    -1,
		Table:           table,
		Globals:         globals,
		Memory:          memory,
		Exited:          true,
		GasPolicy:       m.GasPolicy,
		ImportResolver:  m.ImportResolver,
	}
}

// NewVirtualMachine instantiates a virtual machine for a given WebAssembly module, with
// specific execution options specified under a VMConfig, and a WebAssembly module import
// resolver.
func NewVirtualMachine(
	code []byte,
	config VMConfig,
	impResolver ImportResolver,
	gasPolicy compiler.GasPolicy,
) (_retVM *VirtualMachine, retErr error) {
	if config.EnableJIT {
		fmt.Println("Warning: JIT support is removed.")
	}

	m, err := compiler.LoadModule(code)
	if err != nil {
		return nil, err
	}

	m.DisableFloatingPoint = config.DisableFloatingPoint

	functionCode, err := m.CompileForInterpreter(gasPolicy)
	if err != nil {
		return nil, err
	}

	defer utils.CatchPanic(&retErr)

	table := emptyTable
	globals := emptyGlobals
	funcImports := emptyFuncImports

	if m.Base.Import != nil && impResolver != nil {
		for _, imp := range m.Base.Import.Entries {
			switch imp.Type.Kind() {
			case wasm.ExternalFunction:
				funcImports = append(funcImports, FunctionImportInfo{
					ModuleName: imp.ModuleName,
					FieldName:  imp.FieldName,
					F:          nil, // deferred
				})
			case wasm.ExternalGlobal:
				globals = append(globals, impResolver.ResolveGlobal(imp.ModuleName, imp.FieldName))
			case wasm.ExternalMemory:
				// TODO: Do we want a real import?
				if m.Base.Memory != nil && len(m.Base.Memory.Entries) > 0 {
					panic("cannot import another memory while we already have one")
				}
				m.Base.Memory = &wasm.SectionMemories{
					Entries: []wasm.Memory{
						{
							Limits: wasm.ResizableLimits{
								Initial: uint32(config.DefaultMemoryPages),
							},
						},
					},
				}
			case wasm.ExternalTable:
				// TODO: Do we want a real import?
				if m.Base.Table != nil && len(m.Base.Table.Entries) > 0 {
					panic("cannot import another table while we already have one")
				}
				m.Base.Table = &wasm.SectionTables{
					Entries: []wasm.Table{
						{
							Limits: wasm.ResizableLimits{
								Initial: uint32(config.DefaultTableSize),
							},
						},
					},
				}
			default:
				panic(fmt.Errorf("import kind not supported: %d", imp.Type.Kind()))
			}
		}
	}

	// Load global entries.
	for _, entry := range m.Base.GlobalIndexSpace {
		globals = append(globals, execInitExpr(entry.Init, globals))
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
				offset := int(execInitExpr(e.Offset, globals))
				copy(table[offset:], e.Elems)
			}
		}
	}

	// Load linear memory.
	memory := emptyMemory
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
				offset := int(execInitExpr(e.Offset, globals))
				copy(memory[offset:], e.Data)
			}
		}
	}

	return &VirtualMachine{
		Module:          m,
		Config:          config,
		FunctionCode:    functionCode,
		FunctionImports: funcImports,
		CallStack:       make([]Frame, DefaultCallStackSize),
		CurrentFrame:    -1,
		Table:           table,
		Globals:         globals,
		Memory:          memory,
		Exited:          true,
		GasPolicy:       gasPolicy,
		ImportResolver:  impResolver,
	}, nil
}

func (vm *VirtualMachine) SetAOTService(s AOTService) {
	s.Initialize(vm)
	vm.AOTService = s
}

func bSprintf(builder *strings.Builder, format string, args ...interface{}) { // nolint:interfacer
	builder.WriteString(fmt.Sprintf(format, args...))
}

func escapeName(name string) string {
	ret := ""

	for _, ch := range []byte(name) {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			ret += string(ch)
		} else {
			ret += fmt.Sprintf("\\x%02x", ch)
		}
	}

	return ret
}

func filterName(name string) string {
	ret := ""

	for _, ch := range []byte(name) {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			ret += string(ch)
		}
	}

	return ret
}

func (vm *VirtualMachine) GenerateNEnv(config NCompileConfig) string {
	builder := &strings.Builder{}

	bSprintf(builder, "#include <stdint.h>\n\n")

	if config.DisableMemBoundCheck {
		builder.WriteString("#define POLYMERASE_NO_MEM_BOUND_CHECK\n")
	}

	builder.WriteString(compiler.NGEN_HEADER)
	if !vm.Config.DisableFloatingPoint {
		builder.WriteString(compiler.NGEN_FP_HEADER)
	}

	bSprintf(builder, "static uint64_t globals[] = {")
	for _, v := range vm.Globals {
		bSprintf(builder, "%dull,", uint64(v))
	}
	bSprintf(builder, "};\n")

	for i, code := range vm.FunctionCode {
		bSprintf(builder, "uint64_t %s%d(struct VirtualMachine *", compiler.NGEN_FUNCTION_PREFIX, i)
		for j := 0; j < code.NumParams; j++ {
			bSprintf(builder, ",uint64_t")
		}
		bSprintf(builder, ");\n")
	}

	// call_indirect dispatcher.
	bSprintf(builder, "struct TableEntry { uint64_t num_params; void *func; };\n")
	bSprintf(builder, "static const uint64_t num_table_entries = %d;\n", len(vm.Table))
	bSprintf(builder, "static struct TableEntry table[] = {\n")
	for _, entry := range vm.Table {
		if entry == math.MaxUint32 {
			bSprintf(builder, "{ .num_params = 0, .func = 0 },\n")
		} else {
			functionID := int(entry)
			code := vm.FunctionCode[functionID]

			bSprintf(builder, "{ .num_params = %d, .func = %s%d },\n", code.NumParams, compiler.NGEN_FUNCTION_PREFIX, functionID)
		}
	}
	bSprintf(builder, "};\n")
	bSprintf(builder, "static void * __attribute__((always_inline)) %sresolve_indirect(struct VirtualMachine *vm, uint64_t entry_id, uint64_t num_params) {\n", compiler.NGEN_ENV_API_PREFIX)
	bSprintf(builder, "if(entry_id >= num_table_entries) { vm->throw_s(vm, \"%s\"); }\n", "table entry out of bounds")
	bSprintf(builder, "if(table[entry_id].func == 0) { vm->throw_s(vm, \"%s\"); }\n", "table entry is null")
	bSprintf(builder, "if(table[entry_id].num_params != num_params) { vm->throw_s(vm, \"%s\"); }\n", "argument count mismatch")
	bSprintf(builder, "return table[entry_id].func;\n")
	bSprintf(builder, "}\n")

	bSprintf(builder, "struct ImportEntry { const char *module_name; const char *field_name; ExternalFunction f; };\n")
	bSprintf(builder, "static const uint64_t num_import_entries = %d;\n", len(vm.FunctionImports))
	bSprintf(builder, "static struct ImportEntry imports[] = {\n")
	for _, imp := range vm.FunctionImports {
		bSprintf(builder, "{ .module_name = \"%s\", .field_name = \"%s\", .f = 0 },\n", escapeName(imp.ModuleName), escapeName(imp.FieldName))
	}
	bSprintf(builder, "};\n")
	bSprintf(builder,
		"static uint64_t __attribute__((always_inline)) %sinvoke_import(struct VirtualMachine *vm, uint64_t import_id, uint64_t num_params, uint64_t *params) {\n",
		compiler.NGEN_ENV_API_PREFIX,
	)

	bSprintf(builder, "if(import_id >= num_import_entries) { vm->throw_s(vm, \"%s\"); }\n", "import entry out of bounds")
	bSprintf(builder, "if(imports[import_id].f == 0) { imports[import_id].f = vm->resolve_import(vm, imports[import_id].module_name, imports[import_id].field_name); }\n")
	bSprintf(builder, "if(imports[import_id].f == 0) { vm->throw_s(vm, \"%s\"); }\n", "cannot resolve import")
	bSprintf(builder, "return imports[import_id].f(vm, import_id, num_params, params);\n")
	bSprintf(builder, "}\n")

	return builder.String()
}

func (vm *VirtualMachine) NBuildAliasDef() string {
	builder := &strings.Builder{}

	builder.WriteString("// Aliases for exported functions\n")

	if vm.Module.Base.Export != nil {
		for name, exp := range vm.Module.Base.Export.Entries {
			if exp.Kind == wasm.ExternalFunction {
				bSprintf(builder, "#define %sexport_%s %s%d\n", compiler.NGEN_FUNCTION_PREFIX, filterName(name), compiler.NGEN_FUNCTION_PREFIX, exp.Index)
			}
		}
	}

	return builder.String()
}

func (vm *VirtualMachine) NCompile(config NCompileConfig) string {
	body, err := vm.Module.CompileWithNGen(vm.GasPolicy, uint64(len(vm.Globals)))
	if err != nil {
		panic(err)
	}

	out := vm.GenerateNEnv(config) + "\n" + body
	if config.AliasDef {
		out += "\n"
		out += vm.NBuildAliasDef()
	}

	return out
}

// Init initializes a frame. Must be called on `call` and `call_indirect`.
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
	f.Continuation = 0

	//fmt.Printf("Enter function %d (%s)\n", functionID, vm.Module.FunctionNames[functionID])
}

// Destroy destroys a frame. Must be called on return.
func (f *Frame) Destroy(vm *VirtualMachine) {
	numValueSlots := len(f.Regs) + len(f.Locals)
	vm.NumValueSlots -= numValueSlots

	//fmt.Printf("Leave function %d (%s)\n", f.FunctionID, vm.Module.FunctionNames[f.FunctionID])
}

// GetCurrentFrame returns the current frame.
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

// GetGlobalExport returns the global export with the given name.
func (vm *VirtualMachine) GetGlobalExport(key string) (int, bool) {
	return vm.getExport(key, wasm.ExternalGlobal)
}

// GetFunctionExport returns the function export with the given name.
func (vm *VirtualMachine) GetFunctionExport(key string) (int, bool) {
	return vm.getExport(key, wasm.ExternalFunction)
}

// PrintStackTrace prints the entire VM stack trace for debugging.
func (vm *VirtualMachine) PrintStackTrace() {
	fmt.Println("--- Begin stack trace ---")
	for i := vm.CurrentFrame; i >= 0; i-- {
		functionID := vm.CallStack[i].FunctionID
		fmt.Printf("<%d> [%d] %s\n", i, functionID, vm.Module.FunctionNames[functionID])
	}
	fmt.Println("--- End stack trace ---")
}

// Ignite initializes the first call frame.
func (vm *VirtualMachine) Ignite(functionID int, params ...int64) {
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

func (vm *VirtualMachine) AddAndCheckGas(delta uint64) bool {
	newGas := vm.Gas + delta
	if newGas < vm.Gas {
		panic("gas overflow")
	}

	if vm.Config.GasLimit != 0 && newGas > vm.Config.GasLimit {
		if vm.Config.ReturnOnGasLimitExceeded {
			return false
		}

		panic("gas limit exceeded")
	}

	vm.Gas = newGas

	return true
}

// Execute starts the virtual machines main instruction processing loop.
// This function may return at any point and is guaranteed to return
// at least once every 10000 instructions. Caller is responsible for
// detecting VM status in a loop.
func (vm *VirtualMachine) Execute() {
	if vm.Exited {
		panic("attempting to execute an exited vm")
	}

	if vm.Delegate != nil {
		panic("delegate not cleared")
	}

	if vm.InsideExecute {
		panic("vm execution is not re-entrant")
	}
	vm.InsideExecute = true
	vm.GasLimitExceeded = false

	defer func() {
		vm.InsideExecute = false
		if err := recover(); err != nil {
			vm.Exited = true
			vm.ExitError = err
			vm.StackTrace = string(debug.Stack())
		}
	}()

	frame := vm.GetCurrentFrame()

	for {
		valueID := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
		ins := opcodes.Opcode(frame.Code[frame.IP+4])
		frame.IP += 5

		//fmt.Printf("INS: [%d] %s\n", valueID, ins.String())

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
		case opcodes.I32Const:
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
		case opcodes.I64Const:
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

			if c := a + b; c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}

		case opcodes.F32Sub:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := a - b; c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Mul:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := a * b; c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Div:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := a / b; c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Sqrt:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := float32(math.Sqrt(float64(val))); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Min:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := float32(math.Min(float64(a), float64(b))); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Max:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := float32(math.Max(float64(a), float64(b))); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Ceil:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := float32(math.Ceil(float64(val))); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Floor:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := float32(math.Floor(float64(val))); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Trunc:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := float32(math.Trunc(float64(val))); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Nearest:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := float32(math.RoundToEven(float64(val))); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Abs:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := float32(math.Abs(float64(val))); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32Neg:
			val := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := -val; c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
		case opcodes.F32CopySign:
			a := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := float32(math.Copysign(float64(a), float64(b))); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(math.Float32bits(c))
			}
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

			if c := a + b; c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Sub:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := a - b; c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Mul:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := a * b; c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Div:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := a / b; c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Sqrt:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := math.Sqrt(val); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Min:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := math.Min(a, b); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Max:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := math.Max(a, b); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Ceil:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := math.Ceil(val); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Floor:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := math.Floor(val); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Trunc:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := math.Trunc(val); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Nearest:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := math.RoundToEven(val); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Abs:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := math.Abs(val); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64Neg:
			val := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := -val; c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F64CopySign:
			a := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			b := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]))
			frame.IP += 8

			if c := math.Copysign(a, b); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
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

		case opcodes.I32TruncSF32, opcodes.I32TruncUF32:
			v := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := float32(math.Trunc(float64(v))); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(int32(c))
			}
		case opcodes.I32TruncSF64, opcodes.I32TruncUF64:
			v := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := math.Trunc(v); c != c {
				frame.Regs[valueID] = int64(0x7FC00000)
			} else {
				frame.Regs[valueID] = int64(int32(c))
			}
		case opcodes.I64TruncSF32, opcodes.I64TruncUF32:
			v := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := math.Trunc(float64(v)); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(c)
			}
		case opcodes.I64TruncSF64, opcodes.I64TruncUF64:
			v := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := math.Trunc(v); c != c {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(c)
			}
		case opcodes.F32DemoteF64:
			v := math.Float64frombits(uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(v)))

		case opcodes.F64PromoteF32:
			v := math.Float32frombits(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			if c := float64(v); c == math.Float64frombits(0x7FF8000000000000) {
				frame.Regs[valueID] = int64(0x7FF8000000000001)
			} else {
				frame.Regs[valueID] = int64(math.Float64bits(c))
			}
		case opcodes.F32ConvertSI32:
			v := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(v)))

		case opcodes.F32ConvertUI32:
			v := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(v)))

		case opcodes.F32ConvertSI64:
			v := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(v)))

		case opcodes.F32ConvertUI64:
			v := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float32bits(float32(v)))

		case opcodes.F64ConvertSI32:
			v := int32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			frame.IP += 4
			frame.Regs[valueID] = int64(int32(math.Float64bits(float64(v))))

		case opcodes.F64ConvertUI32:
			v := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			frame.IP += 4
			frame.Regs[valueID] = int64(int32(math.Float64bits(float64(v))))

		case opcodes.F64ConvertSI64:
			v := frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float64bits(float64(v)))

		case opcodes.F64ConvertUI64:
			v := uint64(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			frame.IP += 4
			frame.Regs[valueID] = int64(math.Float64bits(float64(v)))

		case opcodes.I64ExtendUI32:
			v := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))])
			frame.IP += 4
			frame.Regs[valueID] = int64(v)

		case opcodes.I64ExtendSI32:
			v := int32(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4
			frame.Regs[valueID] = int64(v)

		case opcodes.I32Load, opcodes.I64Load32U:
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(LE.Uint32(vm.Memory[effective : effective+4]))
		case opcodes.I64Load32S:
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(int32(LE.Uint32(vm.Memory[effective : effective+4])))
		case opcodes.I64Load:
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(LE.Uint64(vm.Memory[effective : effective+8]))
		case opcodes.I32Load8S, opcodes.I64Load8S:
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(int8(vm.Memory[effective]))
		case opcodes.I32Load8U, opcodes.I64Load8U:
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(vm.Memory[effective])
		case opcodes.I32Load16S, opcodes.I64Load16S:
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(int16(LE.Uint16(vm.Memory[effective : effective+2])))
		case opcodes.I32Load16U, opcodes.I64Load16U:
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			frame.IP += 12

			effective := int(uint64(base) + uint64(offset))
			frame.Regs[valueID] = int64(LE.Uint16(vm.Memory[effective : effective+2]))
		case opcodes.I32Store, opcodes.I64Store32:
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			value := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+12:frame.IP+16]))]

			frame.IP += 16

			effective := int(uint64(base) + uint64(offset))
			LE.PutUint32(vm.Memory[effective:effective+4], uint32(value))
		case opcodes.I64Store:
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			value := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+12:frame.IP+16]))]

			frame.IP += 16

			effective := int(uint64(base) + uint64(offset))
			LE.PutUint64(vm.Memory[effective:effective+8], uint64(value))
		case opcodes.I32Store8, opcodes.I64Store8:
			offset := LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8])
			base := uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP+8:frame.IP+12]))])

			value := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+12:frame.IP+16]))]

			frame.IP += 16

			effective := int(uint64(base) + uint64(offset))
			vm.Memory[effective] = byte(value)
		case opcodes.I32Store16, opcodes.I64Store16:
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
		case opcodes.JmpEither:
			targetA := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			targetB := int(LE.Uint32(frame.Code[frame.IP+4 : frame.IP+8]))
			cond := int(LE.Uint32(frame.Code[frame.IP+8 : frame.IP+12]))
			yieldedReg := int(LE.Uint32(frame.Code[frame.IP+12 : frame.IP+16]))
			frame.IP += 16

			vm.Yielded = frame.Regs[yieldedReg]
			if frame.Regs[cond] != 0 {
				frame.IP = targetA
			} else {
				frame.IP = targetB
			}
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
			}

			frame = vm.GetCurrentFrame()
			frame.Regs[frame.ReturnReg] = val
			//fmt.Printf("Return value %d\n", val)
		case opcodes.ReturnVoid:
			frame.Destroy(vm)
			vm.CurrentFrame--
			if vm.CurrentFrame == -1 {
				vm.Exited = true
				vm.ReturnValue = 0
				return
			}

			frame = vm.GetCurrentFrame()
		case opcodes.GetLocal:
			id := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			val := frame.Locals[id]
			frame.IP += 4
			frame.Regs[valueID] = val
			//fmt.Printf("GetLocal %d = %d\n", id, val)
		case opcodes.SetLocal:
			id := int(LE.Uint32(frame.Code[frame.IP : frame.IP+4]))
			val := frame.Regs[int(LE.Uint32(frame.Code[frame.IP+4:frame.IP+8]))]
			frame.IP += 8
			frame.Locals[id] = val
			//fmt.Printf("SetLocal %d = %d\n", id, val)
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
			//fmt.Println("Call params =", frame.Locals[:argCount])

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
				defer func() {
					if err := recover(); err != nil {
						vm.Exited = true
						vm.ExitError = err
					}
				}()
				imp := vm.FunctionImports[importID]
				if imp.F == nil {
					imp.F = vm.ImportResolver.ResolveFunc(imp.ModuleName, imp.FieldName)
				}
				frame.Regs[valueID] = imp.F(vm)
			}

			return
		case opcodes.CurrentMemory:
			frame.Regs[valueID] = int64(len(vm.Memory) / DefaultPageSize)

		case opcodes.GrowMemory:
			n := int(uint32(frame.Regs[int(LE.Uint32(frame.Code[frame.IP:frame.IP+4]))]))
			frame.IP += 4

			current := len(vm.Memory) / DefaultPageSize
			if vm.Config.MaxMemoryPages == 0 || (current+n >= current && current+n <= vm.Config.MaxMemoryPages) {
				frame.Regs[valueID] = int64(current)
				vm.Memory = append(vm.Memory, make([]byte, n*DefaultPageSize)...)
			} else {
				frame.Regs[valueID] = -1
			}

		case opcodes.Phi:
			frame.Regs[valueID] = vm.Yielded

		case opcodes.AddGas:
			delta := LE.Uint64(frame.Code[frame.IP : frame.IP+8])
			frame.IP += 8
			if !vm.AddAndCheckGas(delta) {
				vm.GasLimitExceeded = true
				return
			}

		case opcodes.FPDisabledError:
			panic("wasm: floating point disabled")

		default:
			panic("unknown instruction")
		}
	}
}
