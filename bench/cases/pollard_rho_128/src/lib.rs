fn pollard_rho_factor_i64(n: i64) -> (i64, i64) {
    let n = n as i128;

    #[inline]
    fn g(x: i128, n: i128) -> i128 {
        ((x * x) + 1) % n
    }

    #[inline]
    fn gcd(mut m: i128, mut n: i128) -> i128 {
        while m != 0 {
            let old_m = m;
            m = n % m;
            n = old_m;
        }

        n.abs()
    }

    let mut x: i128 = 5 /*::rand::random::<u8>() as i128 % 10*/;
    let mut y: i128 = x;
    let mut d: i128 = 1;

    while d == 1 {
        x = g(x, n);
        y = g(g(y, n), n);
        d = gcd((x - y).abs(), n);
    }

    if d == n { 
        return (1, n as i64);
    } else {
        return (d as i64, (n / d) as i64);
    }
}

#[no_mangle]
pub extern "C" fn app_main() -> i64 {
    let a: i64 = 613676879;
    let b: i64 = 895640371;

    let (mut r1, mut r2) = pollard_rho_factor_i64(a * b);
    if r1 > r2 {
        let t = r1;
        r1 = r2;
        r2 = t;
    }
    (r1 << 32) | r2
}