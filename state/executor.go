package state

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto/encoding"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"
	"go.uber.org/multierr"

	abciconv "github.com/dymensionxyz/dymint/conv/abci"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/types"
)

// BlockExecutor creates and applies blocks and maintains state.
type BlockExecutor struct {
	proposerAddress       []byte
	namespaceID           [8]byte
	chainID               string
	proxyAppConsensusConn proxy.AppConnConsensus
	proxyAppQueryConn     proxy.AppConnQuery
	mempool               mempool.Mempool

	eventBus *tmtypes.EventBus

	logger log.Logger

	fraudProofsEnabled bool // TODO: rename. Does this mean enabled to generate ISR, or enabled to verify and create proofs?
	simulateFraud      bool
}

// NewBlockExecutor creates new instance of BlockExecutor.
// Proposer address and namespace ID will be used in all newly created blocks.
func NewBlockExecutor(proposerAddress []byte, namespaceID string, chainID string, mempool mempool.Mempool, proxyApp proxy.AppConns, eventBus *tmtypes.EventBus, logger log.Logger, simulateFraud bool) (*BlockExecutor, error) {
	bytes, err := hex.DecodeString(namespaceID)
	if err != nil {
		return nil, err
	}

	be := BlockExecutor{
		proposerAddress:       proposerAddress,
		chainID:               chainID,
		proxyAppConsensusConn: proxyApp.Consensus(),
		proxyAppQueryConn:     proxyApp.Query(),
		mempool:               mempool,
		eventBus:              eventBus,
		logger:                logger,
		fraudProofsEnabled:    true,
		simulateFraud:         simulateFraud,
	}
	copy(be.namespaceID[:], bytes)
	return &be, nil
}

func (e *BlockExecutor) isAggregator() bool {
	return e.proposerAddress != nil
}

// InitChain calls InitChainSync using consensus connection to app.
func (e *BlockExecutor) InitChain(genesis *tmtypes.GenesisDoc, validators []*tmtypes.Validator) (*abci.ResponseInitChain, error) {
	params := genesis.ConsensusParams
	valUpates := abcitypes.ValidatorUpdates{}

	for _, validator := range validators {
		tmkey, err := tmcrypto.PubKeyToProto(validator.PubKey)
		if err != nil {
			return nil, err
		}

		valUpates = append(valUpates, abcitypes.ValidatorUpdate{
			PubKey: tmkey,
			Power:  validator.VotingPower,
		})
	}

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
		Validators:    valUpates,
		AppStateBytes: genesis.AppState,
		InitialHeight: genesis.InitialHeight,
	})
}

// CreateBlock reaps transactions from mempool and builds a block.
func (e *BlockExecutor) CreateBlock(height uint64, lastCommit *types.Commit, lastHeaderHash [32]byte, state types.State) *types.Block {
	maxBytes := state.ConsensusParams.Block.MaxBytes
	maxGas := state.ConsensusParams.Block.MaxGas

	mempoolTxs := e.mempool.ReapMaxBytesMaxGas(maxBytes, maxGas)

	block := &types.Block{
		Header: types.Header{
			Version: types.Version{
				Block: state.Version.Consensus.Block,
				App:   state.Version.Consensus.App,
			},
			ChainID:         e.chainID,
			NamespaceID:     e.namespaceID,
			Height:          height,
			Time:            uint64(time.Now().UTC().UnixNano()),
			LastHeaderHash:  lastHeaderHash,
			DataHash:        [32]byte{},
			ConsensusHash:   [32]byte{},
			AppHash:         state.AppHash,
			LastResultsHash: state.LastResultsHash,
			ProposerAddress: e.proposerAddress,
		},
		Data: types.Data{
			Txs:                    toDymintTxs(mempoolTxs),
			IntermediateStateRoots: types.IntermediateStateRoots{RawRootsList: nil},
			Evidence:               types.EvidenceData{Evidence: nil},
		},
		LastCommit: *lastCommit,
	}
	copy(block.Header.LastCommitHash[:], e.getLastCommitHash(lastCommit, &block.Header))
	copy(block.Header.DataHash[:], e.getDataHash(block))
	copy(block.Header.AggregatorsHash[:], state.Validators.Hash())

	return block
}

