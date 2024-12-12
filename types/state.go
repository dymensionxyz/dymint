package types

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	

	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/dymensionxyz/dymint/types/pb/dymint"
)

const rollappparams_modulename = "rollappparams"


type State struct {
	Version             tmstate.Version
	RevisionStartHeight uint64
	
	ChainID       string
	InitialHeight uint64 

	
	LastBlockHeight atomic.Uint64

	
	Proposer atomic.Pointer[Sequencer]

	
	
	ConsensusParams tmproto.ConsensusParams

	
	LastResultsHash [32]byte

	
	AppHash [32]byte

	
	RollappParams dymint.RollappParams

	
	LastHeaderHash [32]byte
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


func (s *State) GetProposerHash() []byte {
	proposer := s.Proposer.Load()
	if proposer == nil {
		return nil
	}
	return proposer.MustHash()
}


func (s *State) SetProposer(proposer *Sequencer) {
	s.Proposer.Store(proposer)
}

func (s *State) IsGenesis() bool {
	return s.Height() == 0
}

type RollappParams struct {
	Params *dymint.RollappParams
}



func (s *State) SetHeight(height uint64) {
	s.LastBlockHeight.Store(height)
}


func (s *State) Height() uint64 {
	return s.LastBlockHeight.Load()
}


func (s *State) NextHeight() uint64 {
	if s.IsGenesis() {
		return s.InitialHeight
	}
	return s.Height() + 1
}


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
