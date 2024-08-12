package types

// Hash returns ABCI-compatible hash of a header.
func (h *Header) Hash() [32]byte {
	var hash [32]byte
	abciHeader := ToABCIHeader(h)
	copy(hash[:], abciHeader.Hash())
	return hash
}

// Hash returns ABCI-compatible hash of a block.
func (b *Block) Hash() [32]byte {
	return b.Header.Hash()
}
