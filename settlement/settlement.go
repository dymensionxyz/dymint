package settlement

import (
	"time"

	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
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

type Batch struct {
	// sequencer is the bech32-encoded address of the sequencer sent the update
	Sequencer        string
	StartHeight      uint64
	EndHeight        uint64
	BlockDescriptors []rollapp.BlockDescriptor
	NextSequencer    string

	// MetaData about the batch in the DA layer
	MetaData     *da.DASubmitMetaData
	NumBlocks    uint64 // FIXME: can be removed. not used and will be deprecated
	CreationTime time.Time
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
	Init(config Config, rollappId string, pubsub *pubsub.Server, logger types.Logger, options ...Option) error
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
	// GetSequencerByAddress returns all sequencer information by its address.
	GetSequencerByAddress(address string) (types.Sequencer, error)
	// GetBatchAtHeight returns the batch at the given height.
	GetBatchAtHeight(index uint64, retry bool) (*ResultRetrieveBatch, error)
	// GetLatestHeight returns the latest state update height from the settlement layer.
	GetLatestHeight() (uint64, error)
	// GetLatestFinalizedHeight returns the latest finalized height from the settlement layer.
	GetLatestFinalizedHeight() (uint64, error)
	// GetAllSequencers returns all sequencers for this rollapp (bonded and not bonded).
	GetAllSequencers() ([]types.Sequencer, error)
	// GetBondedSequencers returns the list of the bonded sequencers for this rollapp.
	GetBondedSequencers() ([]types.Sequencer, error)
	// GetProposerAtHeight returns the current proposer for this chain.
	GetProposerAtHeight(height int64) (*types.Sequencer, error)
	// GetNextProposer returns the next proposer for this chain in case of a rotation.
	// If no rotation is in progress, it should return nil.
	GetNextProposer() (*types.Sequencer, error)
	// GetRollapp returns the rollapp information.
	GetRollapp() (*types.Rollapp, error)
	// GetObsoleteDrs returns the list of deprecated DRS.
	GetObsoleteDrs() ([]uint32, error)
	// GetSignerBalance returns the balance of the signer.
	GetSignerBalance() (types.Balance, error)
	// ValidateGenesisBridgeData validates the genesis bridge data.
	ValidateGenesisBridgeData(data rollapp.GenesisBridgeData) error
}
