#ifndef CODE
#error "CODE required"
#endif

#ifndef MAIN
#error "MAIN required"
#endif

#include CODE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static void __attribute__((noreturn)) throw_s(struct VirtualMachine *vm, const char *s) {
    fprintf(stderr, "ERROR: %s\n", s);
    abort();
}

static uint64_t _vm_print_i64(struct VirtualMachine *vm, uint64_t import_id, uint64_t num_params, uint64_t *params) {
    if(num_params != 1) {
        throw_s(vm, "invalid params for print_i64");
    }
    printf("print_i64: %lld\n", * (int64_t *) &params[0]);
    return 0;
}

static uint64_t _vm_print_f64(struct VirtualMachine *vm, uint64_t import_id, uint64_t num_params, uint64_t *params) {
    if(num_params != 1) {
        throw_s(vm, "invalid params for print_f64");
    }
    printf("print_f64: %lf\n", * (double *) &params[0]);
    return 0;
}

static uint64_t _vm_identity_map_f64(struct VirtualMachine *vm, uint64_t import_id, uint64_t num_params, uint64_t *params) {
    if(num_params != 1) {
        throw_s(vm, "invalid params for identity_map_f64");
    }
    return params[0];
}

static ExternalFunction resolve_import(struct VirtualMachine *vm, const char *module_name, const char *field_name) {
    if(strcmp(module_name, "env") != 0) {
        return NULL;
    }
    if(strcmp(field_name, "print_f64") == 0) {
        return _vm_print_f64;
    } else if(strcmp(field_name, "print_i64") == 0) {
        return _vm_print_i64;
    } else if(strcmp(field_name, "identity_map_f64") == 0) {
        return _vm_identity_map_f64;
    } else {
        return NULL;
    }
}

static void grow_memory(struct VirtualMachine *vm, uint64_t inc_size) {
    if(vm->mem_size + inc_size < vm->mem_size) {
        vm->throw_s(vm, "memory size overflow");
    }
    vm->mem_size += inc_size;
    vm->mem = realloc(vm->mem, vm->mem_size);
}

int main() {
    struct VirtualMachine vm;
    vm.throw_s = throw_s;
    vm.resolve_import = resolve_import;
    vm.mem_size = 65536 * 128;
    vm.mem = malloc(vm.mem_size);
    vm.grow_memory = grow_memory;
    MAIN(&vm);
    return 0;
}
