package state

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/cometbft/cometbft/crypto/merkle"
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

func (e *BlockExecutor) UpdateStateAfterInitChain(s *types.State, res *abci.ResponseInitChain, validators []*tmtypes.Validator) {
	// If the app did not return an app hash, we keep the one set from the genesis doc in
	// the state. We don't set appHash since we don't want the genesis doc app hash
	// recorded in the genesis block. We should probably just remove GenesisDoc.AppHash.
	if len(res.AppHash) > 0 {
		copy(s.AppHash[:], res.AppHash)
	}

	//The validators after initChain must be greater than zero, otherwise this state is not loadable
	if len(validators) <= 0 {
		panic("Validators must be greater than zero")
	}

	if res.ConsensusParams != nil {
		params := res.ConsensusParams
		if params.Block != nil {
			s.ConsensusParams.Block.MaxBytes = params.Block.MaxBytes
			s.ConsensusParams.Block.MaxGas = params.Block.MaxGas
		}
		if params.Evidence != nil {
			s.ConsensusParams.Evidence.MaxAgeNumBlocks = params.Evidence.MaxAgeNumBlocks
			s.ConsensusParams.Evidence.MaxAgeDuration = params.Evidence.MaxAgeDuration
			s.ConsensusParams.Evidence.MaxBytes = params.Evidence.MaxBytes
		}
		if params.Validator != nil {
			// Copy params.Validator.PubkeyTypes, and set result's value to the copy.
			// This avoids having to initialize the slice to 0 values, and then write to it again.
			s.ConsensusParams.Validator.PubKeyTypes = append([]string{}, params.Validator.PubKeyTypes...)
		}
		if params.Version != nil {
			s.ConsensusParams.Version.AppVersion = params.Version.AppVersion
		}
		s.Version.Consensus.App = s.ConsensusParams.Version.AppVersion
	}
	// We update the last results hash with the empty hash, to conform with RFC-6962.
	copy(s.LastResultsHash[:], merkle.HashFromByteSlices(nil))

	// Set the validators in the state
	s.Validators = tmtypes.NewValidatorSet(validators).CopyIncrementProposerPriority(1)
	s.NextValidators = s.Validators.Copy()
	s.LastValidators = s.Validators.Copy()
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

// Execute executes the block and returns the ABCIResponses.
func (e *BlockExecutor) Execute(ctx context.Context, state types.State, block *types.Block) (*tmstate.ABCIResponses, error) {
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
	abciHeader := abciconv.ToABCIHeaderPB(&block.Header)
	abciHeader.ChainID = e.chainID
	abciHeader.ValidatorsHash = state.Validators.Hash()
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
