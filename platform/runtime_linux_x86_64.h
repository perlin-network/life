#pragma once

#include <pthread.h>
#include <stdlib.h>
#include <unistd.h>
#include <signal.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <sys/ioctl.h>
#include <sys/syscall.h>
#include <linux/userfaultfd.h>
#include <errno.h>
#include <stdio.h>
#include <setjmp.h>
#include "vm_def.h"

static const unsigned long MMAP_SIZE = 1024ul * 1024 * 1024 * 9;
static const unsigned long STACK_SIZE = 65536;
static const unsigned long SIG42_SPECIAL_STACK_SIZE = 8192;

struct LinuxRuntimeInfo {
    struct VirtualMachine *vm;
    uintptr_t managed_vm;
    int uffd;
    pthread_t mon_thread;
    pthread_t exec_thread;
    int in_exec;
    jmp_buf recovery_env;
    const char *pending_error;
    unsigned long current_stack_size;
    unsigned long page_size;
};

struct DelegateExecutionContext {
    struct VirtualMachine *vm;
    uint64_t (*f)(struct VirtualMachine *vm, void *);
    void *userdata;
    uint64_t result;
};

static int need_mem_bound_check() {
    return 0;
}

static void __x_grow_memory(struct VirtualMachine *vm, uint64_t inc_size) {
    // No concurrent call of __x_grow_memory is allowed.
    // Otherwise there will be a race condition (TOCTOU).
    if(vm->mem_size + inc_size < vm->mem_size) {
        vm->throw_s(vm, "memory size overflow");
    }
    go_vm_pre_notify_grow_memory(vm, inc_size);
    __atomic_fetch_add(&vm->mem_size, inc_size, __ATOMIC_RELAXED);
    go_vm_post_notify_grow_memory(vm);
}

static void * __x_mon_thread(void *__arg) {
    struct LinuxRuntimeInfo *rt_info = __arg;
    while(1) {
        struct uffd_msg msg;
        if(read(rt_info->uffd, &msg, sizeof(struct uffd_msg)) <= 0) {
            abort();
        }
        if(msg.event != UFFD_EVENT_PAGEFAULT) {
            abort();
        }

        unsigned long page_begin = (unsigned long) msg.arg.pagefault.address & (~(rt_info->page_size - 1));
        unsigned long page_end = page_begin + rt_info->page_size;

        int in_stack = (
            page_begin >= (unsigned long) rt_info->vm->mem + MMAP_SIZE - rt_info->current_stack_size
            && page_end <= (unsigned long) rt_info->vm->mem + MMAP_SIZE
        );

        int in_stack_ext = (
            page_begin >= (unsigned long) rt_info->vm->mem + MMAP_SIZE - STACK_SIZE - SIG42_SPECIAL_STACK_SIZE
            && page_end <= (unsigned long) rt_info->vm->mem + MMAP_SIZE
        );

        unsigned long mem_size = __atomic_load_n(&rt_info->vm->mem_size, __ATOMIC_RELAXED);
        if(
            page_end < page_begin // overflow
            || page_begin < (unsigned long) rt_info->vm->mem // out of bounds (1)
            || (!in_stack_ext && page_end - (unsigned long) rt_info->vm->mem > mem_size) // out of bounds (2)
        ) {
            if(rt_info->in_exec) {
                pthread_kill(rt_info->exec_thread, 42);
            } else {
                abort();
            }
            continue;
        }

        struct uffdio_zeropage zeropage_req;

        zeropage_req.range.start = page_begin;
        zeropage_req.range.len = rt_info->page_size;
        zeropage_req.mode = 0;
        if(ioctl(rt_info->uffd, UFFDIO_ZEROPAGE, &zeropage_req) < 0) {
            abort();
        }

        if(in_stack_ext && !in_stack) {
            if(rt_info->in_exec) {
                rt_info->current_stack_size = STACK_SIZE + SIG42_SPECIAL_STACK_SIZE;
                //printf("KILL 42\n");
                pthread_kill(rt_info->exec_thread, 42);
            } else {
                abort();
            }
        }
    }
}

static uint8_t * __x_setup_memory(struct VirtualMachine *vm) {
    struct LinuxRuntimeInfo *rt_info = vm->userdata;

    uint8_t *region = (uint8_t *) mmap(NULL, MMAP_SIZE, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS | MAP_NORESERVE, -1, 0);
    if(region == MAP_FAILED) {
        vm->throw_s(vm, "cannot setup memory mapping");
    }

    int uffd = syscall(__NR_userfaultfd, O_CLOEXEC);
    if(uffd < 0) {
        munmap(region, MMAP_SIZE);
        vm->throw_s(vm, "cannot setup userfaultfd");
    }

    struct uffdio_api uffdio_api_config;
    struct uffdio_register uffdio_register_config;

    uffdio_api_config.api = UFFD_API;
    uffdio_api_config.features = 0;

    if(ioctl(uffd, UFFDIO_API, &uffdio_api_config) < 0) {
        close(uffd);
        munmap(region, MMAP_SIZE);
        vm->throw_s(vm, "cannot initialize userfaultfd");
    }

    uffdio_register_config.range.start = (unsigned long) region;
    uffdio_register_config.range.len = MMAP_SIZE;
    uffdio_register_config.mode = UFFDIO_REGISTER_MODE_MISSING;

    if(ioctl(uffd, UFFDIO_REGISTER, &uffdio_register_config) < 0) {
        close(uffd);
        munmap(region, MMAP_SIZE);
        vm->throw_s(vm, "cannot register userfaultfd");
    }

    rt_info->uffd = uffd;

    pthread_t mon_thread;
    if(pthread_create(&mon_thread, NULL, __x_mon_thread, rt_info)) {
        close(uffd);
        munmap(region, MMAP_SIZE);
        vm->throw_s(vm, "cannot start monitor thread");
    }

    rt_info->mon_thread = mon_thread;
    vm->mem = region;
}

