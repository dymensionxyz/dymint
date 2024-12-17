package da

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/cometbft/cometbft/crypto/merkle"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

type StatusCode int32

type Commitment = []byte

type Blob = []byte

const (
	StatusUnknown StatusCode = iota
	StatusSuccess
	StatusError
)

type Client string

const (
	Mock     Client = "mock"
	Celestia Client = "celestia"
	Avail    Client = "avail"
	Grpc     Client = "grpc"
)

type Option func(DataAvailabilityLayerClient)

type BaseResult struct {
	Code StatusCode

	Message string

	Error error
}

type DASubmitMetaData struct {
	Height uint64

	Namespace []byte

	Client Client

	Commitment Commitment

	Index int

	Length int

	Root []byte
}

type Balance struct {
	Amount math.Int
	Denom  string
}

const PathSeparator = "|"

func (d *DASubmitMetaData) ToPath() string {
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

	if len(pathParts) == 7 {
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

type DACheckMetaData struct {
	Height uint64

	Client Client

	SLIndex uint64

	Namespace []byte

	Commitment Commitment

	Index int

	Length int

	Proofs []*blob.Proof

	NMTRoots []byte

	RowProofs []*merkle.Proof

	Root []byte
}

type ResultSubmitBatch struct {
	BaseResult

	SubmitMetaData *DASubmitMetaData
}

type ResultCheckBatch struct {
	BaseResult

	CheckMetaData *DACheckMetaData
}

type ResultRetrieveBatch struct {
	BaseResult

	Batches []*types.Batch

	CheckMetaData *DACheckMetaData
}

type DataAvailabilityLayerClient interface {
	Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KV, logger types.Logger, options ...Option) error

	Start() error

	Stop() error

	SubmitBatch(batch *types.Batch) ResultSubmitBatch

	GetClientType() Client

	CheckBatchAvailability(daMetaData *DASubmitMetaData) ResultCheckBatch

	WaitForSyncing()

	GetMaxBlobSizeBytes() uint32

	GetSignerBalance() (Balance, error)
}

type BatchRetriever interface {
	RetrieveBatches(daMetaData *DASubmitMetaData) ResultRetrieveBatch

	CheckBatchAvailability(daMetaData *DASubmitMetaData) ResultCheckBatch
}
