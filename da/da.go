package da

import (
	"strconv"
	"strings"

	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/rollkit/celestia-openrpc/types/blob"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// StatusCode is a type for DA layer return status.
// TODO: define an enum of different non-happy-path cases
// that might need to be handled by Dymint independent of
// the underlying DA chain.
type StatusCode uint64

// Commitment should contain serialized cryptographic commitment to Blob value.
type Commitment = []byte

// Data Availability return codes.
const (
	StatusUnknown StatusCode = iota
	StatusSuccess
	StatusTimeout
	StatusError
	StatusBlobNotFound
	StatusBlobNotIncluded
	StatusProofNotMatching
)

// Client defines all the possible da clients
type Client string

// Data availability clients
const (
	Mock     Client = "mock"
	Celestia Client = "celestia"
	Avail    Client = "avail"
)

// Option is a function that sets a parameter on the da layer.
type Option func(DataAvailabilityLayerClient)

// BaseResult contains basic information returned by DA layer.
type BaseResult struct {
	// Code is to determine if the action succeeded.
	Code StatusCode
	// Message may contain DA layer specific information (like DA block height/hash, detailed error message, etc)
	Message string
	// DAHeight informs about a height on Data Availability Layer for given result.
	MetaData *DAMetaData
}

// DAMetaData contains meta data about a batch on the Data Availability Layer.
type DAMetaData struct {
	// Height is the height of the block in the da layer
	Height uint64
	// Client is the client to use to fetch data from the da layer
	Client Client
	//Share commitment
	Commitments []Commitment
	//share position
	Indexes []int
	//share length
	Lengths []int
	//Proofs
	Proofs *blob.Proof
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
		Client: Client(pathParts[0]),
	}, nil
}

// ResultSubmitBatch contains information returned from DA layer after block submission.
type ResultSubmitBatch struct {
	BaseResult
	// Not sure if this needs to be bubbled up to other
	// parts of Dymint.
	// Hash hash.Hash
}

// ResultCheckBatch contains information about block availability, returned from DA layer client.
type ResultCheckBatch struct {
	BaseResult
	// DataAvailable is the actual answer whether the block is available or not.
	// It can be true if and only if Code is equal to StatusSuccess.
	DataAvailable bool
}

// ResultRetrieveBatch contains batch of blocks returned from DA layer client.
type ResultRetrieveBatch struct {
	BaseResult
	// Block is the full block retrieved from Data Availability Layer.
	// If Code is not equal to StatusSuccess, it has to be nil.
	Batches []*types.Batch
}

// DataAvailabilityLayerClient defines generic interface for DA layer block submission.
// It also contains life-cycle methods.
type DataAvailabilityLayerClient interface {
	// Init is called once to allow DA client to read configuration and initialize resources.
	Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KVStore, logger log.Logger, options ...Option) error

	// Start is called once, after Init. It's implementation should start operation of DataAvailabilityLayerClient.
	Start() error

	// Stop is called once, when DataAvailabilityLayerClient is no longer needed.
	Stop() error

	// SubmitBatch submits the passed in block to the DA layer.
	// This should create a transaction which (potentially)
	// triggers a state transition in the DA layer.
	SubmitBatch(batch *types.Batch) ResultSubmitBatch

	GetClientType() Client
}

// BatchRetriever is additional interface that can be implemented by Data Availability Layer Client that is able to retrieve
// block data from DA layer. This gives the ability to use it for block synchronization.
type BatchRetriever interface {
	// RetrieveBatches returns blocks at given data layer height from data availability layer.
	RetrieveBatches(daMetaData *DAMetaData) ResultRetrieveBatch
}
