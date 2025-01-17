package da

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	"github.com/celestiaorg/nmt"
	"github.com/cometbft/cometbft/crypto/merkle"
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

// DAMetaData contains meta data about a batch on the Data Availability Layer.
type DASubmitMetaData struct {
	// Height is the height of the block in the da layer
	Height uint64
	// Namespace ID
	Namespace []byte
	// Client is the client to use to fetch data from the da layer
	Client Client
	// Share commitment, for each blob, used to obtain blobs and proofs
	Commitment Commitment
	// Initial position for each blob in the NMT
	Index int
	// Number of shares of each blob
	Length int
	// any NMT root for the specific height, necessary for non-inclusion proof
	Root []byte
	// WeaveVM arweave block hash means data stored permanently in Arweave block
	WvmArweaveBlockHash string
	// WeaveVM tx hash
	WvmTxHash string
	// WeaveVM block hash
	WvmBlockHash string
}

type Balance struct {
	Amount math.Int
	Denom  string
}

const PathSeparator = "|"

// ToPath converts a DAMetaData to a path.
func (d *DASubmitMetaData) ToPath() string {
	// convert uint64 to string
	if d.Commitment != nil {
		commitment := hex.EncodeToString(d.Commitment)
		dataroot := hex.EncodeToString(d.Root)
		path := []string{
			string((d.Client)),
			strconv.FormatUint(d.Height, 10),
			strconv.Itoa(d.Index),
			strconv.Itoa(d.Length),
			commitment,
			hex.EncodeToString(d.Namespace),
			dataroot,
		}
		for i, part := range path {
			path[i] = strings.Trim(part, PathSeparator)
		}
		return strings.Join(path, PathSeparator)
	} else {
		path := []string{string(d.Client), PathSeparator, strconv.FormatUint(d.Height, 10)}
		return strings.Join(path, PathSeparator)
	}
}

// FromPath parses a path to a DAMetaData.
func (d *DASubmitMetaData) FromPath(path string) (*DASubmitMetaData, error) {
	pathParts := strings.FieldsFunc(path, func(r rune) bool { return r == rune(PathSeparator[0]) })
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid DA path")
	}

	height, err := strconv.ParseUint(pathParts[1], 10, 64)
	if err != nil {
		return nil, err
	}

	submitData := &DASubmitMetaData{
		Height: height,
		Client: Client(pathParts[0]),
	}
	// TODO: check per DA and panic if not enough parts
	if len(pathParts) == 6 {
		submitData.Index, err = strconv.Atoi(pathParts[2])
		if err != nil {
			return nil, err
		}
		submitData.Length, err = strconv.Atoi(pathParts[3])
		if err != nil {
			return nil, err
		}
		submitData.Commitment, err = hex.DecodeString(pathParts[4])
		if err != nil {
			return nil, err
		}
		submitData.Namespace, err = hex.DecodeString(pathParts[5])
		if err != nil {
			return nil, err
		}
		submitData.Root, err = hex.DecodeString(pathParts[6])
		if err != nil {
			return nil, err
		}
	}

	return submitData, nil
}

// DAMetaData contains meta data about a batch on the Data Availability Layer.
type DACheckMetaData struct {
	// Height is the height of the block in the da layer
	Height uint64
	// Client is the client to use to fetch data from the da layer
	Client Client
	// Submission index in the Hub
	SLIndex uint64
	// Namespace ID
	Namespace []byte
	// Share commitment, for each blob, used to obtain blobs and proofs
	Commitment Commitment
	// Initial position for each blob in the NMT
	Index int
	// Number of shares of each blob
	Length int
	// Proofs necessary to validate blob inclusion in the specific height
	Proofs [][]*nmt.Proof
	// NMT roots for each NMT Proof
	NMTRoots []byte
	// Proofs necessary to validate blob inclusion in the specific height
	RowProofs []*merkle.Proof
	// any NMT root for the specific height, necessary for non-inclusion proof
	Root []byte
	// WeaveVM arweave block hash means data stored permanently in Arweave block
	WvmArweaveBlockHash string
	// WeaveVM tx hash
	WvmTxHash string
	// WeaveVM block hash
	WvmBlockHash string
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
	// DAHeight informs about a height on Data Availability Layer for given result.
	CheckMetaData *DACheckMetaData
}

// ResultRetrieveBatch contains batch of blocks returned from DA layer client.
type ResultRetrieveBatch struct {
	BaseResult
	// Block is the full block retrieved from Data Availability Layer.
	// If Code is not equal to StatusSuccess, it has to be nil.
	Batches []*types.Batch
	// DAHeight informs about a height on Data Availability Layer for given result.
	CheckMetaData *DACheckMetaData
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
	CheckBatchAvailability(daMetaData *DASubmitMetaData) ResultCheckBatch

	// Used to check when the DA light client finished syncing
	WaitForSyncing()

	// Returns the maximum allowed blob size in the DA, used to check the max batch size configured
	GetMaxBlobSizeBytes() uint32

	// GetSignerBalance returns the balance for a specific address
	GetSignerBalance() (Balance, error)

	// Something third parties can use to identify rollapp activity on the DA
	DAPath() string
}

// BatchRetriever is additional interface that can be implemented by Data Availability Layer Client that is able to retrieve
// block data from DA layer. This gives the ability to use it for block synchronization.
type BatchRetriever interface {
	// RetrieveBatches returns blocks at given data layer height from data availability layer.
	RetrieveBatches(daMetaData *DASubmitMetaData) ResultRetrieveBatch
	// CheckBatchAvailability checks the availability of the blob received getting proofs and validating them
	CheckBatchAvailability(daMetaData *DASubmitMetaData) ResultCheckBatch
}
