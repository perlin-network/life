(module
    (func (local i32)
        (block
            (loop
                (i32.const 42)
                (set_local 0)
                (br 1)
            )
        )
        (get_local 0)
        (return)
    )
)
