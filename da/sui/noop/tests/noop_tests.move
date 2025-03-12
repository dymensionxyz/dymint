#[test_only]
module noop::noop_tests;
use noop::noop;

#[test]
fun test_noop() {
    let ctx = tx_context::dummy();
    let data = vector::empty<u8>();
    noop::noop(data, &ctx);
    // No assertions needed since the function does nothing
}
