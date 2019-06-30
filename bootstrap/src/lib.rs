extern crate wasmparser;
extern crate protobuf;

mod protos;

use wasmparser::{WasmDecoder, BinaryReaderError, Type as WpType, FuncType as WpFuncType, ImportSectionEntryType, NameEntry, Operator as WpOperator};
use protos::module::{ModuleInfo, FuncType, ValueType, FuncInfo, FuncImportInfo, TableInfo, MemoryInfo, LocalInfo, Operator, Op};
use protobuf::Message;

static mut CODE_BUF: *mut u8 = ::std::ptr::null_mut();
static mut PARSER_RESULT: *mut ParserResult = ::std::ptr::null_mut();
const CODE_BUF_SIZE: usize = 1048576;

struct ParserResult {
    output: Vec<u8>,
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
    let mut output = ModuleInfo::new();
    let mut current_func: usize = 0;
    loop {
        let state = parser.read();
        use wasmparser::ParserState;
        match *state {
            ParserState::EndWasm => {
                set_parser_result(Box::new(ParserResult {
                    output: output.write_to_bytes().unwrap(),
                }));
                return unsafe {
                      (&*PARSER_RESULT).output.as_ptr() as usize as u64
                    | (((&*PARSER_RESULT).output.len() as u64) << 32)
                };
            },
            ParserState::Error(_) => return 0,
            ParserState::TypeSectionEntry(ref ty) => {
                let mut ft = FuncType::new();
                ft.params = ty.params.iter().map(|x| wp_type_to_value_type(*x)).collect();
                ft.returns = ty.returns.iter().map(|x| wp_type_to_value_type(*x)).collect();
                output.types.push(ft);
            },
            ParserState::ImportSectionEntry { module, field, ty } => {
                match ty {
                    ImportSectionEntryType::Function(type_id) => {
                        let mut info = FuncImportInfo::new();
                        info.module = module.into();
                        info.field = field.into();
                        info.type_id = type_id;
                        output.func_imports.push(info);
                    },
                    _ => {}
                }
            },
            ParserState::FunctionSectionEntry(type_id) => {
                let mut info = FuncInfo::new();
                info.type_id = type_id;
                output.functions.push(info);
            },
            ParserState::TableSectionEntry(ty) => {
                let mut info = TableInfo::new();
                info.initial = ty.limits.initial;
                if let Some(x) = ty.limits.maximum {
                    info.maximum = x;
                    info.has_maximum = true;
                }
                output.tables.push(info);
            },
            ParserState::MemorySectionEntry(ty) => {
                let mut info = MemoryInfo::new();
                info.initial = ty.limits.initial;
                if let Some(x) = ty.limits.maximum {
                    info.maximum = x;
                    info.has_maximum = true;
                }
                output.memories.push(info);
            },
            ParserState::NameSectionEntry(ref entry) => {
                match *entry {
                    NameEntry::Function(ref names) => {
                        for name in names.iter() {
                            let index = name.index as usize;
                            if index < output.func_imports.len() {
                                output.func_imports[index].name = name.name.to_string();
                            } else {
                                output.functions[index - output.func_imports.len()].name = name.name.to_string();
                            }
                        }
                    },
                    _ => {}
                }
            },
            ParserState::StartSectionEntry(x) => {
                output.start_func = x;
            },
            ParserState::BeginFunctionBody { .. } => {
            },
            ParserState::FunctionBodyLocals { ref locals } => {
                for &(n, ty) in locals.iter() {
                    let mut loc = LocalInfo::new();
                    loc.count = n;
                    loc.ty = wp_type_to_value_type(ty);
                    output.functions[current_func].locals.push(loc);
                }
            },
            ParserState::CodeOperator(ref op) => {
                let op = match *op {
                    _ => Operator { op: Op::Nop, ..Default::default() } // TODO
                };
                output.functions[current_func].code.push(op);
            },
            ParserState::EndFunctionBody => {
                current_func += 1;
            },
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