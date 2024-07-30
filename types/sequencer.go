package types

import (
	"bytes"
	"fmt"

	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/types"
)

// trying to remain compatible with tendermint/types/validator_set.go
type SequencerSet struct {
	// BondedSet    *types.ValidatorSet
	Validators   []*types.Validator `json:"validators"`
	Proposer     *types.Validator   `json:"proposer"`
	ProposerHash []byte
}

// get pubkey of proposer
func (s *SequencerSet) GetProposerPubKey() tmcrypto.PubKey {
	if s.Proposer == nil {
		return nil
	}
	return s.Proposer.PubKey
}

// set proposer by hash
func (s *SequencerSet) SetProposerByHash(hash []byte) error {
	for _, val := range s.Validators {
		if bytes.Equal(GetHash(val), hash) {
			s.SetProposer(val)
			return nil
		}
	}
	return fmt.Errorf("next sequencer not found in bonded set")
}

// setProposer sets the proposer set and hash.
func (s *SequencerSet) SetProposer(proposer *types.Validator) {
	s.Proposer = proposer
	s.ProposerHash = GetHash(proposer)

	// FIXME: add to the s.BondedSet.Validators if not already there
}

// set the bonded set
func (s *SequencerSet) SetBondedSet(bondedSet *types.ValidatorSet) {
	s.SetBondedValidators(bondedSet.Copy().Validators)
	s.Proposer = bondedSet.Proposer
	s.ProposerHash = GetHash(s.Proposer)
}

func (s *SequencerSet) SetBondedValidators(bondedSet []*types.Validator) {
	s.Validators = bondedSet
}

// GetByAddress
func (s *SequencerSet) GetByAddress(addr []byte) *types.Validator {
	for _, val := range s.Validators {
		if bytes.Equal(val.Address, addr) {
			return val
		}
	}
	return nil
}

// HasAddress returns true if address given is in the validator set, false -
// otherwise.
func (s *SequencerSet) HasAddress(address []byte) bool {
	return s.GetByAddress(address) != nil
}

// ToValSet
func (s *SequencerSet) ToValSet() *types.ValidatorSet {
	set := types.NewValidatorSet(s.Validators)
	set.Proposer = s.Proposer
	return set.CopyIncrementProposerPriority(1)
}

func (s *SequencerSet) String() string {
	return fmt.Sprintf("SequencerSet{Validators: %v, Proposer: %v, ProposerHash: %v}", s.Validators, s.Proposer, s.ProposerHash)
}

func GetHash(seq *types.Validator) []byte {
	if seq == nil {
		return []byte{}
	}
	// we take a set with only the proposer and hash it
	tempProposerSet := types.NewValidatorSet([]*types.Validator{seq})
	return tempProposerSet.Hash()
}

// Makes a copy of the validator list.
func (s *SequencerSet) Copy() *SequencerSet {
	// copy the validators
	valsCopy := make([]*types.Validator, len(s.Validators))
	for i, val := range s.Validators {
		valsCopy[i] = val.Copy()
	}

	return &SequencerSet{
		Validators:   valsCopy,
		Proposer:     s.Proposer,
		ProposerHash: s.ProposerHash,
	}
}
