package block

import (
	"errors"
	"time"

	proto2 "github.com/gogo/protobuf/proto"
	"go.uber.org/multierr"

	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto/encoding"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/types"
	protoutils "github.com/dymensionxyz/dymint/utils/proto"
)

// default minimum block max size allowed. not specific reason to set it to 10K, but we need to avoid no transactions can be included in a block.
const minBlockMaxBytes = 10000

type ExecutorI interface {
	InitChain(genesis *tmtypes.GenesisDoc, genesisChecksum string, valset []*tmtypes.Validator) (*abci.ResponseInitChain, error)
	CreateBlock(height uint64, lastCommit *types.Commit, lastHeaderHash, nextSeqHash [32]byte, state *types.State, maxBlockDataSizeBytes uint64) *types.Block
	Commit(state *types.State, block *types.Block, resp *tmstate.ABCIResponses) ([]byte, int64, error)
	GetAppInfo() (*abci.ResponseInfo, error)
	ExecuteBlock(block *types.Block) (*tmstate.ABCIResponses, error)
	UpdateStateAfterInitChain(s *types.State, res *abci.ResponseInitChain)
	UpdateMempoolAfterInitChain(s *types.State)
	UpdateStateAfterCommit(s *types.State, resp *tmstate.ABCIResponses, appHash []byte, block *types.Block)
	UpdateProposerFromBlock(s *types.State, seqSet *types.SequencerSet, block *types.Block) bool

	/* Consensus Messages */

	AddConsensusMsgs(...proto2.Message)
	GetConsensusMsgs() []proto2.Message
}

var _ ExecutorI = new(Executor)

// Executor creates and applies blocks and maintains state.
type Executor struct {
	localAddress          []byte
	chainID               string
	proxyAppConsensusConn proxy.AppConnConsensus
	proxyAppQueryConn     proxy.AppConnQuery
	mempool               mempool.Mempool
	consensusMsgQueue     *ConsensusMsgQueue

	eventBus *tmtypes.EventBus

	logger types.Logger
}

// NewExecutor creates new instance of BlockExecutor.
// localAddress will be used in sequencer mode only.
func NewExecutor(
	localAddress []byte,
	chainID string,
	mempool mempool.Mempool,
	proxyApp proxy.AppConns,
	eventBus *tmtypes.EventBus,
	consensusMsgsQueue *ConsensusMsgQueue,
	logger types.Logger,
) (ExecutorI, error) {
	be := Executor{
		localAddress:          localAddress,
		chainID:               chainID,
		proxyAppConsensusConn: proxyApp.Consensus(),
		proxyAppQueryConn:     proxyApp.Query(),
		mempool:               mempool,
		eventBus:              eventBus,
		consensusMsgQueue:     consensusMsgsQueue,
		logger:                logger,
	}
	return &be, nil
}

// AddConsensusMsgs adds new consensus msgs to the queue.
// The method is thread-safe.
func (e *Executor) AddConsensusMsgs(msgs ...proto2.Message) {
	e.consensusMsgQueue.Add(msgs...)
}

// GetConsensusMsgs dequeues consensus msgs from the queue.
// The method is thread-safe.
func (e *Executor) GetConsensusMsgs() []proto2.Message {
	return e.consensusMsgQueue.Get()
}

// InitChain calls InitChainSync using consensus connection to app.
func (e *Executor) InitChain(genesis *tmtypes.GenesisDoc, genesisChecksum string, valset []*tmtypes.Validator) (*abci.ResponseInitChain, error) {
	valUpdates := abci.ValidatorUpdates{}

	// prepare the validator updates as expected by the ABCI app
	for _, validator := range valset {
		tmkey, err := tmcrypto.PubKeyToProto(validator.PubKey)
		if err != nil {
			return nil, err
		}

		valUpdates = append(valUpdates, abci.ValidatorUpdate{
			PubKey: tmkey,
			Power:  validator.VotingPower,
		})
	}

	params := genesis.ConsensusParams

	return e.proxyAppConsensusConn.InitChainSync(abci.RequestInitChain{
		Time:    genesis.GenesisTime,
		ChainId: genesis.ChainID,
		ConsensusParams: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxBytes: params.Block.MaxBytes,
				MaxGas:   params.Block.MaxGas,
			},
			Evidence: &tmproto.EvidenceParams{
				MaxAgeNumBlocks: params.Evidence.MaxAgeNumBlocks,
				MaxAgeDuration:  params.Evidence.MaxAgeDuration,
				MaxBytes:        params.Evidence.MaxBytes,
			},
			Validator: &tmproto.ValidatorParams{
				PubKeyTypes: params.Validator.PubKeyTypes,
			},
			Version: &tmproto.VersionParams{
				AppVersion: params.Version.AppVersion,
			},
		}, Validators:   valUpdates,
		AppStateBytes:   genesis.AppState,
		InitialHeight:   genesis.InitialHeight,
		GenesisChecksum: genesisChecksum,
	})
}

