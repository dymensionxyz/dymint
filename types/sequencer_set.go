package types

import (
	"bytes"
	"fmt"

	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/types"
)

// sequencer is a struct that holds the sequencer's settlement address and tendermint validator
// it's populated from the SL client
// uses tendermint's validator types for compatibility
type Sequencer struct {
	// SettlementAddress is the address of the sequencer in the settlement layer (bech32 string)
	SettlementAddress string `json:"settlement_address"`
	// tendermint validator type for compatibility. holds the public key and cons address
	val types.Validator
}

func NewSequencer(pubKey tmcrypto.PubKey, settlementAddress string) *Sequencer {
	if pubKey == nil {
		return nil
	}
	return &Sequencer{
		SettlementAddress: settlementAddress,
		val:               *types.NewValidator(pubKey, 1),
	}
}

// IsEmpty returns true if the sequencer is empty
// we check if the pubkey is nil
func (s Sequencer) IsEmpty() bool {
	return s.val.PubKey == nil
}

func (s Sequencer) ConsAddress() string {
	return s.val.Address.String()
}

func (s Sequencer) PubKey() tmcrypto.PubKey {
	return s.val.PubKey
}

func (s Sequencer) TMValidator() *types.Validator {
	return &s.val
}

func (s Sequencer) TMValidators() []*types.Validator {
	return []*types.Validator{s.TMValidator()}
}

func (s Sequencer) TMValset() (*types.ValidatorSet, error) {
	return types.ValidatorSetFromExistingValidators(s.TMValidators())
}

// Hash returns tendermint compatible hash of the sequencer
func (s Sequencer) Hash() ([]byte, error) {
	vs, err := s.TMValset()
	if err != nil {
		return nil, fmt.Errorf("tm valset: %w", err)
	}
	return vs.Hash(), nil
}

// MustHash returns tendermint compatible hash of the sequencer
func (s Sequencer) MustHash() []byte {
	h, err := s.Hash()
	if err != nil {
		panic(fmt.Errorf("hash: %w", err))
	}
	return h
}

// SequencerSet is a set of rollapp sequencers and a proposer.
type SequencerSet struct {
	// Sequencers is the set of sequencers registered in the settlement layer
	// it holds the entire set of sequencers, including unbonded sequencers
	Sequencers []Sequencer `json:"sequencers"`
	// Proposer is the sequencer that is the proposer for the current sequencer set
	// can be nil if no proposer is set
	// proposer is also included in the sequencers set
	Proposer *Sequencer `json:"proposer"`
}

func (s *SequencerSet) GetProposerPubKey() tmcrypto.PubKey {
	if s.Proposer == nil {
		return nil
	}
	return s.Proposer.PubKey()
}

// ProposerHash returns the hash of the proposer
func (s *SequencerSet) ProposerHash() []byte {
	if s.Proposer == nil {
		return make([]byte, 0, 32)
	}
	return s.Proposer.MustHash()
}

// SetProposerByHash sets the proposer by hash.
// It returns an error if the hash is not found in the sequencer set
// Used when updating proposer from the L2 blocks (nextSequencerHash header field)
func (s *SequencerSet) SetProposerByHash(hash []byte) error {
	for _, seq := range s.Sequencers {
		if bytes.Equal(seq.MustHash(), hash) {
			s.SetProposer(&seq)
			return nil
		}
	}
	// can't find the proposer in the sequencer set
	// can happen in cases where the node is not synced with the SL and the sequencer array in the set is not updated
	return ErrMissingProposerPubKey
}

// SetProposer sets the proposer and adds it to the sequencer set if not already present.
func (s *SequencerSet) SetProposer(proposer *Sequencer) {
	if proposer == nil {
		s.Proposer = nil
		return
	}
	s.Proposer = proposer

	// Add proposer to bonded set if not already present
	// can happen in cases where the node is not synced with the SL and the sequencer array in the set is not updated
	if s.GetByConsAddress(proposer.val.Address) == nil {
		s.Sequencers = append(s.Sequencers, *proposer)
	}
}

// SetSequencers sets the sequencers of the sequencer set.
func (s *SequencerSet) SetSequencers(sequencers []Sequencer) {
	s.Sequencers = sequencers
}

// GetByAddress returns the sequencer with the given settlement address.
// used when handling events from the settlement, where the settlement address is used
func (s *SequencerSet) GetByAddress(settlement_address string) *Sequencer {
	for _, seq := range s.Sequencers {
		if seq.SettlementAddress == settlement_address {
			return &seq
		}
	}
	return nil
}

// GetByConsAddress returns the sequencer with the given consensus address.
func (s *SequencerSet) GetByConsAddress(cons_addr []byte) *Sequencer {
	for _, seq := range s.Sequencers {
		if bytes.Equal(seq.val.Address, cons_addr) {
			return &seq
		}
	}
	return nil
}

func (s *SequencerSet) String() string {
	return fmt.Sprintf("SequencerSet: %v", s.Sequencers)
}

/* -------------------------- backward compatibility ------------------------- */
// old dymint version used tendermint.ValidatorSet for sequencers
// these methods are used for backward compatibility
func NewSequencerFromValidator(val types.Validator) *Sequencer {
	return &Sequencer{
		SettlementAddress: "",
		val:               val,
	}
}

// LoadFromValSet sets the sequencers from a tendermint validator set.
// used for backward compatibility. should be used only for queries (used by rpc/client)
func (s *SequencerSet) LoadFromValSet(valSet *types.ValidatorSet) {
	if valSet == nil {
		return
	}

	sequencers := make([]Sequencer, len(valSet.Validators))
	for i, val := range valSet.Validators {
		sequencers[i] = *NewSequencerFromValidator(*val)
	}
	s.SetSequencers(sequencers)
	s.SetProposer(NewSequencerFromValidator(*valSet.Proposer))
}
