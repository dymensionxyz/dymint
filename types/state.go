package types

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	// TODO(tzdybal): copy to local project?

	"github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/dymensionxyz/dymint/version"
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
	ConsensusParams tmproto.ConsensusParams

	// Merkle root of the results from executing prev block
	LastResultsHash [32]byte

	// the latest AppHash we've received from calling abci.Commit()
	AppHash [32]byte

	// New rollapp parameters .
	RollappParams dymint.RollappParams

	// LastHeaderHash is the hash of the last block header.
	LastHeaderHash [32]byte

	// The last DRS versions including height upgrade
	DrsVersionHistory []*dymint.DRSVersion

	drsMux sync.Mutex
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
	return nil
}

// GetDRSVersion returns the DRS version stored in rollapp params updates for a specific height.
// It only works for non-finalized heights.
// If drs history is empty (because there is no version update for non-finalized heights) it will return current version.
func (s *State) GetDRSVersion(height uint64) (string, error) {
	defer s.drsMux.Unlock()
	s.drsMux.Lock()

	if len(s.DrsVersionHistory) == 0 {
		return version.Commit, nil
	}
	drsVersion := ""
	for _, drs := range s.DrsVersionHistory {
		if height >= drs.Height {
			drsVersion = drs.Version
		} else {
			break
		}
	}
	if drsVersion == "" {
		return drsVersion, gerrc.ErrNotFound
	}
	return drsVersion, nil
}

// AddDRSVersion adds a new record for the DRS version update heights.
func (s *State) AddDRSVersion(height uint64, version string) {
	defer s.drsMux.Unlock()
	s.drsMux.Lock()
	s.DrsVersionHistory = append(s.DrsVersionHistory, &dymint.DRSVersion{Height: height, Version: version})
}

// ClearDrsVersionHeights clears drs version previous to the specified height,
// but keeping always the last drs version record.
// sequencers clear anything previous to the last submitted height
// and full-nodes clear up to last finalized height
func (s *State) ClearDRSVersionHeights(height uint64) {

	if len(s.DrsVersionHistory) == 1 {
		return
	}

	for i, drs := range s.DrsVersionHistory {
		if drs.Height < height {
			s.drsMux.Lock()
			s.DrsVersionHistory = append(s.DrsVersionHistory[:i], s.DrsVersionHistory[i+1:]...)
			s.drsMux.Unlock()
		}
	}
}
