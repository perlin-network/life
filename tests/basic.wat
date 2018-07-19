(module
    (func (local i32)
        (loop
            (i32.const 42)
            (set_local 0)
        )
        (loop (result i32)
            (get_local 0)
        )
        (return)
    )
)
