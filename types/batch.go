package types

const (
	MaxBlockSizeAdjustment = 0.9 // have a safety margin of 10% in regard of MaxBlockBatchSizeBytes
)

// Batch defines a struct for block aggregation for support of batching.
// TODO: maybe change to BlockBatch
type Batch struct {
	Blocks  []*Block
	Commits []*Commit
	// LastBatch is true if this is the last batch of the sequencer (i.e completes it's rotation flow).
	LastBatch  bool
	DRSVersion []uint32
	Revision   uint64
}

// StartHeight is the height of the first block in the batch.
func (b Batch) StartHeight() uint64 {
	if len(b.Blocks) == 0 {
		return 0
	}
	return b.Blocks[0].Header.Height
}

// EndHeight is the height of the last block in the batch
func (b Batch) EndHeight() uint64 {
	if len(b.Blocks) == 0 {
		return 0
	}
	return b.Blocks[len(b.Blocks)-1].Header.Height
}

// NumBlocks is the number of blocks in the batch
func (b Batch) NumBlocks() uint64 {
	return uint64(len(b.Blocks))
}

// SizeBlockAndCommitBytes returns the sum of the size of bytes of the blocks and commits
// The actual size of the batch may be different due to additional metadata and protobuf
// optimizations.
func (b Batch) SizeBlockAndCommitBytes() int {
	cnt := 0
	for _, block := range b.Blocks {
		cnt += block.SizeBytes()
	}
	for _, commit := range b.Commits {
		cnt += commit.SizeBytes()
	}
	return cnt
}

func (b Batch) SizeBytes() int {
	return b.ToProto().Size()
}
