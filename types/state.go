package types

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	// TODO(tzdybal): copy to local project?

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/dymensionxyz/dymint/types/pb/dymint"
)

const rollappparams_modulename = "rollappparams"

// State contains information about current state of the blockchain.
type State struct {
	Version             tmstate.Version
	RevisionStartHeight uint64
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
}

func (s *State) GetProposer() *Sequencer {
	return s.Proposer.Load()
}

// Warning: can be nil during 'graceful' shutdown
func (s *State) GetProposerPubKey() tmcrypto.PubKey {
	proposer := s.Proposer.Load()
	if proposer == nil {
		return nil
	}
	return proposer.PubKey()
}

func (s *State) SafeProposerPubKey() (tmcrypto.PubKey, error) {
	ret := s.GetProposerPubKey()
	if ret == nil {
		return nil, gerrc.ErrNotFound.Wrap("proposer")
	}
	return ret, nil
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
	params, ok := objmap[rollappparams_modulename]
	if !ok {
		return fmt.Errorf("module not defined in genesis: %s", rollappparams_modulename)
	}

	var rollappParams RollappParams
	err = json.Unmarshal(params, &rollappParams)
	if err != nil {
		return err
	}
	s.RollappParams = *rollappParams.Params
	return nil
}

func (s *State) GetRevision() uint64 {
	return s.Version.Consensus.App
}

func (s *State) SetRevision(revision uint64) {
	s.Version.Consensus.App = revision
}
