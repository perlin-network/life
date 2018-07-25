(module
    (export "app_main" (func $app_main))
    (import "env" "__life_ping" (func $ping (param i32) (result i32)))
    (import "env" "__life_magic" (global $magic i32))
    (global $v i32 (i32.const 9))
    (func $app_main (result i32)
        i32.const 42
        get_global $magic
        get_global $v
        i32.add
        i32.add
        call $ping ;; 476
    )
)
