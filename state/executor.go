package state

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
}

// NewBlockExecutor creates new instance of BlockExecutor.
// Proposer address and namespace ID will be used in all newly created blocks.
func NewBlockExecutor(proposerAddress []byte, namespaceID string, chainID string, mempool mempool.Mempool, proxyApp proxy.AppConns, eventBus *tmtypes.EventBus, logger log.Logger) (*BlockExecutor, error) {
	bytes, err := hex.DecodeString(namespaceID)
	if err != nil {
		return nil, err
	}

	var be = BlockExecutor{
		proposerAddress:       proposerAddress,
		chainID:               chainID,
		proxyAppConsensusConn: proxyApp.Consensus(),
		proxyAppQueryConn:     proxyApp.Query(),
		mempool:               mempool,
		eventBus:              eventBus,
		logger:                logger,
	}
	copy(be.namespaceID[:], bytes)
	return &be, nil
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
	//Dymint ignores any setValidator responses from the app, as it is manages the validator set based on the settlement consensus
	//TODO: this will be changed when supporting multiple sequencers from the hub
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

	//TEST FRAUDS
	getAppHashRes, err := e.proxyAppConsensusConn.GetAppHashSync(abcitypes.RequestGetAppHash{})
	if err != nil {
		return 0, err
	}
	e.logger.Info("AppHash before commit", "appHash", hex.EncodeToString(getAppHashRes.GetAppHash()))

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

	e.logger.Info("commit response", "appHash", hex.EncodeToString(state.AppHash[:]))

	getAppHashRes, err = e.proxyAppConsensusConn.GetAppHashSync(abcitypes.RequestGetAppHash{})
	if err != nil {
		return 0, err
	}
	e.logger.Info("AppHash after commit", "appHash", hex.EncodeToString(getAppHashRes.GetAppHash()))
	e.logger.Info("header appHash", "appHash", hex.EncodeToString(block.Header.AppHash[:]))
	// e.proxyAppConsensusConn.GenerateFraudProofSync(abcitypes.RequestGenerateFraudProof{
	// 	BeginBlockRequest: resp.BeginBlock,
	// )
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
	//TODO: we can probably pass the state as a pointer and update it directly
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

func (e *BlockExecutor) setOrVerifyISR(phase string, ISRs [][]byte, generateISR bool, idx int) ([][]byte, error) {
	isr, err := e.getAppHash()
	if err != nil {
		return nil, err
	}

	if generateISR {
		//sequencer mode
		ISRs = append(ISRs, isr)
		e.logger.Info(phase, "ISR", hex.EncodeToString(isr))
		return ISRs, nil
	} else {
		//verifier mode
		e.logger.Info("verifying ISR", "phase", phase)
		if !bytes.Equal(isr, ISRs[idx]) {
			e.logger.Error(phase, "ISR mismatch", "ISR", hex.EncodeToString(isr), "expected", hex.EncodeToString(ISRs[idx]))
			return nil, types.ErrInvalidISR
		}
		e.logger.Info(phase, "ISR", hex.EncodeToString(isr))
		return nil, nil
	}
}

// Execute executes the block and returns the ABCIResponses.
func (e *BlockExecutor) Execute(ctx context.Context, state types.State, block *types.Block) (*tmstate.ABCIResponses, error) {
	abciResponses := new(tmstate.ABCIResponses)
	abciResponses.DeliverTxs = make([]*abci.ResponseDeliverTx, len(block.Data.Txs))

	txIdx := 0
	validTxs := 0
	invalidTxs := 0

	expectedISRCount := 3 + len(block.Data.Txs) // initial, beginblock, len(delivertxs), endblock
	currISRIdx := 0
	var generateISR bool //sequencer mode. otherwise it's verifier mode

	var err error

	//FIXME: validate ISRs exists for full node / when syncing
	//FIXME: the enable fraud proofs flag should be "height guarded" to allow syncing after toggling
	/*
		if e.fraudProofsEnabled && fullnode && blockISR == nil {
	*/

	ISRs := make([][]byte, expectedISRCount)
	blockISRs := block.Data.IntermediateStateRoots.RawRootsList
	if blockISRs != nil {
		if len(blockISRs) != expectedISRCount {
			e.logger.Error("ISR count mismatch", "expected", expectedISRCount, "actual", len(blockISRs))
			return nil, errors.New("ISR count mismatch")
		}
		generateISR = false
		ISRs = blockISRs
	} else {
		//FIXME: make sure we're the aggregator
		generateISR = true
	}

	//FIXME: return and handle types.ErrInvalidISR for proof generation
	e.setOrVerifyISR("initial ISR", ISRs, generateISR, currISRIdx)
	currISRIdx++

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

	e.setOrVerifyISR("begin block ISR", ISRs, generateISR, currISRIdx)
	currISRIdx++

	deliverTxRequests := make([]*abci.RequestDeliverTx, 0, len(block.Data.Txs))
	for _, tx := range block.Data.Txs {
		deliverTxRequest := abci.RequestDeliverTx{Tx: tx}
		deliverTxRequests = append(deliverTxRequests, &deliverTxRequest)
		res := e.proxyAppConsensusConn.DeliverTxAsync(deliverTxRequest)
		if res.GetException() != nil {
			return nil, errors.New(res.GetException().GetError())
		}

		e.setOrVerifyISR("deliver TX ISR", ISRs, generateISR, currISRIdx)
		currISRIdx++
	}

	reqEndBlock := abci.RequestEndBlock{Height: int64(block.Header.Height)}
	abciResponses.EndBlock, err = e.proxyAppConsensusConn.EndBlockSync(reqEndBlock)
	if err != nil {
		return nil, err
	}

	e.setOrVerifyISR("endblock ISR", ISRs, generateISR, currISRIdx)
	currISRIdx++

	if len(block.Data.Txs) > 0 && block.Header.Height > 5 {
		fraud, err := e.generateFraudProof(&reqBeginBlock, deliverTxRequests, &reqEndBlock)
		if err != nil {
			e.logger.Error("failed to generate fraud proof", "error", err)
			return nil, err
		}

		if fraud == nil {
			e.logger.Error("fraud proof is nil")
			return nil, errors.New("fraud proof is nil")
		}

		// Open a new file for writing only
		file, err := os.Create("fraudProof_rollapp_with_tx.json")
		if err != nil {
			return nil, err
		}
		defer file.Close()

		// Serialize the struct to JSON and write it to the file
		jsonEncoder := json.NewEncoder(file)
		err = jsonEncoder.Encode(fraud)
		if err != nil {
			return nil, err
		}

		e.logger.Info("fraud proof generated")
		panic("fraud proof generated")
	}

	//TODO: add 'if enabled'
	if blockISRs == nil {
		//we've already valdiated we're the aggreator in this case
		//double checking we've produced ISRs as expected
		if currISRIdx != expectedISRCount || len(ISRs) != expectedISRCount {
			e.logger.Error("ISR count mismatch", "expected", expectedISRCount, "actual", currISRIdx)
			return nil, errors.New("ISR count mismatch")
		}
		e.logger.Info("setting ISRs", "ISRs", ISRs)
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

func (e *BlockExecutor) getAppHash() ([]byte, error) {
	isrResp, err := e.proxyAppConsensusConn.GetAppHashSync(abci.RequestGetAppHash{})
	if err != nil {
		return nil, err
	}
	return isrResp.AppHash, nil
}

func (e *BlockExecutor) generateFraudProof(beginBlockRequest *abci.RequestBeginBlock, deliverTxRequests []*abci.RequestDeliverTx, endBlockRequest *abci.RequestEndBlock) (*abci.FraudProof, error) {
	generateFraudProofRequest := abci.RequestGenerateFraudProof{}
	if beginBlockRequest == nil {
		return nil, fmt.Errorf("begin block request cannot be a nil parameter")
	}
	generateFraudProofRequest.BeginBlockRequest = *beginBlockRequest
	if deliverTxRequests != nil {
		generateFraudProofRequest.DeliverTxRequests = deliverTxRequests
	}

	//FIXME: HACK: always set fraudelent ST as the deliverTxRequest
	// generateFraudProofRequest.EndBlockRequest = endBlockRequest

	resp, err := e.proxyAppConsensusConn.GenerateFraudProofSync(generateFraudProofRequest)
	if err != nil {
		return nil, err
	}
	if resp.FraudProof == nil {
		return nil, fmt.Errorf("fraud proof generation failed")
	}
	return resp.FraudProof, nil
}

// func validateValidatorUpdates(abciUpdates []abci.ValidatorUpdate,
// 	params tmproto.ValidatorParams) error {
// 	for _, valUpdate := range abciUpdates {
// 		if valUpdate.GetPower() < 0 {
// 			return fmt.Errorf("voting power can't be negative %v", valUpdate)
// 		} else if valUpdate.GetPower() == 0 {
// 			// continue, since this is deleting the validator, and thus there is no
// 			// pubkey to check
// 			continue
// 		}

// 		// Check if validator's pubkey matches an ABCI type in the consensus params
// 		pk, err := cryptoenc.PubKeyFromProto(valUpdate.PubKey)
// 		if err != nil {
// 			return err
// 		}

// 		if !tmtypes.IsValidPubkeyType(params, pk.Type()) {
// 			return fmt.Errorf("validator %v is using pubkey %s, which is unsupported for consensus",
// 				valUpdate, pk.Type())
// 		}
// 	}
// 	return nil
// }
