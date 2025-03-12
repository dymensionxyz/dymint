#[test_only]
module noop::noop_tests;
use noop::noop;

#[test]
fun test_noop() {
    let ctx = tx_context::dummy(); // Create a dummy TxContext
    let data = vector::empty<u8>(); // Create an empty vector<u8>
    noop::noop(data, &ctx); // Call the noop function
    // No assertions needed since the function does nothing
}
