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

type ResultBase struct {
	// Code is to determine if the action succeeded.
	Code StatusCode
	// Message may contain settlement layer specific information (like detailed error message, etc)
	Message string
	// TODO(omritoptix): Move StateIndex to be part of the batch struct
	// StateIndex is the rollapp-specific index the batch was saved in the SL
	StateIndex uint64
}

type BatchMetaData struct {
	DA *da.DASubmitMetaData
}

type Batch struct {
	StartHeight uint64
	EndHeight   uint64
	AppHashes   [][32]byte
	// MetaData about the batch in the DA layer
	MetaData *BatchMetaData
}

type ResultRetrieveBatch struct {
	ResultBase
	*Batch
}

type State struct {
	StateIndex uint64
}

type ResultGetHeightState struct {
	ResultBase // NOTE: the state index of this will not be populated
	State
}

// Option is a function that sets a parameter on the settlement layer.
type Option func(ClientI)

// ClientI defines generic interface for Settlement layer interaction.
type ClientI interface {
	// Init is called once for the client initialization
	Init(config Config, pubsub *pubsub.Server, logger types.Logger, options ...Option) error
	// Start is called once, after Init. It's implementation should start the client service.
	Start() error
	// Stop is called once, after Start. It should stop the client service.
	Stop() error
	// SubmitBatch tries submitting the batch in an async way to the settlement layer. This should create a transaction which (potentially)
	// triggers a state transition in the settlement layer. Events are emitted on success or failure.
	SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) error
	// GetLatestBatch returns the latest batch from the settlement layer.
	GetLatestBatch() (*ResultRetrieveBatch, error)
	// GetBatchAtIndex returns the batch at the given index.
	GetBatchAtIndex(index uint64) (*ResultRetrieveBatch, error)

	// GetSequencers returns the list of the sequencers for this chain.
	GetSequencers() ([]*types.Sequencer, error)
	// GetProposer returns the current proposer for this chain.
	GetProposer() *types.Sequencer

	GetHeightState(uint64) (*ResultGetHeightState, error)
}
