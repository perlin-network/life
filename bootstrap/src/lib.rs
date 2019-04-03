extern crate wasmparser;
extern crate serde;
#[macro_use]
extern crate serde_derive;
extern crate rmp_serde;
extern crate serde_json;

use wasmparser::{WasmDecoder, BinaryReaderError, Type as WpType, FuncType as WpFuncType};

static mut CODE_BUF: *mut u8 = ::std::ptr::null_mut();
static mut PARSER_RESULT: *mut ParserResult = ::std::ptr::null_mut();
const CODE_BUF_SIZE: usize = 1048576;

struct ParserResult {
    output: Vec<u8>,
}

#[derive(Serialize, Deserialize, Clone, Debug, Default)]
struct ModuleInfo {
    types: Vec<FuncType>,
    functions: Vec<FunctionInfo>,
}

#[derive(Serialize, Deserialize, Clone, Debug, Default)]
struct FunctionInfo {
    
}

#[derive(Serialize, Deserialize, Clone, Debug)]
struct FuncType {
    params: Vec<ValueType>,
    returns: Vec<ValueType>,
}

#[derive(Serialize, Deserialize, Copy, Clone, Debug)]
enum ValueType {
    I32,
    I64,
    F32,
    F64,
}

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

fn set_parser_result(st: Box<ParserResult>) {
    unsafe {
        if !PARSER_RESULT.is_null() {
            Box::from_raw(PARSER_RESULT);
        }
        PARSER_RESULT = Box::into_raw(st);
    }
}

#[no_mangle]
pub extern "C" fn check_and_parse(code_ptr: *const u8, code_len: usize) -> u64 {
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
    let mut output = ModuleInfo::default();
    loop {
        let state = parser.read();
        use wasmparser::ParserState;
        match *state {
            ParserState::EndWasm => {
                set_parser_result(Box::new(ParserResult {
                    output: serde_json::to_vec(&output).unwrap(),
                }));
                return unsafe {
                      (&*PARSER_RESULT).output.as_ptr() as usize as u64
                    | (((&*PARSER_RESULT).output.len() as u64) << 32)
                };
            },
            ParserState::Error(_) => return 0,
            ParserState::TypeSectionEntry(ref ty) => {
                output.types.push(FuncType {
                    params: ty.params.iter().map(|x| wp_type_to_value_type(*x)).collect(),
                    returns: ty.returns.iter().map(|x| wp_type_to_value_type(*x)).collect(),
                });
            }
            _ => {}
        }
    }
}

fn wp_type_to_value_type(x: WpType) -> ValueType {
    match x {
        WpType::I32 => ValueType::I32,
        WpType::I64 => ValueType::I64,
        WpType::F32 => ValueType::F32,
        WpType::F64 => ValueType::F64,
        _ => unreachable!(),
    }
}