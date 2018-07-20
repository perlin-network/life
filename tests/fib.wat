(module
    (func (result i32)
        i32.const 35
        call 1
    )
    (func (param i32) (result i32)
        i32.const 1
        i32.const 1
        get_local 0
        i32.eq
        br_if 0

        i32.const 2
        get_local 0
        i32.eq
        br_if 0

        drop

        i32.const -1
        get_local 0
        i32.add
        call 1

        i32.const -2
        get_local 0
        i32.add
        call 1

        i32.add
    )
)
