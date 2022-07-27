package types

// Batch defines a struct for block aggregation for support of batching.
// TODO: maybe change to BlockBatch
type Batch struct {
	StartHeight uint64
	EndHeight   uint64
	Blocks      []*Block
	// Path for fetching the batch from a DA layer
	DAPath string
}
