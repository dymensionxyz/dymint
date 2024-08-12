package types

import (
	"bytes"
	"fmt"

	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	cmtproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/types"
)

// SequencerSet is a set of rollapp sequencers and a proposer.
// we use tendermint's Validator struct to represent sequencers and proposer
type SequencerSet struct {
	Sequencers   []*types.Validator `json:"sequencers"`
	Proposer     *types.Validator   `json:"proposer"`
	ProposerHash []byte
}

func (s *SequencerSet) GetProposerPubKey() tmcrypto.PubKey {
	if s.Proposer == nil {
		return nil
	}
	return s.Proposer.PubKey
}

// SetProposerByHash sets the proposer by hash.
// It returns an error if the hash is not found in the sequencer set.
// Used when updating proposer from the L2 blocks
func (s *SequencerSet) SetProposerByHash(hash []byte) error {
	for _, val := range s.Sequencers {
		if bytes.Equal(GetHash(val), hash) {
			s.SetProposer(val)
			return nil
		}
	}
	return ErrMissingProposerPubKey
}

// SetProposer sets the proposer and adds it to the sequencer set.
func (s *SequencerSet) SetProposer(proposer *types.Validator) {
	s.Proposer = proposer
	s.ProposerHash = GetHash(proposer)

	// Add proposer to bonded set if not already present
	if !s.HasAddress(proposer.Address) {
		s.Sequencers = append(s.Sequencers, proposer)
	}
}

func (s *SequencerSet) SetSequencers(sequencers []*types.Validator) {
	s.Sequencers = sequencers
}

func (s *SequencerSet) GetByAddress(addr []byte) *types.Validator {
	for _, val := range s.Sequencers {
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

// ToValSet returns the sequencer set as a tendermint ValidatorSet.
// used for PRC client compatibility
func (s *SequencerSet) ToValSet() *types.ValidatorSet {
	set := types.NewValidatorSet(s.Sequencers)
	set.Proposer = s.Proposer
	return set.CopyIncrementProposerPriority(1) // needed to set proposer on the valSet
}

func (s *SequencerSet) ToProto() (*pb.SequencerSet, error) {
	protoSet := new(pb.SequencerSet)
	seqsProto := make([]*cmtproto.Validator, len(s.Sequencers))
	for i := 0; i < len(s.Sequencers); i++ {
		valp, err := s.Sequencers[i].ToProto()
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

func (s *SequencerSet) FromProto(protoSet pb.SequencerSet) error {
	seqs := make([]*types.Validator, len(protoSet.Sequencers))
	for i, valProto := range protoSet.Sequencers {
		val, err := types.ValidatorFromProto(valProto)
		if err != nil {
			return fmt.Errorf("fromProto: validatorSet error: %w", err)
		}
		seqs[i] = val
	}

	var proposer *types.Validator
	if protoSet.Proposer != nil {
		valProposer, err := types.ValidatorFromProto(protoSet.Proposer)
		if err != nil {
			return fmt.Errorf("fromProto: validatorSet proposer error: %w", err)
		}
		proposer = valProposer
	}

	s.Sequencers = seqs
	s.Proposer = proposer
	s.ProposerHash = GetHash(proposer)

	return nil
}

func (s *SequencerSet) String() string {
	return fmt.Sprintf("SequencerSet: %v", s.Sequencers)
}

func GetHash(seq *types.Validator) []byte {
	if seq == nil {
		return []byte{}
	}
	// we take a set with only the proposer and hash it
	tempProposerSet := types.NewValidatorSet([]*types.Validator{seq})
	return tempProposerSet.Hash()
}

// NewSequencerSet creates a new empty sequencer set.
// used for testing
func NewSequencerSet() *SequencerSet {
	return &SequencerSet{
		Sequencers:   types.NewValidatorSet(nil).Validators,
		Proposer:     types.NewValidatorSet(nil).Proposer,
		ProposerHash: make([]byte, 32), // Initialize as a slice of 32 bytes
	}
}
