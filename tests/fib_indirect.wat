(module
    (type $ty_fib (func (param i32) (result i32)))
    (table 2 anyfunc)
    (elem (i32.const 0) 1)

    (func (result i32)
        i32.const 35
        i32.const 0
        call_indirect (type $ty_fib)
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
        i32.const 0
        call_indirect (type $ty_fib)

        i32.const -2
        get_local 0
        i32.add
        i32.const 0
        call_indirect (type $ty_fib)

        i32.add
    )
)
