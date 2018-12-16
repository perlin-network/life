extern "C" {
    fn print_f64(x: f64);
    fn identity_map_f64(x: f64) -> f64;
}

#[no_mangle]
pub extern "C" fn sqrt(v: f64) -> f64 {
	let mut lower: f64 = 0.0;
	let mut upper: f64 = v;

	while upper - lower > 1e-3 {
		let mid = (lower + upper) / 2.0;
		if mid * mid > v {
			upper = mid;
		} else {
			lower = mid;
		}
	}
	lower
}

#[no_mangle]
pub extern "C" fn run_sqrt() {
	unsafe { print_f64(sqrt(identity_map_f64(3.0))) };
}
