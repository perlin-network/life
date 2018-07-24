(module
    (func (result i64)
        i32.const -1
        i64.extend_u/i32 ;; 4294967295
        i32.const -10
        i64.extend_s/i32 ;; -10
        i64.sub ;; 4294967305
    )
)
