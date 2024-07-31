package types

import (
	"bytes"
	"fmt"

	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	cmtproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/types"
)

// similar to tendermint/types/validator_set.go
// but doesn't enforces proposer to be set, and keeps proposer hash
type SequencerSet struct {
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

// ToProto
func (s *SequencerSet) ToProto() (*pb.SequencerSet, error) {
	protoSet := new(pb.SequencerSet)
	seqsProto := make([]*cmtproto.Validator, len(s.Validators))
	for i := 0; i < len(s.Validators); i++ {
		valp, err := s.Validators[i].ToProto()
		if err != nil {
			return nil, fmt.Errorf("toProto: validatorSet error: %w", err)
		}
		seqsProto[i] = valp
	}
	protoSet.Sequencers = seqsProto

	if s.Proposer != nil {
		valProposer, err := s.Proposer.ToProto()
		if err != nil {
			return nil, fmt.Errorf("toProto: validatorSet proposer error: %w", err)
		}
		protoSet.Proposer = valProposer
	}

	return protoSet, nil
}

func NewSequencerSetFromProto(protoSet pb.SequencerSet) (*SequencerSet, error) {
	seqs := make([]*types.Validator, len(protoSet.Sequencers))
	for i, valProto := range protoSet.Sequencers {
		val, err := types.ValidatorFromProto(valProto)
		if err != nil {
			return nil, fmt.Errorf("fromProto: validatorSet error: %w", err)
		}
		seqs[i] = val
	}

	var proposer *types.Validator
	if protoSet.Proposer != nil {
		valProposer, err := types.ValidatorFromProto(protoSet.Proposer)
		if err != nil {
			return nil, fmt.Errorf("fromProto: validatorSet proposer error: %w", err)
		}
		proposer = valProposer
	}

	return &SequencerSet{
		Validators:   seqs,
		Proposer:     proposer,
		ProposerHash: GetHash(proposer),
	}, nil
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
