package types

const (
	MaxBlockSizeAdjustment = 0.9 // have a safety margin of 10% in regard of MaxBlockBatchSizeBytes
)

// Batch defines a struct for block aggregation for support of batching.
// TODO: maybe change to BlockBatch
type Batch struct {
	StartHeight uint64
	EndHeight   uint64
	Blocks      []*Block
	Commits     []*Commit
}
