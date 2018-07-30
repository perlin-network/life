package exec

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

typedef long long i64;
typedef int i32;
typedef unsigned long long u64;
typedef unsigned int u32;
typedef unsigned char u8;

typedef i32 (*EntryFunc)(
	i64 *regs,
	i64 *locals,
	i64 *globals,
	u8 *memory,
	i64 memory_len,
	i64 *yielded,
	i32 continuation,
	i64 *ret
);

static i32 invoke_entry(
	void *entry,
	i64 *regs,
	i64 *locals,
	i64 *globals,
	u8 *memory,
	i64 memory_len,
	i64 *yielded,
	i32 continuation,
	i64 *ret
) {
	EntryFunc f = entry;
	return f(
		regs,
		locals,
		globals,
		memory,
		memory_len,
		yielded,
		continuation,
		ret
	);
}
*/
import "C"

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"unsafe"
)

var entryName = C.CString("run")

type DynamicModule struct {
	dlHandle unsafe.Pointer
	entry    unsafe.Pointer
}

func (m *DynamicModule) unsafeCleanup() {
	C.dlclose(m.dlHandle)
}

func (m *DynamicModule) Run(vm *VirtualMachine, ret *int64) int32 {
	frame := vm.GetCurrentFrame()

	var regs *C.i64
	var locals *C.i64
	var globals *C.i64
	var memory *C.u8

	if len(frame.Regs) > 0 {
		regs = (*C.i64)(&frame.Regs[0])
	}
	if len(frame.Locals) > 0 {
		locals = (*C.i64)(&frame.Locals[0])
	}
	if len(vm.Globals) > 0 {
		globals = (*C.i64)(&vm.Globals[0])
	}
	if len(vm.Memory) > 0 {
		memory = (*C.u8)(&vm.Memory[0])
	}
	return int32(C.invoke_entry(
		m.entry,
		regs,
		locals,
		globals,
		memory,
		C.i64(len(vm.Memory)),
		(*C.i64)(&vm.Yielded),
		(C.i32)(frame.Continuation),
		(*C.i64)(ret),
	))
}

func CompileDynamicModule(source string) *DynamicModule {
	tempFile, err := ioutil.TempFile("", "life-jit")
	if err != nil {
		panic(err)
	}
	tempFileName := tempFile.Name()
	tempFile.Close()

	defer os.Remove(tempFileName)

	cmd := exec.Command("cc", "-O1", "-fPIC", "-x", "c", "-shared", "-o", tempFileName, "-")
	cmd.Stdin = bytes.NewReader([]byte(source))
	_, err = cmd.Output()
	if err != nil {
		err := err.(*exec.ExitError)
		if err.Stderr != nil {
			panic(string(err.Stderr))
		} else {
			panic(err)
		}
	}

	tempFilePathC := C.CString(tempFileName)
	dlHandle := C.dlopen(tempFilePathC, C.RTLD_LAZY)
	C.free(unsafe.Pointer(tempFilePathC))

	if dlHandle == nil {
		panic("dlopen failed")
	}

	entry := C.dlsym(dlHandle, entryName)
	if entry == nil {
		C.dlclose(dlHandle)
		panic("dlsym failed")
	}

	dm := &DynamicModule{
		dlHandle: dlHandle,
		entry:    entry,
	}

	runtime.SetFinalizer(dm, func(dm *DynamicModule) {
		dm.unsafeCleanup()
	})
	return dm
}
