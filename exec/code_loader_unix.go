package exec

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/syscall.h>

#define __NR_memfd_create 319 // https://code.woboq.org/qt5/include/asm/unistd_64.h.html

typedef long long i64;
typedef int i32;
typedef i32 (*EntryFunc)(i64 *regs, i64 *locals, i64 *yielded, i32 continuation, i64 *ret);

static int memfd_create(const char *name, unsigned int flags) {
	return syscall(__NR_memfd_create, name, flags);
}
static i32 invoke_entry(void *entry, i64 *regs, i64 *locals, i64 *yielded, i32 continuation, i64 *ret) {
	EntryFunc f = entry;
	return f(regs, locals, yielded, continuation, ret);
}
*/
import "C"

import (
	"fmt"
	"bytes"
	"os/exec"
	"encoding/base64"
	"crypto/rand"
	"unsafe"
	"os"
)

var entryName = C.CString("run")

type DynamicModule struct {
	shmFd C.int
	dlHandle unsafe.Pointer
	entry unsafe.Pointer
}

func (m *DynamicModule) Destroy() {
	C.dlclose(m.dlHandle)
	C.close(m.shmFd)
}

func (m *DynamicModule) Run(vm *VirtualMachine, ret *int64) int32 {
	frame := vm.GetCurrentFrame()

	var regs *C.i64
	var locals *C.i64

	if len(frame.Regs) > 0 {
		regs = (*C.i64)(&frame.Regs[0])
	}
	if len(frame.Locals) > 0 {
		locals = (*C.i64)(&frame.Locals[0])
	}
	return int32(C.invoke_entry(
		m.entry,
		regs,
		locals,
		(*C.i64)(&vm.Yielded),
		(C.i32)(frame.Continuation),
		(*C.i64)(ret),
	))
}

func CompileDynamicModule(source string) *DynamicModule {
	cmd := exec.Command("cc", "-x", "c", "-shared", "-o", "/dev/stdout", "-")
	cmd.Stdin = bytes.NewReader([]byte(source))
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	shmNameRaw := make([]byte, 16)
	_, err = rand.Read(shmNameRaw)
	if err != nil {
		panic(err)
	}
	shmName := base64.StdEncoding.EncodeToString(shmNameRaw)

	shmNameC := C.CString(shmName)
	shmFd := C.memfd_create(shmNameC, 1)
	C.free(unsafe.Pointer(shmNameC))

	if shmFd < 0 {
		panic("unable to create shm fd")
	}

	n := C.write(shmFd, unsafe.Pointer(&out[0]), C.ulong(len(out)))
	if n < 0 {
		C.close(shmFd)
		panic("write failed")
	}

	shmPath := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), shmFd)
	shmPathC := C.CString(shmPath)
	dlHandle := C.dlopen(shmPathC, C.RTLD_LAZY)
	C.free(unsafe.Pointer(shmPathC))

	if dlHandle == nil {
		C.close(shmFd)
		panic("dlopen failed")
	}

	entry := C.dlsym(dlHandle, entryName)
	if entry == nil {
		C.dlclose(dlHandle)
		C.close(shmFd)
		panic("dlsym failed")
	}

	return &DynamicModule {
		shmFd: shmFd,
		dlHandle: dlHandle,
		entry: entry,
	}
}
