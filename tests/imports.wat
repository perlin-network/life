(module
    (export "app_main" (func $app_main))
    (import "env" "__life_ping" (func (param i32) (result i32)))
    (func $app_main (result i32)
        i32.const 42
        call 0
    )
)
