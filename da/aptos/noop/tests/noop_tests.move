#[test_only]
module noop::noop_tests {
    use noop::noop;
    use std::vector;

    #[test]
    fun test_noop() {
        let data = vector::empty<u8>();
        noop::noop(data);
        // No assertions needed since the function does nothing
    }
}