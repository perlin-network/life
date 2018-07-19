(module
    (func (result i32) (local i32)
        (i32.const 0)
        (set_local 0)
        (if (result i32) (get_local 0)
            (then
                i32.const 1
            )
            (else
                i32.const 99
                i32.const 1
                br_if 0
                drop
                i32.const 2
            )
        )
        return
    )
)
