// +build !android

package platform

/*
#cgo LDFLAGS: -ldl -lpthread

#include "vm_def.h"
#include <string.h>

#if defined(__linux__) && defined(__x86_64__)
#include "runtime_linux_x86_64.h"
//#include "runtime_generic.h"
#else
#include "runtime_generic.h"
#endif

#include <dlfcn.h>
#include <stdlib.h>
#include <stdint.h>

typedef const char const_char;

struct InvokeInfo {
	void *sym;
	int num_params;
	const uint64_t *params;
};

static uint64_t __do_invoke(struct VirtualMachine *vm, void *_info) {
	struct InvokeInfo *info = _info;

	switch(info->num_params) {
		case 0: {
			uint64_t (*f)(struct VirtualMachine *vm) = info->sym;
			return f(vm);
		}
		case 1: {
			uint64_t (*f)(struct VirtualMachine *vm, uint64_t) = info->sym;
			return f(vm, info->params[0]);
		}
		case 2: {
			uint64_t (*f)(struct VirtualMachine *vm, uint64_t, uint64_t) = info->sym;
			return f(vm, info->params[0], info->params[1]);
		}
		default: {
			abort();
		}
	}
}

static uint64_t unsafe_invoke_function_0(struct VirtualMachine *vm, void *sym) {
	struct InvokeInfo ii = {
		.sym = sym,
		.num_params = 0,
		.params = 0,
	};
	return vm_execute(vm, __do_invoke, &ii);
}

static uint64_t unsafe_invoke_function_1(struct VirtualMachine *vm, void *sym, uint64_t p0) {
	uint64_t params[] = {p0};
	struct InvokeInfo ii = {
		.sym = sym,
		.num_params = 1,
		.params = params,
	};
	return vm_execute(vm, __do_invoke, &ii);
}

static uint64_t unsafe_invoke_function_2(struct VirtualMachine *vm, void *sym, uint64_t p0, uint64_t p1) {
	uint64_t params[] = {p0, p1};
	struct InvokeInfo ii = {
		.sym = sym,
		.num_params = 2,
		.params = params,
	};
	return vm_execute(vm, __do_invoke, &ii);
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
	managedVM := (*exec.VirtualMachine)(unsafe.Pointer(uintptr(C.vm_get_managed(vm))))

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

//export go_vm_pre_notify_grow_memory
func go_vm_pre_notify_grow_memory(vm *C.struct_VirtualMachine, incSize C.uint64_t) {

}

//export go_vm_post_notify_grow_memory
func go_vm_post_notify_grow_memory(vm *C.struct_VirtualMachine) {
	updateMemory(vm)
}

func updateMemory(vm *C.struct_VirtualMachine) {
	managedVM := (*exec.VirtualMachine)(unsafe.Pointer(uintptr(C.vm_get_managed(vm))))
	memorySlice := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(vm.mem)),
		Len:  int(vm.mem_size),
		Cap:  int(vm.mem_size),
	}
	managedVM.Memory = *(*[]byte)(unsafe.Pointer(&memorySlice))
}

type AOTContext struct {
	dlHandle unsafe.Pointer
	vmHandle *C.struct_VirtualMachine
}

func (c *AOTContext) resolveNameForInvocation(name string) unsafe.Pointer {
	nameC := C.CString(name)
	sym := C.dlsym(c.dlHandle, nameC)
	C.free(unsafe.Pointer(nameC))

	if sym == nil {
		panic("function not found")
	}

	return sym
}

func (c *AOTContext) UnsafeInvokeFunction_0(vm *exec.VirtualMachine, name string) uint64 {
	return uint64(C.unsafe_invoke_function_0(
		c.vmHandle,
		c.resolveNameForInvocation(name),
	))
}

func (c *AOTContext) UnsafeInvokeFunction_1(vm *exec.VirtualMachine, name string, p0 uint64) uint64 {
	return uint64(C.unsafe_invoke_function_1(
		c.vmHandle,
		c.resolveNameForInvocation(name),
		C.uint64_t(p0),
	))
}

func (c *AOTContext) UnsafeInvokeFunction_2(vm *exec.VirtualMachine, name string, p0, p1 uint64) uint64 {
	return uint64(C.unsafe_invoke_function_2(
		c.vmHandle,
		c.resolveNameForInvocation(name),
		C.uint64_t(p0),
		C.uint64_t(p1),
	))
}

func FullAOTCompile(vm *exec.VirtualMachine) *AOTContext {
	code := vm.NCompile(exec.NCompileConfig{
		AliasDef:             false,
		DisableMemBoundCheck: C.need_mem_bound_check() == 0,
	})
	tempDir, err := ioutil.TempDir("", "life-aot-")
	if err != nil {
		log.Println(err)
		return nil
	}

	inPath := path.Join(tempDir, "in.c")
	outPath := path.Join(tempDir, "out")

	err = ioutil.WriteFile(inPath, []byte(code), 0644)
	if err != nil {
		log.Println(err)
		return nil
	}

	cmd := os_exec.Command("clang", "-fPIC", "-O2", "-o", outPath, "-shared", inPath)
	out, err := cmd.CombinedOutput()

	if len(out) > 0 {
		log.Printf("compiler warnings/errors: \n%s\n", string(out))
	}

	if err != nil {
		log.Println(err)
		return nil
	}

	outPathC := C.CString(outPath)
	handle := C.dlopen(outPathC, C.RTLD_NOW|C.RTLD_LOCAL)
	C.free(unsafe.Pointer(outPathC))
	if handle == nil {
		log.Println("unable to open compiled code")
		return nil
	}

	nativeVM := C.vm_alloc()
	C.vm_build(nativeVM, C.uintptr_t(uintptr(unsafe.Pointer(vm))), C.uint64_t(len(vm.Memory)))
	if len(vm.Memory) > 0 {
		C.memcpy(unsafe.Pointer(nativeVM.mem), unsafe.Pointer(&vm.Memory[0]), C.ulong(len(vm.Memory)))
	}

	updateMemory(nativeVM)

	ctx := &AOTContext{
		dlHandle: handle,
		vmHandle: nativeVM,
	}

	runtime.SetFinalizer(ctx, func(ctx *AOTContext) {
		C.dlclose(ctx.dlHandle)
		C.vm_destroy(ctx.vmHandle)
		C.free(unsafe.Pointer(ctx.vmHandle))
	})

	return ctx
}
