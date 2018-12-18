#pragma once

#include <stdint.h>
#include <stdlib.h>

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
void go_vm_pre_notify_grow_memory(struct VirtualMachine *vm, uint64_t inc_size);
void go_vm_post_notify_grow_memory(struct VirtualMachine *vm);
uint64_t go_vm_dispatch_import_invocation(struct VirtualMachine *vm, uint64_t import_id, uint64_t num_params, uint64_t *params);

static struct VirtualMachine * vm_alloc() {
    return (struct VirtualMachine *) malloc(sizeof(struct VirtualMachine));
}
