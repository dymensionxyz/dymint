package settlement

import (
	"github.com/celestiaorg/optimint/log"
	"github.com/celestiaorg/optimint/store"
	"github.com/celestiaorg/optimint/types"
)

// StatusCode is a type for settlement layer return status.
type StatusCode uint64

// settlement layer return codes.
const (
	StatusUnknown StatusCode = iota
	StatusSuccess
	StatusTimeout
	StatusError
)

// BaseResult contains basic information returned by the settlement layer.
type BaseResult struct {
	// Code is to determine if the action succeeded.
	Code StatusCode
	// Message may contain settlement layer specific information (like detailed error message, etc)
	Message string
}

// ResultSubmitBatch contains information returned from settlement layer after batch submission.
type ResultSubmitBatch struct {
	BaseResult
}

// ResultRetrieveBatch contains information returned from settlement layer after batch retrieval.
type ResultRetrieveBatch struct {
	BaseResult
	StartHeight uint64
	EndHeight   uint64
	AppHashes   [][32]byte
}

// LayerClient defines generic interface for Settlement layer interaction.
type LayerClient interface {

	// Init is called once for the client initialization
	Init(config []byte, settlementKV store.KVStore, logger log.Logger) error

	// Start is called once, after Init. It's implementation should start the client service.
	Start() error

	// Stop is called once, after Start. It should stop the client service.
	Stop() error

	// SubmitBatch submits the batch to the settlement layer. This should create a transaction which (potentially)
	// triggers a state transition in the settlement layer.
	SubmitBatch(batch *types.Batch) ResultSubmitBatch

	// RetrieveBatch Gets the batch which contains the given height. Empty height returns the latest batch.
	RetrieveBatch(height ...uint64) (ResultRetrieveBatch, error)

	// TODO(omritoptix): Support getting multiple batches and pagination
}
