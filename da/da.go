package da

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

// StatusCode is a type for DA layer return status.
// TODO: define an enum of different non-happy-path cases
// that might need to be handled by Dymint independent of
// the underlying DA chain. Use int32 to match the protobuf
// enum representation.
type StatusCode int32

// Commitment should contain serialized cryptographic commitment to Blob value.
type Commitment = []byte

// Blob is the data submitted/received from DA interface.
type Blob = []byte

// Data Availability return codes.
const (
	StatusUnknown StatusCode = iota
	StatusSuccess
	StatusError
)

// Client defines all the possible da clients
type Client string

// Data availability clients
const (
	Mock     Client = "mock"
	Celestia Client = "celestia"
	Avail    Client = "avail"
	Grpc     Client = "grpc"
	WeaveVM  Client = "weavevm"
)

// Option is a function that sets a parameter on the da layer.
type Option func(DataAvailabilityLayerClient)

// BaseResult contains basic information returned by DA layer.
type BaseResult struct {
	// Code is to determine if the action succeeded.
	Code StatusCode
	// Message may contain DA layer specific information (like DA block height/hash, detailed error message, etc)
	Message string
	// Error is the error returned by the DA layer
	Error error
}

type Balance struct {
	Amount math.Int
	Denom  string
}

const PathSeparator = "|"

type DASubmitMetaData struct {
	Client Client
	DAPath string
}

// ToPath converts a DAMetaData to a path.
func (d *DASubmitMetaData) ToPath() string {
	path := []string{string(d.Client), PathSeparator, d.DAPath}
	return strings.Join(path, PathSeparator)
}

// FromPath parses a path to a DAMetaData.
func (d *DASubmitMetaData) FromPath(path string) (*DASubmitMetaData, error) {
	pathParts := strings.FieldsFunc(path, func(r rune) bool { return r == rune(PathSeparator[0]) })
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid DA path")
	}

	submitData := &DASubmitMetaData{
		Client: Client(pathParts[0]),
		DAPath: strings.Trim(path, pathParts[0]+PathSeparator),
	}
	return submitData, nil
}

// ResultSubmitBatch contains information returned from DA layer after block submission.
type ResultSubmitBatch struct {
	BaseResult
	// DAHeight informs about a height on Data Availability Layer for given result.
	SubmitMetaData *DASubmitMetaData
}

// ResultCheckBatch contains information about block availability, returned from DA layer client.
type ResultCheckBatch struct {
	BaseResult
}

// ResultRetrieveBatch contains batch of blocks returned from DA layer client.
type ResultRetrieveBatch struct {
	BaseResult
	// Batches are the blob(s) with blocks retrieved from Data Availability Layer.
	// If Code is not equal to StatusSuccess, it has to be nil.
	Batches []*types.Batch
}

// DataAvailabilityLayerClient defines generic interface for DA layer block submission.
// It also contains life-cycle methods.
type DataAvailabilityLayerClient interface {
	// Init is called once to allow DA client to read configuration and initialize resources.
	Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KV, logger types.Logger, options ...Option) error

	// Start is called once, after Init. It's implementation should start operation of DataAvailabilityLayerClient.
	Start() error

	// Stop is called once, when DataAvailabilityLayerClient is no longer needed.
	Stop() error

	// SubmitBatch submits the passed in block to the DA layer.
	// This should create a transaction which (potentially)
	// triggers a state transition in the DA layer.
	SubmitBatch(batch *types.Batch) ResultSubmitBatch

	GetClientType() Client

	// CheckBatchAvailability checks the availability of the blob submitted getting proofs and validating them
	CheckBatchAvailability(daPath string) ResultCheckBatch

	// Returns the maximum allowed blob size in the DA, used to check the max batch size configured
	GetMaxBlobSizeBytes() uint64

	// GetSignerBalance returns the balance for a specific address
	GetSignerBalance() (Balance, error)

	// Something third parties can use to identify rollapp activity on the DA
	DAPath() string
}

// BatchRetriever is additional interface that can be implemented by Data Availability Layer Client that is able to retrieve
// block data from DA layer. This gives the ability to use it for block synchronization.
type BatchRetriever interface {
	// RetrieveBatches returns blocks at given data layer height from data availability layer.
	RetrieveBatches(daPath string) ResultRetrieveBatch
	// CheckBatchAvailability checks the availability of the blob received getting proofs and validating them
	CheckBatchAvailability(daPath string) ResultCheckBatch
}
