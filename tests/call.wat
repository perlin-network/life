(module
    (func
        (i32.const 10)
        (call 1)
        (return)
    )
    (func (param i32) (result i32)
        get_local 0
        i32.const 42
        i32.add
    )
)
