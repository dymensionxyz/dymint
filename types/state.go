package types

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	// TODO(tzdybal): copy to local project?

	"github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

// State contains information about current state of the blockchain.
type State struct {
	Version tmstate.Version

	// immutable
	ChainID       string
	InitialHeight uint64 // should be 1, not 0, when starting from height 1

	// LastBlockHeight=0 at genesis (ie. block(H=0) does not exist)
	LastBlockHeight atomic.Uint64

	// BaseHeight is the height of the first block we have in store after pruning.
	BaseHeight uint64

	// Sequencers is the set of sequencers that are currently active on the rollapp.
	Sequencers SequencerSet

	// Consensus parameters used for validating blocks.
	// Changes returned by EndBlock and updated after Commit.
	ConsensusParams                  tmproto.ConsensusParams
	LastHeightConsensusParamsChanged uint64

	// Merkle root of the results from executing prev block
	LastResultsHash [32]byte

	// the latest AppHash we've received from calling abci.Commit()
	AppHash [32]byte

	// New rollapp parameters .
	RollappParams dymint.RollappParams

	// LastHeaderHash is the hash of the last block header.
	LastHeaderHash [32]byte
	// The last height which the node validated from available state updates.
	LastValidatedHeight atomic.Uint64
	// The last rollapp finalized height.
	LastFinalizedHeight atomic.Uint64
	// The last DRS versions including height upgrade
	DrsVersionHistory []*dymint.DRSVersion
}

func (s *State) IsGenesis() bool {
	return s.Height() == 0
}

type RollappParams struct {
	Params *dymint.RollappParams
}

// SetHeight sets the height saved in the Store if it is higher than the existing height
// returns OK if the value was updated successfully or did not need to be updated
func (s *State) SetHeight(height uint64) {
	s.LastBlockHeight.Store(height)
}

// Height returns height of the highest block saved in the Store.
func (s *State) Height() uint64 {
	return s.LastBlockHeight.Load()
}

// SetLastValidatedHeight sets the last validated height from state updates.
func (s *State) SetLastValidatedHeight(height uint64) {
	s.LastValidatedHeight.Store(max(s.GetLastValidatedHeight(), height))
}

// GetLastValidatedHeight returns the last validated height from state updates.
func (s *State) GetLastValidatedHeight() uint64 {
	return s.LastValidatedHeight.Load()
}

// NextValidationHeight returns the next height needs to be validated from state updates.
func (s *State) NextValidationHeight() uint64 {
	return max(s.LastValidatedHeight.Load(), s.LastFinalizedHeight.Load()) + 1
}

// SetLastFinalizedHeight sets the last finalization height from finalization events.
func (s *State) SetLastFinalizedHeight(height uint64) {
	s.LastFinalizedHeight.Store(max(s.GetLastFinalizedHeight(), height))
}

// GetLastFinalizedHeight returns last finalized height.
func (s *State) GetLastFinalizedHeight() uint64 {
	return s.LastValidatedHeight.Load()
}

// NextHeight returns the next height that expected to be stored in store.
func (s *State) NextHeight() uint64 {
	if s.IsGenesis() {
		return s.InitialHeight
	}
	return s.Height() + 1
}

// SetRollappParamsFromGenesis sets the rollapp consensus params from genesis
func (s *State) SetRollappParamsFromGenesis(appState json.RawMessage) error {
	var objmap map[string]json.RawMessage
	err := json.Unmarshal(appState, &objmap)
	if err != nil {
		return err
	}
	params, ok := objmap["rollappparams"]
	if !ok {
		return fmt.Errorf("rollappparams not defined in genesis")
	}

	var rollappParams RollappParams
	err = json.Unmarshal(params, &rollappParams)
	if err != nil {
		return err
	}
	s.RollappParams = *rollappParams.Params
	s.AddDRSVersion(0, rollappParams.Params.Version)
	return nil
}

// GetDRSVersion returns the DRS version stored in rollapp params updates for a specific height.
// It is not keeping all historic but only for non-finalized height.
// If input height is already finalized it returns empty string and not found error.
func (s *State) GetDRSVersion(height uint64) (string, error) {
	if height <= s.GetLastFinalizedHeight() {
		return "", gerrc.ErrNotFound
	}
	drsVersion := ""
	for _, drs := range s.DrsVersionHistory {
		if height >= drs.Height {
			drsVersion = drs.Version
		}
	}
	return drsVersion, nil
}

// AddDRSVersion adds a new record for the DRS version update heights.
func (s *State) AddDRSVersion(height uint64, version string) {
	s.DrsVersionHistory = append(s.DrsVersionHistory, &dymint.DRSVersion{Height: height, Version: version})
}

// ClearDrsVersionHeights clears previous drs version update heights previous to finalization height,
// but keeping always the last drs version record.
func (s *State) ClearDRSVersionHeights(height uint64) {
	if len(s.DrsVersionHistory) == 1 {
		return
	}
	for i, drs := range s.DrsVersionHistory {
		if drs.Height < height {
			s.DrsVersionHistory = append(s.DrsVersionHistory[:i], s.DrsVersionHistory[i+1:]...)
		}
	}
}
