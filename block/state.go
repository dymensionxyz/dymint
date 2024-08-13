package block

import (
<<<<<<< HEAD
	"bytes"
	"encoding/json"
=======
>>>>>>> c8f53bb (new params proto)
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/cometbft/cometbft/crypto/merkle"
	abci "github.com/tendermint/tendermint/abci/types"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/version"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

// LoadStateOnInit tries to load lastState from Store, and if it's not available it reads GenesisDoc.
func (m *Manager) LoadStateOnInit(store store.Store, genesis *tmtypes.GenesisDoc, logger types.Logger) error {
	s, err := store.LoadState()
	if errors.Is(err, types.ErrNoStateFound) {
		logger.Info("failed to find state in the store, creating new state from genesis")
		s, err = NewStateFromGenesis(genesis)
	}

	if err != nil {
		return fmt.Errorf("get initial state: %w", err)
	}

	m.State = s
	return nil
}

// NewStateFromGenesis reads blockchain State from genesis.
// The active sequencer list will be set on InitChain
func NewStateFromGenesis(genDoc *tmtypes.GenesisDoc) (*types.State, error) {
	err := genDoc.ValidateAndComplete()
	if err != nil {
		return nil, fmt.Errorf("in genesis doc: %w", err)
	}

	// InitStateVersion sets the Consensus.Block and Software versions,
	// but leaves the Consensus.App version blank.
	// The Consensus.App version will be set during the Handshake, once
	// we hear from the app what protocol version it is running.
	InitStateVersion := tmstate.Version{
		Consensus: tmversion.Consensus{
			Block: version.BlockProtocol,
			App:   0,
		},
		Software: version.TMCoreSemVer,
	}

	s := types.State{
		Version: InitStateVersion,
		ChainID: genDoc.ChainID,

		InitialHeight: uint64(genDoc.InitialHeight),
		BaseHeight:    uint64(genDoc.InitialHeight),

		LastHeightConsensusParamsChanged: genDoc.InitialHeight,
	}
	s.SetHeight(0)
	copy(s.AppHash[:], genDoc.AppHash)

	err = s.LoadConsensusFromAppState(genDoc.AppState)
	if err != nil {
		return nil, fmt.Errorf("in genesis doc: %w", err)
	}

	return &s, nil
}

// UpdateStateFromApp is responsible for aligning the state of the store from the abci app
func (m *Manager) UpdateStateFromApp() error {
	proxyAppInfo, err := m.Executor.GetAppInfo()
	if err != nil {
		return errorsmod.Wrap(err, "get app info")
	}

	appHeight := uint64(proxyAppInfo.LastBlockHeight)
	resp, err := m.Store.LoadBlockResponses(appHeight)
	if err != nil {
		return errorsmod.Wrap(err, "load block responses")
	}

<<<<<<< HEAD
	// update the state with the app hashes created on the app commit
	m.Executor.UpdateStateAfterCommit(m.State, resp, proxyAppInfo.LastBlockAppHash, appHeight)
	return nil
=======
	// update the state with the hash, last store height and last validators.
	m.Executor.UpdateStateAfterCommit(m.State, resp, proxyAppInfo.LastBlockAppHash, appHeight, vals)
	_, err = m.Store.SaveState(m.State, nil)
	if err != nil {
		return errorsmod.Wrap(err, "update state")
	}
<<<<<<< HEAD
	return stateUpdateErr
>>>>>>> 1e690cc (version checks)
=======
	return nil
>>>>>>> d4a0646 (renaming + fix)
}

func (e *Executor) UpdateStateAfterInitChain(s *types.State, res *abci.ResponseInitChain) {
	// If the app did not return an app hash, we keep the one set from the genesis doc in
	// the state. We don't set appHash since we don't want the genesis doc app hash
	// recorded in the genesis block. We should probably just remove GenesisDoc.AppHash.
	if len(res.AppHash) > 0 {
		copy(s.AppHash[:], res.AppHash)
	}

	// We update the last results hash with the empty hash, to conform with RFC-6962.
	copy(s.LastResultsHash[:], merkle.HashFromByteSlices(nil))
}

func (e *Executor) UpdateMempoolAfterInitChain(s *types.State) {
	e.mempool.SetPreCheckFn(mempool.PreCheckMaxBytes(s.ConsensusParams.Params.BlockMaxSize))
	e.mempool.SetPostCheckFn(mempool.PostCheckMaxGas(s.ConsensusParams.Params.BlockMaxGas))
}

// UpdateStateAfterCommit updates the state with the app hash and last results hash
func (e *Executor) UpdateStateAfterCommit(s *types.State, resp *tmstate.ABCIResponses, appHash []byte, height uint64) {

	copy(s.AppHash[:], appHash[:])
	copy(s.LastResultsHash[:], tmtypes.NewResults(resp.DeliverTxs).Hash())

	s.SetHeight(height)

	if resp.EndBlock.RollappConsensusParamUpdates == nil {
		return
	}
	s.ConsensusParams.Params.BlockMaxSize = resp.EndBlock.RollappConsensusParamUpdates.Block.MaxBytes
	s.ConsensusParams.Params.BlockMaxGas = resp.EndBlock.RollappConsensusParamUpdates.Block.MaxGas
	s.ConsensusParams.Params.Da = resp.EndBlock.RollappConsensusParamUpdates.Da
	s.ConsensusParams.Params.Commit = resp.EndBlock.RollappConsensusParamUpdates.Commit

	return
}

// UpdateProposerFromBlock updates the proposer from the block
// The next proposer is defined in the block header (NextSequencersHash)
// In case of a node that a becomes the proposer, we return true to mark the role change
// currently the node will rebooted to apply the new role
// TODO: (https://github.com/dymensionxyz/dymint/issues/1008)
func (e *Executor) UpdateProposerFromBlock(s *types.State, block *types.Block) bool {
	// no sequencer change
	if bytes.Equal(block.Header.SequencerHash[:], block.Header.NextSequencersHash[:]) {
		return false
	}

	if block.Header.NextSequencersHash == [32]byte{} {
		// the chain will be halted until proposer is set
		// TODO: recover from halt (https://github.com/dymensionxyz/dymint/issues/1021)
		e.logger.Info("rollapp left with no proposer. chain is halted")
		s.Sequencers.SetProposer(nil)
		return false
	}

	// if hash changed, update the active sequencer
	err := s.Sequencers.SetProposerByHash(block.Header.NextSequencersHash[:])
	if err != nil {
		e.logger.Error("update new proposer", "err", err)
		panic(fmt.Sprintf("failed to update new proposer: %v", err))
	}

	localSeq := s.Sequencers.GetByConsAddress(e.localAddress)
	if localSeq != nil && bytes.Equal(localSeq.Hash(), block.Header.NextSequencersHash[:]) {
		return true
	}

	return false
}
