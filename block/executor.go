package block

import (
	"errors"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto/encoding"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"
	"go.uber.org/multierr"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/types"
)

// Executor creates and applies blocks and maintains state.
type Executor struct {
	localAddress          []byte
	chainID               string
	proxyAppConsensusConn proxy.AppConnConsensus
	proxyAppQueryConn     proxy.AppConnQuery
	mempool               mempool.Mempool

	eventBus *tmtypes.EventBus

	logger types.Logger
}

// NewExecutor creates new instance of BlockExecutor.
// localAddress will be used in sequencer mode only.
func NewExecutor(localAddress []byte, chainID string, mempool mempool.Mempool, proxyApp proxy.AppConns, eventBus *tmtypes.EventBus, logger types.Logger) (*Executor, error) {
	be := Executor{
		localAddress:          localAddress,
		chainID:               chainID,
		proxyAppConsensusConn: proxyApp.Consensus(),
		proxyAppQueryConn:     proxyApp.Query(),
		mempool:               mempool,
		eventBus:              eventBus,
		logger:                logger,
	}
	return &be, nil
}

// InitChain calls InitChainSync using consensus connection to app.
func (e *Executor) InitChain(genesis *tmtypes.GenesisDoc, valset []*tmtypes.Validator) (*abci.ResponseInitChain, error) {
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
		},
		Validators:    valUpdates,
		AppStateBytes: genesis.AppState,
		InitialHeight: genesis.InitialHeight,
	})
}

// CreateBlock reaps transactions from mempool and builds a block.
func (e *Executor) CreateBlock(height uint64, lastCommit *types.Commit, lastHeaderHash, nextSeqHash [32]byte, state *types.State, maxBlockDataSizeBytes uint32) *types.Block {
	if state.RollappParams.Blockmaxbytes > 0 {
		maxBlockDataSizeBytes = min(maxBlockDataSizeBytes, state.RollappParams.Blockmaxbytes)
	}
	mempoolTxs := e.mempool.ReapMaxBytesMaxGas(int64(maxBlockDataSizeBytes), state.ConsensusParams.Block.MaxGas)

	block := &types.Block{
		Header: types.Header{
			Version: types.Version{
				Block: state.Version.Consensus.Block,
				App:   state.Version.Consensus.App,
			},
			ChainID:         e.chainID,
			Height:          height,
			Time:            uint64(time.Now().UTC().UnixNano()),
			LastHeaderHash:  lastHeaderHash,
			DataHash:        [32]byte{},
			ConsensusHash:   [32]byte{},
			AppHash:         state.AppHash,
			LastResultsHash: state.LastResultsHash,
			ProposerAddress: e.localAddress,
		},
		Data: types.Data{
			Txs:                    toDymintTxs(mempoolTxs),
			IntermediateStateRoots: types.IntermediateStateRoots{RawRootsList: nil},
			Evidence:               types.EvidenceData{Evidence: nil},
		},
		LastCommit: *lastCommit,
	}
	copy(block.Header.LastCommitHash[:], types.GetLastCommitHash(lastCommit, &block.Header))
	copy(block.Header.DataHash[:], types.GetDataHash(block))
	copy(block.Header.SequencerHash[:], state.Sequencers.ProposerHash())
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

	maxBytes := state.RollappParams.Blockmaxbytes
	maxGas := state.ConsensusParams.Block.MaxGas
	err = e.mempool.Update(int64(block.Header.Height), fromDymintTxs(block.Data.Txs), deliverTxs)
	if err != nil {
		return nil, 0, err
	}
	e.mempool.SetPreCheckFn(mempool.PreCheckMaxBytes(int64(maxBytes)))
	e.mempool.SetPostCheckFn(mempool.PostCheckMaxGas(maxGas))

	return resp.Data, resp.RetainHeight, err
}

// ExecuteBlock executes the block and returns the ABCIResponses. Block should be valid (passed validation checks).
func (e *Executor) ExecuteBlock(state *types.State, block *types.Block) (*tmstate.ABCIResponses, error) {
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

	abciResponses.EndBlock, err = e.proxyAppConsensusConn.EndBlockSync(abci.RequestEndBlock{Height: int64(block.Header.Height)})
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
			Height:   int64(block.Header.Height),
		}))
	}
	for i, dtx := range resp.DeliverTxs {
		err = multierr.Append(err, e.eventBus.PublishEventTx(tmtypes.EventDataTx{
			TxResult: abci.TxResult{
				Height: int64(block.Header.Height),
				Index:  uint32(i),
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
