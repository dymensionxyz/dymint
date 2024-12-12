package settlement

import (
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
)


type StatusCode uint64


const (
	StatusUnknown StatusCode = iota
	StatusSuccess
	StatusTimeout
	StatusError
)

type ResultBase struct {
	
	Code StatusCode
	
	Message string
	
	
	StateIndex uint64
}

type BatchMetaData struct {
	DA *da.DASubmitMetaData
}

type Batch struct {
	
	Sequencer        string
	StartHeight      uint64
	EndHeight        uint64
	BlockDescriptors []rollapp.BlockDescriptor
	NextSequencer    string

	
	MetaData  *BatchMetaData
	NumBlocks uint64 
}

type ResultRetrieveBatch struct {
	ResultBase
	*Batch
}

type State struct {
	StateIndex uint64
}

type ResultGetHeightState struct {
	ResultBase 
	State
}


type Option func(ClientI)


type ClientI interface {
	
	Init(config Config, rollappId string, pubsub *pubsub.Server, logger types.Logger, options ...Option) error
	
	Start() error
	
	Stop() error
	
	
	SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) error
	
	GetLatestBatch() (*ResultRetrieveBatch, error)
	
	GetBatchAtIndex(index uint64) (*ResultRetrieveBatch, error)
	
	GetSequencerByAddress(address string) (types.Sequencer, error)
	
	GetBatchAtHeight(index uint64) (*ResultRetrieveBatch, error)
	
	GetLatestHeight() (uint64, error)
	
	GetLatestFinalizedHeight() (uint64, error)
	
	GetAllSequencers() ([]types.Sequencer, error)
	
	GetBondedSequencers() ([]types.Sequencer, error)
	
	GetProposerAtHeight(height int64) (*types.Sequencer, error)
	
	
	GetNextProposer() (*types.Sequencer, error)
	
	GetRollapp() (*types.Rollapp, error)
	
	GetObsoleteDrs() ([]uint32, error)
	
	GetSignerBalance() (types.Balance, error)
	
	ValidateGenesisBridgeData(data rollapp.GenesisBridgeData) error
}
