extern crate wasmparser;

use wasmparser::WasmDecoder;

static mut CODE_BUF: *mut u8 = ::std::ptr::null_mut();
const CODE_BUF_SIZE: usize = 1048576;

#[no_mangle]
pub extern "C" fn get_code_buf(n: usize) -> *mut u8 {
    if n > CODE_BUF_SIZE {
        return ::std::ptr::null_mut();
    }
    unsafe {
        if CODE_BUF.is_null() {
            let mut vec: Vec<u8> = Vec::with_capacity(CODE_BUF_SIZE);
            vec.set_len(CODE_BUF_SIZE);
            CODE_BUF = Box::into_raw(vec.into_boxed_slice()) as *mut u8;
        }
        return CODE_BUF;
    }
}

#[no_mangle]
pub extern "C" fn check(code_ptr: *mut u8, code_len: usize) -> i32 {
    let code = unsafe { ::std::slice::from_raw_parts(code_ptr, code_len) };
    let mut parser = wasmparser::ValidatingParser::new(
        code,
        Some(wasmparser::ValidatingParserConfig {
            operator_config: wasmparser::OperatorValidatorConfig {
                enable_threads: false,
                enable_reference_types: false,
                enable_simd: false,
                enable_bulk_memory: false,
            },
            mutable_global_imports: false,
        }),
    );
    loop {
        let state = parser.read();
        match *state {
            wasmparser::ParserState::EndWasm => return 1,
            wasmparser::ParserState::Error(_) => return 0,
            _ => {}
        }
    }
}
