package block

import (
	"bytes"
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

		ConsensusParams:                  *genDoc.ConsensusParams,
		LastHeightConsensusParamsChanged: genDoc.InitialHeight,
	}
	s.SetHeight(0)
	copy(s.AppHash[:], genDoc.AppHash)

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

	// update the state with the app hashes created on the app commit
	m.Executor.UpdateStateAfterCommit(m.State, resp, proxyAppInfo.LastBlockAppHash, appHeight)
	return nil
}

func (e *Executor) UpdateStateAfterInitChain(s *types.State, res *abci.ResponseInitChain) {
	// If the app did not return an app hash, we keep the one set from the genesis doc in
	// the state. We don't set appHash since we don't want the genesis doc app hash
	// recorded in the genesis block. We should probably just remove GenesisDoc.AppHash.
	if len(res.AppHash) > 0 {
		copy(s.AppHash[:], res.AppHash)
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
}

func (e *Executor) UpdateMempoolAfterInitChain(s *types.State) {
	e.mempool.SetPreCheckFn(mempool.PreCheckMaxBytes(s.ConsensusParams.Block.MaxBytes))
	e.mempool.SetPostCheckFn(mempool.PostCheckMaxGas(s.ConsensusParams.Block.MaxGas))
}

// UpdateStateAfterCommit updates the state with the app hash and last results hash
func (e *Executor) UpdateStateAfterCommit(s *types.State, resp *tmstate.ABCIResponses, appHash []byte, height uint64) {
	copy(s.AppHash[:], appHash[:])
	copy(s.LastResultsHash[:], tmtypes.NewResults(resp.DeliverTxs).Hash())

	// TODO: load consensus params from endblock?

	s.SetHeight(height)
}

// UpdateProposerFromBlock updates the proposer from the block
// in case of proposer change, the existing proposer sets the nextProposerHash in the block header
func (e *Executor) UpdateProposerFromBlock(s *types.State, block *types.Block) bool {
	// no sequencer change
	if bytes.Equal(block.Header.SequencerHash[:], block.Header.NextSequencersHash[:]) {
		return false
	}

	if block.Header.NextSequencersHash == [32]byte{} {
		// the chain will be halted until proposer is set
		e.logger.Info("rollapp left with no proposer. chain is halted")
		s.Sequencers.SetProposer(nil)
		return true
	}

	// if hash changed, update the active sequencer
	err := s.Sequencers.SetProposerByHash(block.Header.NextSequencersHash[:])
	if err != nil {
		e.logger.Error("update new proposer", "err", err)
		panic(fmt.Sprintf("failed to update new proposer: %v", err))
	}

	val := s.Sequencers.GetByConsAddress(e.localAddress)
	if val == nil {
		e.logger.Error("local key not found in sequencer set")
		panic("local key not found in sequencer set")
	}

	if bytes.Equal(types.GetHash(val), block.Header.NextSequencersHash[:]) {
		e.logger.Info("node changing to proposer role")
		return true
	}

	return false
}
