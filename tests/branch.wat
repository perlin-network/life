(module
    (func (result i32)
        (block
            (i32.const 33)
            (drop)
            (i32.const 1)
            (br_if 0)
        )
        (block (result i32)
            (i32.const 42)
            (i32.const 1)
            (br_if 0)
            (drop)
            (i32.const 11)
        )
        return
    )
)
