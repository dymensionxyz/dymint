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

type Revision struct {
	StartHeight uint64
	Revision    tmstate.Version
}

// State contains information about current state of the blockchain.
type State struct {
	Revisions []Revision
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
	rollappParams, err := GetRollappParamsFromGenesis(appState)
	if err != nil {
		return fmt.Errorf("get rollapp params from genesis: %w", err)
	}
	s.RollappParams = *rollappParams.Params
	return nil
}

func (s *State) GetRevisions() []Revision {
	return s.Revisions
}

func (s *State) GetLastRevision() uint64 {
	return s.Revisions[len(s.Revisions)-1].Revision.Consensus.App
}

func (s *State) GetRevision() uint64 {
	if len(s.Revisions) == 0 {
		return 0
	}
	return s.Revisions[len(s.Revisions)-1].Revision.Consensus.App
}

func (s *State) GetVersion() tmstate.Version {
	if len(s.Revisions) == 0 {
		return tmstate.Version{}
	}
	return s.Revisions[len(s.Revisions)-1].Revision
}

func (s *State) SetRevision(revisions []Revision) {
	s.Revisions = revisions
}

func (s *State) ValidateRevision(height uint64, version Version) error {
	if len(s.Revisions) == 0 {
		panic("no revisions in state")
	}
	rev := s.Revisions[0]
	for i := 1; i < len(s.Revisions); i++ {
		if height >= s.Revisions[i].StartHeight {
			rev = s.Revisions[i]
		} else {
			break
		}
	}
	if version.App != rev.Revision.Consensus.App || version.Block != rev.Revision.Consensus.Block {
		return ErrVersionMismatch
	}
	return nil
}
