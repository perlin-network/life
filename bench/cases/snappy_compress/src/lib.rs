extern crate snap;

#[no_mangle]
pub extern "C" fn app_main() -> i32 {
    const SIZE: usize = 1048576 * 8;
    let bytes: Vec<u8> = vec! [ 0; SIZE ];

    let mut total_len: i32 = 0;
    for _ in 0..1000 {
        total_len += snap::Encoder::new().compress_vec(&bytes).unwrap().len() as i32;
    }
    total_len
}
