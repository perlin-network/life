#pragma once

#include <stdlib.h>
#include "vm_def.h"

static int need_mem_bound_check() {
    return 1;
}

static void __x_grow_memory(struct VirtualMachine *vm, uint64_t inc_size) {
    if(vm->mem_size + inc_size < vm->mem_size) {
        vm->throw_s(vm, "memory size overflow");
    }
    go_vm_pre_notify_grow_memory(vm, inc_size);
    vm->mem_size += inc_size;
    vm->mem = realloc(vm->mem, vm->mem_size);
    go_vm_post_notify_grow_memory(vm);
}

static void vm_build(struct VirtualMachine *vm, uintptr_t managed_vm, uint64_t mem_size) {
    vm->throw_s = go_vm_throw_s;
    vm->resolve_import = go_vm_resolve_import;
    vm->mem_size = mem_size;
    vm->mem = malloc(mem_size);
    vm->grow_memory = __x_grow_memory;
    vm->userdata = (void *) managed_vm;
}

static void vm_destroy(struct VirtualMachine *vm) {
    if(vm->mem) free(vm->mem);
}

static uintptr_t vm_get_managed(struct VirtualMachine *vm) {
    return (uintptr_t) vm->userdata;
}

static uint64_t vm_execute(struct VirtualMachine *vm, uint64_t (*f)(struct VirtualMachine *, void *), void *userdata) {
    return f(vm, userdata);
}
