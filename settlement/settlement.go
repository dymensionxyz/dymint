package settlement

import (
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
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

// DAMetaData contains meta data about a batch on the Data Availability Layer.
type DAMetaData struct {
	// Height is the height of the block in the da layer
	Height uint64
	// Path is the path to fetch data from the da layer
	Path string
	// Client is the client to use to fetch data from the da layer
	Client da.Client
}

// BatchMetaData aggregates all the batch metadata.
type BatchMetaData struct {
	DA *DAMetaData
}

// Batch defines a batch structure for the settlement layer
type Batch struct {
	StartHeight uint64
	EndHeight   uint64
	AppHashes   [][32]byte
	// MetaData about the batch in the DA layer
	MetaData *BatchMetaData
}

// ResultSubmitBatch contains information returned from settlement layer after batch submission.
type ResultSubmitBatch struct {
	BaseResult
}

// ResultRetrieveBatch contains information returned from settlement layer after batch retrieval.
type ResultRetrieveBatch struct {
	BaseResult
	*Batch
}

// LayerClient defines generic interface for Settlement layer interaction.
type LayerClient interface {

	// Init is called once for the client initialization
	Init(config []byte, pubsub *pubsub.Server, logger log.Logger) error

	// Start is called once, after Init. It's implementation should start the client service.
	Start() error

	// Stop is called once, after Start. It should stop the client service.
	Stop() error

	// SubmitBatch submits the batch to the settlement layer. This should create a transaction which (potentially)
	// triggers a state transition in the settlement layer.
	SubmitBatch(batch *types.Batch, daResult *da.ResultSubmitBatch) ResultSubmitBatch

	// RetrieveBatch Gets the batch which contains the given height. Empty height returns the latest batch.
	RetrieveBatch(height ...uint64) (ResultRetrieveBatch, error)

	// TODO(omritoptix): Support getting multiple batches and pagination
}