// Validate validates block and commit.
func (e *BlockExecutor) Validate(state types.State, block *types.Block, commit *types.Commit, proposer *types.Sequencer) error {
	if err := e.validateBlock(state, block); err != nil {
		return err
	}
	if err := e.validateCommit(proposer, commit, &block.Header); err != nil {
		return err
	}
	return nil
}

// UpdateStateFromResponses updates state based on the ABCIResponses.
func (e *BlockExecutor) UpdateStateFromResponses(resp *tmstate.ABCIResponses, state types.State, block *types.Block) (types.State, error) {
	// Dymint ignores any setValidator responses from the app, as it is manages the validator set based on the settlement consensus
	// TODO: this will be changed when supporting multiple sequencers from the hub
	validatorUpdates := []*tmtypes.Validator{}

	if state.ConsensusParams.Block.MaxBytes == 0 {
		e.logger.Error("maxBytes=0", "state.ConsensusParams.Block", state.ConsensusParams.Block)
	}

	state, err := e.updateState(state, block, resp, validatorUpdates)
	if err != nil {
		return types.State{}, err
	}

	return state, nil
}

// Commit commits the block
func (e *BlockExecutor) Commit(ctx context.Context, state *types.State, block *types.Block, resp *tmstate.ABCIResponses) (int64, error) {
	appHash, retainHeight, err := e.commit(ctx, state, block, resp.DeliverTxs)
	if err != nil {
		return 0, err
	}

	copy(state.AppHash[:], appHash[:])
	copy(state.LastResultsHash[:], tmtypes.NewResults(resp.DeliverTxs).Hash())

	err = e.publishEvents(resp, block, *state)
	if err != nil {
		e.logger.Error("failed to fire block events", "error", err)
		return 0, err
	}
	return retainHeight, nil
}

func (e *BlockExecutor) updateState(state types.State, block *types.Block, abciResponses *tmstate.ABCIResponses, validatorUpdates []*tmtypes.Validator) (types.State, error) {
	nValSet := state.NextValidators.Copy()
	lastHeightValSetChanged := state.LastHeightValidatorsChanged
	// Dymint can work without validators
	if len(nValSet.Validators) > 0 {
		if len(validatorUpdates) > 0 {
			err := nValSet.UpdateWithChangeSet(validatorUpdates)
			if err != nil {
				return state, nil
			}
			// Change results from this height but only applies to the next next height.
			lastHeightValSetChanged = int64(block.Header.Height + 1 + 1)
		}

		// TODO(tzdybal):  right now, it's for backward compatibility, may need to change this
		nValSet.IncrementProposerPriority(1)
	}

	hash := block.Header.Hash()
	// TODO: we can probably pass the state as a pointer and update it directly
	s := types.State{
		Version:         state.Version,
		ChainID:         state.ChainID,
		InitialHeight:   state.InitialHeight,
		SLStateIndex:    state.SLStateIndex,
		LastBlockHeight: int64(block.Header.Height),
		LastBlockTime:   time.Unix(0, int64(block.Header.Time)),
		LastBlockID: tmtypes.BlockID{
			Hash: hash[:],
			// for now, we don't care about part set headers
		},
		NextValidators:                   nValSet,
		Validators:                       state.NextValidators.Copy(),
		LastHeightValidatorsChanged:      lastHeightValSetChanged,
		ConsensusParams:                  state.ConsensusParams,
		LastHeightConsensusParamsChanged: state.LastHeightConsensusParamsChanged,
		// We're gonna update those fields only after we commit the blocks
		AppHash:         state.AppHash,
		LastValidators:  state.LastValidators.Copy(),
		LastStoreHeight: state.LastStoreHeight,

		LastResultsHash: state.LastResultsHash,
		BaseHeight:      state.BaseHeight,
	}

	return s, nil
}

