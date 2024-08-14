package types

import (
	"bytes"
	"fmt"

	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/types"
)

// sequencer is a struct that holds the sequencer's settlement address and tendermint validator
type Sequencer struct {
	// SettlementAddress is the address of the sequencer in the settlement layer (bech32 string)
	SettlementAddress string `json:"settlement_address"`
	// tendermint validator type for compatibility. holds the public key and cons address
	types.Validator // FIXME: try to have it as unexported
}

func NewSequencer(pubKey tmcrypto.PubKey, settlementAddress string) *Sequencer {
	if pubKey == nil {
		return nil
	}
	return &Sequencer{
		SettlementAddress: settlementAddress,
		Validator:         *types.NewValidator(pubKey, 1),
	}
}

// used only for backward compatibility, as previously sequencers were created directly as tendermint validators
func NewSequencerFromValidator(val types.Validator) *Sequencer {
	return &Sequencer{
		SettlementAddress: "",
		Validator:         val,
	}
}

// IsEmpty returns true if the sequencer is empty
func (s Sequencer) IsEmpty() bool {
	return s.Address == nil
}

func (s Sequencer) TMValidator() (*types.Validator, error) {
	return &s.Validator, nil
}

func (s Sequencer) ConsAddress() string {
	return s.Address.String()
}

// SequencerSet is a set of rollapp sequencers and a proposer.
type SequencerSet struct {
	Sequencers   []Sequencer `json:"sequencers"`
	Proposer     *Sequencer  `json:"proposer"`
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
	for _, seq := range s.Sequencers {
		if bytes.Equal(GetHash(&seq), hash) {
			s.SetProposer(&seq)
			return nil
		}
	}
	return ErrMissingProposerPubKey
}

// SetProposer sets the proposer and adds it to the sequencer set.
func (s *SequencerSet) SetProposer(proposer *Sequencer) {
	s.Proposer = proposer
	s.ProposerHash = GetHash(proposer)

	// Add proposer to bonded set if not already present
	if proposer != nil && !s.HasConsAddress(proposer.Address) {
		s.Sequencers = append(s.Sequencers, *proposer)
	}
}

func (s *SequencerSet) SetSequencers(sequencers []Sequencer) {
	s.Sequencers = sequencers
}

// GetByAddress returns the sequencer with the given address.
// The address type is tendermint's Address type. not to be confused with bech32 address
func (s *SequencerSet) GetByAddress(settlement_address string) *Sequencer {
	for _, seq := range s.Sequencers {
		if seq.SettlementAddress == settlement_address {
			return &seq
		}
	}
	return nil
}

func (s *SequencerSet) GetByConsAddress(cons_addr []byte) *Sequencer {
	for _, seq := range s.Sequencers {
		if bytes.Equal(seq.Address, cons_addr) {
			return &seq
		}
	}
	return nil
}

// HasConsAddress returns true if address given is in the validator set, false -
// otherwise.
func (s *SequencerSet) HasConsAddress(cons_addr []byte) bool {
	return s.GetByConsAddress(cons_addr) != nil
}

func (s *SequencerSet) ToProto() (*pb.SequencerSet, error) {
	protoSet := new(pb.SequencerSet)

	seqsProto := make([]*pb.Sequencer, len(s.Sequencers))
	for i := 0; i < len(s.Sequencers); i++ {
		valp, err := s.Sequencers[i].ToProto()
		if err != nil {
			return nil, fmt.Errorf("toProto: validatorSet error: %w", err)
		}
		seq := new(pb.Sequencer)
		seq.SettlementAddress = s.Sequencers[i].SettlementAddress
		seq.Validator = valp
		seqsProto[i] = seq
	}
	protoSet.Sequencers = seqsProto

	if s.Proposer != nil {
		valProposer, err := s.Proposer.ToProto()
		if err != nil {
			return nil, fmt.Errorf("toProto: validatorSet proposer error: %w", err)
		}
		seq := new(pb.Sequencer)
		seq.Validator = valProposer
		seq.SettlementAddress = s.Proposer.SettlementAddress
		protoSet.Proposer = seq
	}

	return protoSet, nil
}

func (s *SequencerSet) FromProto(protoSet pb.SequencerSet) error {
	seqs := make([]Sequencer, len(protoSet.Sequencers))
	for i, seqProto := range protoSet.Sequencers {
		val, err := types.ValidatorFromProto(seqProto.Validator)
		if err != nil {
			return fmt.Errorf("fromProto: validatorSet error: %w", err)
		}
		seqs[i].Validator = *val
		seqs[i].SettlementAddress = seqProto.SettlementAddress
	}

	var proposer *Sequencer
	if protoSet.Proposer != nil {
		valProposer, err := types.ValidatorFromProto(protoSet.Proposer.Validator)
		if err != nil {
			return fmt.Errorf("fromProto: validatorSet proposer error: %w", err)
		}
		proposer = new(Sequencer)
		proposer.Validator = *valProposer
		proposer.SettlementAddress = protoSet.Proposer.SettlementAddress
	}

	s.Sequencers = seqs
	s.Proposer = proposer
	s.ProposerHash = GetHash(proposer)

	return nil
}

func (s *SequencerSet) String() string {
	return fmt.Sprintf("SequencerSet: %v", s.Sequencers)
}

// FIXME: check as this doesn't have the settlement address
func (s *SequencerSet) SetSequencersFromValSet(valSet *types.ValidatorSet) {
	if valSet == nil {
		*s = *NewSequencerSet()
		return
	}

	sequencers := make([]Sequencer, len(valSet.Validators))
	for i, val := range valSet.Validators {
		sequencers[i] = *NewSequencerFromValidator(*val)
	}
	s.SetSequencers(sequencers)
	s.SetProposer(NewSequencerFromValidator(*valSet.Proposer))
}

// FIXME: change to be sequencer method
func GetHash(seq *Sequencer) []byte {
	if seq == nil {
		return []byte{}
	}
	// we take a set with only the proposer and hash it
	tempProposerSet := types.NewValidatorSet([]*types.Validator{&seq.Validator})
	return tempProposerSet.Hash()
}

// NewSequencerSet creates a new empty sequencer set.
// used for testing
func NewSequencerSet() *SequencerSet {
	return &SequencerSet{
		Sequencers:   nil,
		Proposer:     nil,
		ProposerHash: make([]byte, 32), // Initialize as a slice of 32 bytes
	}
}
