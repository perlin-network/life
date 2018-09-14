(module
    (func $main (export "app_main") (result i32)
        (local i32)

        (i32.const 1)
        (i32.const 2)
        (i32.add)
        (set_local 0)

        (i32.const 1)
    )
)
