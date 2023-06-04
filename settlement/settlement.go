package settlement

import (
	"strconv"
	"strings"

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
	// StateIndex is the rollapp-specific index the batch was saved in the SL
	StateIndex uint64
}

// DAMetaData contains meta data about a batch on the Data Availability Layer.
type DAMetaData struct {
	// Height is the height of the block in the da layer
	Height uint64
	// Client is the client to use to fetch data from the da layer
	Client da.Client
}

// ToPath converts a DAMetaData to a path.
func (d *DAMetaData) ToPath() string {
	// convert uint64 to string
	path := []string{string(d.Client), ".", strconv.FormatUint(d.Height, 10)}
	return strings.Join(path, "")
}

// FromPath parses a path to a DAMetaData.
func (d *DAMetaData) FromPath(path string) (*DAMetaData, error) {
	pathParts := strings.FieldsFunc(path, func(r rune) bool { return r == '.' })
	height, err := strconv.ParseUint(pathParts[1], 10, 64)
	if err != nil {
		return nil, err
	}
	return &DAMetaData{
		Height: height,
		Client: da.Client(pathParts[0]),
	}, nil
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

// ResultRetrieveBatch contains information returned from settlement layer after batch retrieva
type ResultRetrieveBatch struct {
	BaseResult
	*Batch
}

// Option is a function that sets a parameter on the settlement layer.
type Option func(LayerI)

// LayerI defines generic interface for Settlement layer interaction.
type LayerI interface {

	// Init is called once for the client initialization
	Init(config Config, pubsub *pubsub.Server, logger log.Logger, options ...Option) error

	// Start is called once, after Init. It's implementation should start the client service.
	Start() error

	// Stop is called once, after Start. It should stop the client service.
	Stop() error

	// SubmitBatch tries submiting the batch in an async way to the settlement layer. This should create a transaction which (potentially)
	// triggers a state transition in the settlement layer. Events are emitted on success or failure.
	SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch)

	// RetrieveBatch Gets the batch which contains the given height. Empty height returns the latest batch.
	RetrieveBatch(stateIndex ...uint64) (*ResultRetrieveBatch, error)

	// GetSequencersList returns the list of the sequencers for this chain.
	GetSequencersList() []*types.Sequencer

	// GetProposer returns the current proposer for this chain.
	GetProposer() *types.Sequencer
}

// HubClient is an helper interface for a more granualr interaction with the hub.
// Implementing a new settlement layer client basically requires embedding the base client
// and implementing the helper interfaces.
type HubClient interface {
	Start() error
	Stop() error
	PostBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch)
	GetLatestBatch(rollappID string) (*ResultRetrieveBatch, error)
	GetBatchAtIndex(rollappID string, index uint64) (*ResultRetrieveBatch, error)
	GetSequencers(rollappID string) ([]*types.Sequencer, error)
}
