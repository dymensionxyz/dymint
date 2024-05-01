package settlement

import (
	"github.com/dymensionxyz/dymint/da"
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
	// TODO(omritoptix): Move StateIndex to be part of the batch struct
	// StateIndex is the rollapp-specific index the batch was saved in the SL
	StateIndex uint64
}

// BatchMetaData aggregates all the batch metadata.
type BatchMetaData struct {
	DA *da.DASubmitMetaData
}

// Batch defines a batch structure for the settlement layer
type Batch struct {
	StartHeight uint64
	EndHeight   uint64
	AppHashes   [][32]byte
	// MetaData about the batch in the DA layer
	MetaData *BatchMetaData
}

// ResultRetrieveBatch contains information returned from settlement layer after batch retrieva
type ResultRetrieveBatch struct {
	BaseResult
	*Batch
}

type State struct {
	StateIndex uint64
}

type ResultGetHeightState struct {
	BaseResult // NOTE: the state index of this will not be populated
	State
}

// Option is a function that sets a parameter on the settlement layer.
type Option func(LayerI)

// LayerI defines generic interface for Settlement layer interaction.
type LayerI interface {
	// Init is called once for the client initialization
	Init(config Config, pubsub *pubsub.Server, logger types.Logger, options ...Option) error

	// Start is called once, after Init. It's implementation should start the client service.
	Start() error

	// Stop is called once, after Start. It should stop the client service.
	Stop() error

	// SubmitBatch tries submitting the batch in an async way to the settlement layer. This should create a transaction which (potentially)
	// triggers a state transition in the settlement layer. Events are emitted on success or failure.
	SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) error

	// RetrieveBatch Gets the batch which contains the given height. Empty height returns the latest batch.
	RetrieveBatch(stateIndex ...uint64) (*ResultRetrieveBatch, error)

	// GetSequencersList returns the list of the sequencers for this chain.
	GetSequencersList() []*types.Sequencer

	// GetProposer returns the current proposer for this chain.
	GetProposer() *types.Sequencer

	GetHeightState(uint64) (*ResultGetHeightState, error)
}

// HubClient is a helper interface for a more granular interaction with the hub.
// Implementing a new settlement layer client basically requires embedding the base client
// and implementing the helper interfaces.
type HubClient interface {
	Start() error
	Stop() error
	PostBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) error
	GetLatestBatch(rollappID string) (*ResultRetrieveBatch, error)
	GetBatchAtIndex(rollappID string, index uint64) (*ResultRetrieveBatch, error)
	GetHeightState(rollappID string, index uint64) (*ResultGetHeightState, error)
	GetSequencers(rollappID string) ([]*types.Sequencer, error)
}