// CreateBlock reaps transactions from mempool and builds a block.
func (e *Executor) CreateBlock(
	height uint64,
	lastCommit *types.Commit,
	lastHeaderHash, nextSeqHash [32]byte,
	state *types.State,
	maxBlockDataSizeBytes uint64,
) *types.Block {
	maxBlockDataSizeBytes = min(maxBlockDataSizeBytes, uint64(max(minBlockMaxBytes, state.ConsensusParams.Block.MaxBytes))) //nolint:gosec // MaxBytes is always positive
	mempoolTxs := e.mempool.ReapMaxBytesMaxGas(int64(maxBlockDataSizeBytes), state.ConsensusParams.Block.MaxGas)            //nolint:gosec // size is always positive and falls in int64

	block := &types.Block{
		Header: types.Header{
			Version: types.Version{
				Block: state.Version.Consensus.Block,
				App:   state.Version.Consensus.App,
			},
			ChainID:         e.chainID,
			Height:          height,
			Time:            time.Now().UTC().UnixNano(),
			LastHeaderHash:  lastHeaderHash,
			DataHash:        [32]byte{},
			ConsensusHash:   [32]byte{},
			AppHash:         state.AppHash,
			LastResultsHash: state.LastResultsHash,
			ProposerAddress: e.localAddress,
		},
		Data: types.Data{
			Txs:               toDymintTxs(mempoolTxs),
			ConsensusMessages: protoutils.FromProtoMsgSliceToAnySlice(e.consensusMsgQueue.Get()...),
		},
		LastCommit: *lastCommit,
	}

	block.Header.SetDymHeader(types.MakeDymHeader(block.Data.ConsensusMessages))
	copy(block.Header.LastCommitHash[:], types.GetLastCommitHash(lastCommit, &block.Header))
	copy(block.Header.DataHash[:], types.GetDataHash(block))
	copy(block.Header.SequencerHash[:], state.GetProposerHash())
	copy(block.Header.NextSequencersHash[:], nextSeqHash[:])
	return block
}

// Commit commits the block
func (e *Executor) Commit(state *types.State, block *types.Block, resp *tmstate.ABCIResponses) ([]byte, int64, error) {
	appHash, retainHeight, err := e.commit(state, block, resp.DeliverTxs)
	if err != nil {
		return nil, 0, err
	}

	err = e.publishEvents(resp, block)
	if err != nil {
		e.logger.Error("fire block events", "error", err)
		return nil, 0, err
	}
	return appHash, retainHeight, nil
}

// GetAppInfo returns the latest AppInfo from the proxyApp.
func (e *Executor) GetAppInfo() (*abci.ResponseInfo, error) {
	return e.proxyAppQueryConn.InfoSync(abci.RequestInfo{})
}

func (e *Executor) commit(state *types.State, block *types.Block, deliverTxs []*abci.ResponseDeliverTx) ([]byte, int64, error) {
	e.mempool.Lock()
	defer e.mempool.Unlock()

	err := e.mempool.FlushAppConn()
	if err != nil {
		return nil, 0, err
	}

	resp, err := e.proxyAppConsensusConn.CommitSync()
	if err != nil {
		return nil, 0, err
	}

	maxBytes := state.ConsensusParams.Block.MaxBytes
	maxGas := state.ConsensusParams.Block.MaxGas
	err = e.mempool.Update(int64(block.Header.Height), fromDymintTxs(block.Data.Txs), deliverTxs) //nolint:gosec // height is non-negative and falls in int64
	if err != nil {
		return nil, 0, err
	}
	e.mempool.SetPreCheckFn(mempool.PreCheckMaxBytes(maxBytes))
	e.mempool.SetPostCheckFn(mempool.PostCheckMaxGas(maxGas))

	return resp.Data, resp.RetainHeight, err
}

