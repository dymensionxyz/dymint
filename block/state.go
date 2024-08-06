package block

import (
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
		Version:       InitStateVersion,
		ChainID:       genDoc.ChainID,
		InitialHeight: uint64(genDoc.InitialHeight),

		BaseHeight: uint64(genDoc.InitialHeight),

		LastHeightValidatorsChanged: genDoc.InitialHeight,

		LastHeightConsensusParamsChanged: genDoc.InitialHeight,
	}
	// s.SetConsensusParamsFromAppState(genDoc.AppState)
	s.LastBlockHeight.Store(0)
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
	vals, err := m.Store.LoadValidators(appHeight)
	if err != nil {
		return errorsmod.Wrap(err, "load block responses")
	}

	// update the state with the hash, last store height and last validators.
	stateUpdateErr := m.Executor.UpdateStateAfterCommit(m.State, resp, proxyAppInfo.LastBlockAppHash, appHeight, vals)
	_, err = m.Store.SaveState(m.State, nil)
	if err != nil {
		return errorsmod.Wrap(err, "update state")
	}
	return stateUpdateErr
}

func (e *Executor) UpdateStateAfterInitChain(s *types.State, res *abci.ResponseInitChain, validators []*tmtypes.Validator) {
	// If the app did not return an app hash, we keep the one set from the genesis doc in
	// the state. We don't set appHash since we don't want the genesis doc app hash
	// recorded in the genesis block. We should probably just remove GenesisDoc.AppHash.
	if len(res.AppHash) > 0 {
		copy(s.AppHash[:], res.AppHash)
	}

	// The validators after initChain must be greater than zero, otherwise this state is not loadable
	if len(validators) <= 0 {
		panic("Validators must be greater than zero")
	}

	// We update the last results hash with the empty hash, to conform with RFC-6962.
	copy(s.LastResultsHash[:], merkle.HashFromByteSlices(nil))

	// Set the validators in the state
	s.Validators = tmtypes.NewValidatorSet(validators).CopyIncrementProposerPriority(1)
	s.NextValidators = s.Validators.Copy()
}

func (e *Executor) UpdateMempoolAfterInitChain(s *types.State) {
	e.mempool.SetPreCheckFn(mempool.PreCheckMaxBytes(s.ConsensusParams.Params.BlockMaxSize))
	e.mempool.SetPostCheckFn(mempool.PostCheckMaxGas(s.ConsensusParams.Params.BlockMaxGas))
}

// UpdateStateAfterCommit using commit response
func (e *Executor) UpdateStateAfterCommit(s *types.State, resp *tmstate.ABCIResponses, appHash []byte, height uint64, valSet *tmtypes.ValidatorSet) error {
	copy(s.AppHash[:], appHash[:])
	copy(s.LastResultsHash[:], tmtypes.NewResults(resp.DeliverTxs).Hash())

	s.Validators = s.NextValidators.Copy()
	s.NextValidators = valSet.Copy()

	s.SetHeight(height)

	if resp.EndBlock.RollappConsensusParamUpdates == nil {
		return nil
	}
	s.ConsensusParams.Params.BlockMaxSize = resp.EndBlock.RollappConsensusParamUpdates.Block.MaxBytes
	s.ConsensusParams.Params.BlockMaxGas = resp.EndBlock.RollappConsensusParamUpdates.Block.MaxGas

	var err error
	if s.ConsensusParams.Params.Da != resp.EndBlock.RollappConsensusParamUpdates.Da {
		e.logger.Debug("Updating DA", "da", s.ConsensusParams.Params.Da, "newda", resp.EndBlock.RollappConsensusParamUpdates.Da)
		s.ConsensusParams.Params.Da = resp.EndBlock.RollappConsensusParamUpdates.Da
		err = fmt.Errorf("%w, please update da config for %s", ErrDAUpgrade, s.ConsensusParams.Params.Da)
	}
	if s.ConsensusParams.Params.Commit != resp.EndBlock.RollappConsensusParamUpdates.Commit {
		e.logger.Debug("Updating version", "version", s.ConsensusParams.Params.Commit, "version", resp.EndBlock.RollappConsensusParamUpdates.Commit)
		s.ConsensusParams.Params.Commit = resp.EndBlock.RollappConsensusParamUpdates.Commit
		err = fmt.Errorf("%w, please upgrade binary to commit %s", ErrVersionUpgrade, s.ConsensusParams.Params.Commit)

	}
	return err
}
