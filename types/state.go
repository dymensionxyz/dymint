package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"

	// TODO(tzdybal): copy to local project?

	"github.com/dymensionxyz/dymint/types/pb/dymint"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
)

type Param struct {
	Params json.RawMessage `json:"params"`
}

type ConsensusParams struct {
	// maximum amount of gas that all transactions included in a block can use
	Da           string
	Version      string
	Blockmaxgas  string
	Blockmaxsize uint32
}

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
	ConsensusParams                  dymint.RollappConsensusParams
	LastHeightConsensusParamsChanged int64

	// Merkle root of the results from executing prev block
	LastResultsHash [32]byte

	// the latest AppHash we've received from calling abci.Commit()
	AppHash [32]byte
}

func (s *State) IsGenesis() bool {
	return s.Height() == 0
}

type RollappParams struct {
	Params *dymint.RollappConsensusParams
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

// SetConsensusParamsFromGenesis sets the rollapp consensus params from genesis
func (s *State) SetConsensusParamsFromGenesis(appState json.RawMessage) error {
	var objmap map[string]json.RawMessage
	err := json.Unmarshal(appState, &objmap)
	if err != nil {
		return err
	}
	params, ok := objmap["rollappparams"]
	if !ok {
		return fmt.Errorf("rollappparams not defined in genesis")
	}

	var param Param
	err = json.Unmarshal(params, &param)
	if err != nil {
		return err
	}

	var consensusParams ConsensusParams
	err = json.Unmarshal(param.Params, &consensusParams)
	if err != nil {
		return err
	}

	s.ConsensusParams.Blockmaxgas, err = strconv.ParseInt(consensusParams.Blockmaxgas, 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse block max gas from genesis")
	}
	s.ConsensusParams.Blockmaxsize = int64(consensusParams.Blockmaxsize)
	s.ConsensusParams.Version = consensusParams.Version
	s.ConsensusParams.Da = consensusParams.Da

	return nil
}
