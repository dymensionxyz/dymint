package types

import (
	"bytes"
	"fmt"

	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/types"
)

// trying to remain compatible with tendermint/types/validator_set.go
type SequencerSet struct {
	BondedSet    *types.ValidatorSet
	ProposerHash []byte
}

// get pubkey of proposer
func (s *SequencerSet) GetProposerPubKey() tmcrypto.PubKey {
	return s.BondedSet.Proposer.PubKey
}

// set proposer by hash
func (s *SequencerSet) SetProposerByHash(hash []byte) error {
	for _, val := range s.BondedSet.Validators {
		if bytes.Equal(GetHash(val), hash) {
			s.SetProposer(val)
			return nil
		}
	}
	// return types.ErrValidatorNotFound
	return fmt.Errorf("next sequencer not found in bonded set")
}

// setProposer sets the proposer set and hash.
func (s *SequencerSet) SetProposer(proposer *types.Validator) {
	// TODO: add to the s.BondedSet.Validators if not already there

	s.BondedSet.Proposer = proposer
	// we take a set with only the proposer and hash it
	tempProposerSet := types.NewValidatorSet([]*types.Validator{proposer})
	s.ProposerHash = tempProposerSet.Hash()
}

// set the bonded set
func (s *SequencerSet) SetBondedSet(bondedSet *types.ValidatorSet) {
	s.BondedSet = bondedSet.Copy()
	s.ProposerHash = GetHash(s.BondedSet.Proposer)
}

func GetHash(seq *types.Validator) []byte {
	// we take a set with only the proposer and hash it
	tempProposerSet := types.NewValidatorSet([]*types.Validator{seq})
	return tempProposerSet.Hash()
}
