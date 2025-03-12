module noop::noop {
    use std::vector;
    use sui::tx_context;

    public fun noop(_: vector<u8> ,_: &TxContext) {
        // Do nothing
    }
}