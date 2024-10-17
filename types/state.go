package types

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	// TODO(tzdybal): copy to local project?

	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/dymensionxyz/dymint/types/pb/dymint"
)

// State contains information about current state of the blockchain.
type State struct {
	Version tmstate.Version

	// immutable
	ChainID       string
	InitialHeight uint64 // should be 1, not 0, when starting from height 1

	// LastBlockHeight=0 at genesis (ie. block(H=0) does not exist)
	LastBlockHeight atomic.Uint64

	// Proposer is a sequencer that acts as a proposer. Can be nil if no proposer is set.
	Proposer atomic.Pointer[Sequencer]

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

	// Last block time
	LastBlockTime atomic.Uint64 // time in tai64 format
	// Last submitted block time
	LastSubmittedBlockTime atomic.Uint64 // time in tai64 format
}

func (s *State) GetProposer() *Sequencer {
	return s.Proposer.Load()
}

func (s *State) GetProposerPubKey() tmcrypto.PubKey {
	proposer := s.Proposer.Load()
	if proposer == nil {
		return nil
	}
	return proposer.PubKey()
}

// GetProposerHash returns the hash of the proposer
func (s *State) GetProposerHash() []byte {
	proposer := s.Proposer.Load()
	if proposer == nil {
		return nil
	}
	return proposer.MustHash()
}

// SetProposer sets the proposer. It may set the proposer to nil.
func (s *State) SetProposer(proposer *Sequencer) {
	s.Proposer.Store(proposer)
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

// SetLastBlockTime saves the last block produced timestamp
func (s *State) SetLastBlockTime(time time.Time) {
	if time.After(s.GetLastBlockTime()) {
		s.LastBlockTime.Store(uint64(time.UTC().UnixNano()))
	}
}

// GetLastBlockTime returns the last block produced timestamp
func (s *State) GetLastBlockTime() time.Time {
	return time.Unix(0, int64(s.LastBlockTime.Load()))
}

// SetLastSubmittedBlockTime saves the last block submitted to SL timestamp
func (s *State) SetLastSubmittedBlockTime(time time.Time) {
	if time.After(s.GetLastSubmittedBlockTime()) {
		s.LastSubmittedBlockTime.Store(uint64(time.UTC().UnixNano()))
	}
}

// GetLastSubmittedBlockTime returns the last block submitted to SL timestamp
func (s *State) GetLastSubmittedBlockTime() time.Time {
	return time.Unix(0, int64(s.LastSubmittedBlockTime.Load()))
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

func (s *State) GetSkewTime() time.Duration {
	if s.GetLastBlockTime().Before(s.GetLastSubmittedBlockTime()) {
		return 0
	}
	return s.GetLastBlockTime().Sub(s.GetLastSubmittedBlockTime())
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