// GetAppInfo returns the latest AppInfo from the proxyApp.
func (e *BlockExecutor) GetAppInfo() (*abcitypes.ResponseInfo, error) {
	return e.proxyAppQueryConn.InfoSync(abcitypes.RequestInfo{})
}

func (e *BlockExecutor) commit(ctx context.Context, state *types.State, block *types.Block, deliverTxs []*abci.ResponseDeliverTx) ([]byte, int64, error) {
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
	err = e.mempool.Update(int64(block.Header.Height), fromDymintTxs(block.Data.Txs), deliverTxs, mempool.PreCheckMaxBytes(maxBytes), mempool.PostCheckMaxGas(maxGas))
	if err != nil {
		return nil, 0, err
	}

	return resp.Data, resp.RetainHeight, err
}

func (e *BlockExecutor) validateBlock(state types.State, block *types.Block) error {
	err := block.ValidateBasic()
	if err != nil {
		return err
	}
	if block.Header.Version.App != state.Version.Consensus.App ||
		block.Header.Version.Block != state.Version.Consensus.Block {
		return errors.New("block version mismatch")
	}
	if state.LastBlockHeight <= 0 && block.Header.Height != uint64(state.InitialHeight) {
		return errors.New("initial block height mismatch")
	}
	if state.LastBlockHeight > 0 && block.Header.Height != uint64(state.LastStoreHeight)+1 {
		return errors.New("block height mismatch")
	}
	if !bytes.Equal(block.Header.AppHash[:], state.AppHash[:]) {
		return errors.New("AppHash mismatch")
	}
	if !bytes.Equal(block.Header.LastResultsHash[:], state.LastResultsHash[:]) {
		return errors.New("LastResultsHash mismatch")
	}

	return nil
}

func (e *BlockExecutor) validateCommit(proposer *types.Sequencer, commit *types.Commit, header *types.Header) error {
	abciHeaderPb := abciconv.ToABCIHeaderPB(header)
	abciHeaderBytes, err := abciHeaderPb.Marshal()
	if err != nil {
		return err
	}
	if err = commit.Validate(proposer, abciHeaderBytes); err != nil {
		return err
	}
	return nil
}

