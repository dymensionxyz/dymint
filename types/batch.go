package types

const (
	MaxBlockSizeAdjustment = 0.9 // have a safety margin of 10% in regard of MaxBlockBatchSizeBytes
)

// Batch defines a struct for block aggregation for support of batching.
// TODO: maybe change to BlockBatch
type Batch struct {
	Blocks  []*Block
	Commits []*Commit
}

// StartHeight is the height of the first block in the batch.
func (b Batch) StartHeight() uint64 { // TODO: check log calls end in ()
	if len(b.Blocks) == 0 {
		return 0
	}
	return b.Blocks[0].Header.Height
}

// EndHeight is the height of the last block in the batch
func (b Batch) EndHeight() uint64 { // TODO: check log calls end in ()
	if len(b.Blocks) == 0 {
		return 0
	}
	return b.Blocks[len(b.Blocks)-1].Header.Height
}

// NumBlocks is the number of blocks in the batch
func (b Batch) NumBlocks() uint64 {
	return uint64(len(b.Blocks))
}

func (b Batch) SizeBytes() int {
	return b.ToProto().Size()
}