static void vm_build(struct VirtualMachine *vm, uintptr_t managed_vm, uint64_t mem_size) {
    vm->throw_s = go_vm_throw_s;
    vm->resolve_import = go_vm_resolve_import;
    vm->mem_size = mem_size;
    vm->grow_memory = __x_grow_memory;

    struct LinuxRuntimeInfo *rt_info = malloc(sizeof(struct LinuxRuntimeInfo));
    rt_info->vm = vm;
    rt_info->managed_vm = managed_vm;
    rt_info->page_size = sysconf(_SC_PAGE_SIZE);
    rt_info->in_exec = 0;
    rt_info->current_stack_size = STACK_SIZE;
    rt_info->pending_error = NULL;
    vm->userdata = rt_info;

    __x_setup_memory(vm);
}

static void vm_destroy(struct VirtualMachine *vm) {
    struct LinuxRuntimeInfo *rt_info = vm->userdata;
    pthread_cancel(rt_info->mon_thread);
    pthread_join(rt_info->mon_thread, NULL);
    close(rt_info->uffd);
    munmap(vm->mem, MMAP_SIZE);
}

static __thread struct VirtualMachine *current_vm = NULL;

static void __x_handle_sig42(int signum) {
    if(!current_vm) abort();
    current_vm->throw_s(current_vm, "access violation");
}

static void __x_throw_s(struct VirtualMachine *vm, const char *s) {
    struct LinuxRuntimeInfo *rt_info = vm->userdata;
    rt_info->pending_error = s;
    longjmp(rt_info->recovery_env, 1);
}

#ifdef __x86_64__
asm(
    "__x_switch_stack:\n"
    "push %rbx\n"
    "mov %rsp, %rbx\n"
    "mov %rdi, %rsp\n"
    "mov %rdx, %rdi\n"
    "call *%rsi\n"
    "mov %rbx, %rsp\n"
    "pop %rbx\n"
    "ret\n"
);
#else
#error "stack switching is not supported for your architecture"
#endif

uint64_t __x_switch_stack(void *stack, uint64_t (*f)(struct DelegateExecutionContext *ctx), struct DelegateExecutionContext *ctx);

static uint64_t perform_delegate_execution(struct DelegateExecutionContext *ctx) {
    return ctx->f(ctx->vm, ctx->userdata);
}

static void * __x_execute_delegate(void *__arg) {
    struct DelegateExecutionContext *ctx = __arg;
    struct LinuxRuntimeInfo *rt_info = ctx->vm->userdata;

    signal(42, __x_handle_sig42);
    current_vm = ctx->vm;

    ctx->vm->throw_s = __x_throw_s;
    rt_info->pending_error = NULL;

    if(setjmp(rt_info->recovery_env) == 0) {
        ctx->result = __x_switch_stack((void *) ((unsigned long) ctx->vm->mem + MMAP_SIZE), perform_delegate_execution, ctx);
    } else {
        ctx->result = 0;
    }
    ctx->vm->throw_s = go_vm_throw_s;

    current_vm = NULL;
    return NULL;
}

static uint64_t vm_execute(struct VirtualMachine *vm, uint64_t (*f)(struct VirtualMachine *, void *), void *userdata) {
    struct LinuxRuntimeInfo *rt_info = vm->userdata;
    struct DelegateExecutionContext *ctx = malloc(sizeof(struct DelegateExecutionContext));
    ctx->vm = vm;
    ctx->f = f;
    ctx->userdata = userdata;

    rt_info->in_exec = 1;

    if(pthread_create(&rt_info->exec_thread, NULL, __x_execute_delegate, ctx)) {
        rt_info->in_exec = 0;
        free(ctx);

        vm->throw_s(vm, "cannot create thread for execution");
    }

    pthread_join(rt_info->exec_thread, NULL);

    rt_info->in_exec = 0;
    free(ctx);

    if(rt_info->pending_error) {
        vm->throw_s(vm, rt_info->pending_error);
    }

    return ctx->result;
}

static uintptr_t vm_get_managed(struct VirtualMachine *vm) {
    return ((struct LinuxRuntimeInfo *) vm->userdata)->managed_vm;
}