// Execute executes the block and returns the ABCIResponses.
func (e *BlockExecutor) Execute(ctx context.Context, state types.State, block *types.Block) (*tmstate.ABCIResponses, error) { // TODO: unused?
	var err error
	abciResponses := new(tmstate.ABCIResponses)
	abciResponses.DeliverTxs = make([]*abci.ResponseDeliverTx, len(block.Data.Txs))

	txIdx := 0
	validTxs := 0
	invalidTxs := 0

	expectedISRCount := 3 + len(block.Data.Txs) // initial, beginblock, len(delivertxs), endblock

	// TODO: wrap into struct with helper methods
	ISRs := make([][]byte, expectedISRCount)
	currISRIdx := 0

	generateISR := false // verifier mode. True implies sequencer mode

	blockISRs := block.Data.IntermediateStateRoots.RawRootsList
	if blockISRs == nil {
		if e.fraudProofsEnabled && !e.isAggregator() {
			e.logger.Error("block has no ISRs")
			return nil, types.ErrBlockMissingISR
		}
		generateISR = true
	} else {
		if len(blockISRs) != expectedISRCount {
			e.logger.Error("ISR count mismatch", "expected", expectedISRCount, "actual", len(blockISRs))
			return nil, types.ErrBlockISRCountMismatch
		}
		ISRs = blockISRs
	}

	ISRs, err = e.setOrVerifyISR("initial ISR", ISRs, generateISR, currISRIdx)
	currISRIdx++
	// not supposed to happen as no state changed yet
	if err != nil {
		return nil, err
	}

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
	abciHeader := abciconv.ToABCIHeaderPB(&block.Header)
	abciHeader.ChainID = e.chainID
	abciHeader.ValidatorsHash = state.Validators.Hash()
	reqBeginBlock := abci.RequestBeginBlock{
		Hash:   hash[:],
		Header: abciHeader,
		LastCommitInfo: abci.LastCommitInfo{
			Round: 0,
			Votes: nil,
		},
		ByzantineValidators: nil,
	}
	abciResponses.BeginBlock, err = e.proxyAppConsensusConn.BeginBlockSync(reqBeginBlock)
	if err != nil {
		return nil, err
	}

	ISRs, err = e.setOrVerifyISR("begin block ISR", ISRs, generateISR, currISRIdx)
	currISRIdx++
	if err != nil {
		if errors.Is(err, types.ErrInvalidISR) {
			e.generateFraudProof(&reqBeginBlock, nil, nil) // TODO: return early? Handle error?
		}
		return nil, err
	}

	deliverTxRequests := make([]*abci.RequestDeliverTx, 0, len(block.Data.Txs))
	for _, tx := range block.Data.Txs {
		deliverTxRequest := abci.RequestDeliverTx{Tx: tx}
		deliverTxRequests = append(deliverTxRequests, &deliverTxRequest)
		res := e.proxyAppConsensusConn.DeliverTxAsync(deliverTxRequest)
		if res.GetException() != nil {
			return nil, errors.New(res.GetException().GetError())
		}

		_, err = e.setOrVerifyISR("deliverTx", ISRs, generateISR, currISRIdx)
		currISRIdx++
		if err != nil {
			if errors.Is(err, types.ErrInvalidISR) {
				e.generateFraudProof(&reqBeginBlock, deliverTxRequests, nil) // TODO: return early? Handle error?
			}
			return nil, err
		}
	}

	reqEndBlock := abci.RequestEndBlock{Height: int64(block.Header.Height)}
	abciResponses.EndBlock, err = e.proxyAppConsensusConn.EndBlockSync(reqEndBlock)
	if err != nil {
		return nil, err
	}

	ISRs, err = e.setOrVerifyISR("endblock ISR", ISRs, generateISR, currISRIdx)
	currISRIdx++
	if err != nil {
		if errors.Is(err, types.ErrInvalidISR) {
			e.generateFraudProof(&reqBeginBlock, deliverTxRequests, &reqEndBlock) // TODO: return early? Handle error?
		}
		return nil, err
	}

	if blockISRs == nil {
		// we've already validated we're the aggregator in this case
		// double-checking we've produced ISRs as expected
		if len(ISRs) != expectedISRCount {
			e.logger.Error("ISR count mismatch", "expected", expectedISRCount, "actual", len(blockISRs))
			return nil, errors.New("ISR count mismatch")
		}
		block.Data.IntermediateStateRoots.RawRootsList = ISRs
		return abciResponses, nil
	}

	return abciResponses, nil
}

func (e *BlockExecutor) getLastCommitHash(lastCommit *types.Commit, header *types.Header) []byte {
	lastABCICommit := abciconv.ToABCICommit(lastCommit, header)
	return lastABCICommit.Hash()
}

func (e *BlockExecutor) getDataHash(block *types.Block) []byte {
	abciData := tmtypes.Data{
		Txs: abciconv.ToABCIBlockDataTxs(&block.Data),
	}
	return abciData.Hash()
}

func (e *BlockExecutor) publishEvents(resp *tmstate.ABCIResponses, block *types.Block, state types.State) error {
	if e.eventBus == nil {
		return nil
	}

	abciBlock, err := abciconv.ToABCIBlock(block)
	abciBlock.Header.ValidatorsHash = state.Validators.Hash()
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
