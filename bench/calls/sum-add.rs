extern {
    fn sum(x: i32, y: i32) -> i32;
}

#[no_mangle]
pub extern fn add1(x: i32, y: i32) -> i32 {
    unsafe { sum(x, y) + 1 }
}

#[no_mangle]
pub extern fn callSumAndAdd1(x: i32, y: i32, cnt: i32) -> i32 {
	let mut res = x;
	for _i in 0..cnt {
	    unsafe { res = sum(res, y) + 1 }
	}
	return res
}