// ExecuteBlock executes the block and returns the ABCIResponses. Block should be valid (passed validation checks).
func (e *Executor) ExecuteBlock(block *types.Block) (*tmstate.ABCIResponses, error) {
	abciResponses := new(tmstate.ABCIResponses)
	abciResponses.DeliverTxs = make([]*abci.ResponseDeliverTx, len(block.Data.Txs))

	txIdx := 0
	validTxs := 0
	invalidTxs := 0

	var err error

	e.proxyAppConsensusConn.SetResponseCallback(func(req *abci.Request, res *abci.Response) {
		if r, ok := res.Value.(*abci.Response_DeliverTx); ok {
			txRes := r.DeliverTx
			if txRes.Code == abci.CodeTypeOK {
				validTxs++
			} else {
				e.logger.Debug("Invalid tx", "code", txRes.Code, "log", txRes.Log)
				invalidTxs++
			}
			abciResponses.DeliverTxs[txIdx] = txRes
			txIdx++
		}
	})

	hash := block.Hash()
	abciHeader := types.ToABCIHeaderPB(&block.Header)
	abciResponses.BeginBlock, err = e.proxyAppConsensusConn.BeginBlockSync(
		abci.RequestBeginBlock{
			Hash:   hash[:],
			Header: abciHeader,
			LastCommitInfo: abci.LastCommitInfo{
				Round: 0,
				Votes: nil,
			},
			ByzantineValidators: nil,
			ConsensusMessages:   block.Data.ConsensusMessages,
		})
	if err != nil {
		return nil, err
	}

	for _, tx := range block.Data.Txs {
		res := e.proxyAppConsensusConn.DeliverTxAsync(abci.RequestDeliverTx{Tx: tx})
		if res.GetException() != nil {
			return nil, errors.New(res.GetException().GetError())
		}
	}

	abciResponses.EndBlock, err = e.proxyAppConsensusConn.EndBlockSync(abci.RequestEndBlock{Height: int64(block.Header.Height)}) //nolint:gosec // height is non-negative and falls in int64
	if err != nil {
		return nil, err
	}

	return abciResponses, nil
}

func (e *Executor) publishEvents(resp *tmstate.ABCIResponses, block *types.Block) error {
	if e.eventBus == nil {
		return nil
	}

	abciBlock, err := types.ToABCIBlock(block)
	if err != nil {
		return err
	}

	err = multierr.Append(err, e.eventBus.PublishEventNewBlock(tmtypes.EventDataNewBlock{
		Block:            abciBlock,
		ResultBeginBlock: *resp.BeginBlock,
		ResultEndBlock:   *resp.EndBlock,
	}))
	err = multierr.Append(err, e.eventBus.PublishEventNewBlockHeader(tmtypes.EventDataNewBlockHeader{
		Header:           abciBlock.Header,
		NumTxs:           int64(len(abciBlock.Txs)),
		ResultBeginBlock: *resp.BeginBlock,
		ResultEndBlock:   *resp.EndBlock,
	}))
	for _, ev := range abciBlock.Evidence.Evidence {
		err = multierr.Append(err, e.eventBus.PublishEventNewEvidence(tmtypes.EventDataNewEvidence{
			Evidence: ev,
			Height:   int64(block.Header.Height), //nolint:gosec // height is non-negative and falls in int64
		}))
	}
	for i, dtx := range resp.DeliverTxs {
		err = multierr.Append(err, e.eventBus.PublishEventTx(tmtypes.EventDataTx{
			TxResult: abci.TxResult{
				Height: int64(block.Header.Height), //nolint:gosec // block height is within int64 range
				Index:  uint32(i),                  //nolint:gosec // num of deliver txs is less than 2^32
				Tx:     abciBlock.Data.Txs[i],
				Result: *dtx,
			},
		}))
	}
	return err
}

func toDymintTxs(txs tmtypes.Txs) types.Txs {
	optiTxs := make(types.Txs, len(txs))
	for i := range txs {
		optiTxs[i] = []byte(txs[i])
	}
	return optiTxs
}

func fromDymintTxs(optiTxs types.Txs) tmtypes.Txs {
	txs := make(tmtypes.Txs, len(optiTxs))
	for i := range optiTxs {
		txs[i] = []byte(optiTxs[i])
	}
	return txs
}
