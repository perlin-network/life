extern crate snap;

#[no_mangle]
pub extern "C" fn app_main() -> i32 {
    const SIZE: usize = 1048576 * 8;
    let bytes: Vec<u8> = vec! [ 0; SIZE ];
    let encoded = snap::Encoder::new().compress_vec(&bytes).unwrap();
    encoded.len() as i32
}
