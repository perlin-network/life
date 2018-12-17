package platform

/*
#include <dlfcn.h>
#include <stdlib.h>
#include <stdint.h>

typedef const char const_char;

struct VirtualMachine;
typedef uint64_t (*ExternalFunction)(struct VirtualMachine *vm, uint64_t import_id, uint64_t num_params, uint64_t *params);
struct VirtualMachine {
	void (*throw_s)(struct VirtualMachine *vm, const char *s);
	ExternalFunction (*resolve_import)(struct VirtualMachine *vm, const char *module_name, const char *field_name);
	uint64_t mem_size;
	uint8_t *mem;
	void (*grow_memory)(struct VirtualMachine *vm, uint64_t inc_size);
	void *userdata;
};

void go_vm_throw_s(struct VirtualMachine *vm, const char *s);
ExternalFunction go_vm_resolve_import(struct VirtualMachine *vm, const char *module_name, const char *field_name);
void go_vm_grow_memory(struct VirtualMachine *vm, uint64_t inc_size);
uint64_t go_vm_dispatch_import_invocation(struct VirtualMachine *vm, uint64_t import_id, uint64_t num_params, uint64_t *params);

static uint64_t unsafe_invoke_function_0(void *sym, uintptr_t managed_vm, uint8_t *mem, uint64_t mem_size) {
	uint64_t (*f)(struct VirtualMachine *vm) = sym;
	struct VirtualMachine vm = {
		.throw_s = go_vm_throw_s,
		.resolve_import = go_vm_resolve_import,
		.mem_size = mem_size,
		.mem = mem,
		.grow_memory = go_vm_grow_memory,
		.userdata = (void *) managed_vm,
	};
	return f(&vm);
}
*/
import "C"

import (
	"github.com/perlin-network/life/exec"
	"io/ioutil"
	"log"
	os_exec "os/exec"
	"path"
	"reflect"
	"runtime"
	"unsafe"
)

//export go_vm_throw_s
func go_vm_throw_s(vm *C.struct_VirtualMachine, s *C.const_char) {
	gs := C.GoString(s)
	panic(gs)
}

//export go_vm_resolve_import
func go_vm_resolve_import(vm *C.struct_VirtualMachine, moduleName *C.const_char, fieldName *C.const_char) C.ExternalFunction {
	return C.ExternalFunction(C.go_vm_dispatch_import_invocation)
}

//export go_vm_dispatch_import_invocation
func go_vm_dispatch_import_invocation(vm *C.struct_VirtualMachine, importID C.uint64_t, numParams C.uint64_t, params *C.uint64_t) C.uint64_t {
	managedVM := (*exec.VirtualMachine)(vm.userdata)

	imp := &managedVM.FunctionImports[importID]
	if imp.F == nil {
		imp.F = managedVM.ImportResolver.ResolveFunc(imp.ModuleName, imp.FieldName)
	}

	managedVM.CurrentFrame = 0

	localsSlice := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(params)),
		Len:  int(numParams),
		Cap:  int(numParams),
	}
	managedVM.GetCurrentFrame().Locals = *(*[]int64)(unsafe.Pointer(&localsSlice)) // very unsafe - should we just allocate a new slice?
	return C.uint64_t(imp.F(managedVM))
}

//export go_vm_grow_memory
func go_vm_grow_memory(vm *C.struct_VirtualMachine, incSize uint64) {
	managedVM := (*exec.VirtualMachine)(vm.userdata)

	managedVM.Memory = append(managedVM.Memory, make([]byte, int(incSize))...)
	vm.mem_size = (C.uint64_t)(uint64(len(managedVM.Memory)))
	vm.mem = (*C.uint8_t)(&managedVM.Memory[0])
}

type AOTContext struct {
	dlHandle unsafe.Pointer
}

func (c *AOTContext) UnsafeInvokeFunction_0(vm *exec.VirtualMachine, name string) uint64 {
	nameC := C.CString(name)
	sym := C.dlsym(c.dlHandle, nameC)
	C.free(unsafe.Pointer(nameC))

	if sym == nil {
		panic("function not found")
	}

	return uint64(C.unsafe_invoke_function_0(
		sym,
		C.uintptr_t(uintptr(unsafe.Pointer(vm))),
		(*C.uint8_t)(&vm.Memory[0]),
		(C.uint64_t)(uint64(len(vm.Memory))),
	))
}

func FullAOTCompile(vm *exec.VirtualMachine) *AOTContext {
	code := vm.NCompile(exec.NCompileConfig{AliasDef: false})
	tempDir, err := ioutil.TempDir("", "life-aot-")
	if err != nil {
		panic(err)
	}

	inPath := path.Join(tempDir, "in.c")
	outPath := path.Join(tempDir, "out")

	err = ioutil.WriteFile(inPath, []byte(code), 0644)
	if err != nil {
		panic(err)
	}

	cmd := os_exec.Command("cc", "-O2", "-o", outPath, "-shared", inPath)
	out, err := cmd.CombinedOutput()

	if len(out) > 0 {
		log.Printf("compiler warnings/errors: \n%s\n", string(out))
	}

	if err != nil {
		panic(err)
	}

	outPathC := C.CString(outPath)
	handle := C.dlopen(outPathC, C.RTLD_NOW|C.RTLD_LOCAL)
	C.free(unsafe.Pointer(outPathC))
	if handle == nil {
		panic("unable to open compiled code")
	}

	ctx := &AOTContext{
		dlHandle: handle,
	}

	runtime.SetFinalizer(ctx, func(ctx *AOTContext) {
		C.dlclose(ctx.dlHandle)
	})

	return ctx
}
