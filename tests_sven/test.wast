(module
    (func $main (export "app_main") (result i32)
        (local i32)
        (local i32)
        (local i32)

        (i32.const 0)
        (set_local 0)

        (i32.const 1)
        (set_local 1)

        (i32.const 2)
        (set_local 2)

        (i32.const 1)
        (get_local 1)
        (i32.add)
    )
)
