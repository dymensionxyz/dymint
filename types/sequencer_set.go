package types

import (
	"bytes"
	"fmt"
	"sync"

	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/types"
)

// Sequencer is a struct that holds the sequencer's settlement address and tendermint validator.
// It's populated from the SL client. Uses tendermint's validator types for compatibility.
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

// SequencerSet is a set of rollapp sequencers. It holds the entire set of sequencers
// that were ever associated with the rollapp (including bonded/unbonded/unbonding).
// It is populated from the Hub on start and is periodically updated from the Hub polling.
// This type is thread-safe.
type SequencerSet struct {
	mu         sync.RWMutex
	sequencers []Sequencer
}

func NewSequencerSet(s ...Sequencer) *SequencerSet {
	return &SequencerSet{
		mu:         sync.RWMutex{},
		sequencers: s,
	}
}

// SetSequencers sets the sequencers of the sequencer set.
func (s *SequencerSet) SetSequencers(sequencers []Sequencer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequencers = sequencers
}

// AppendSequencer appends a new sequencer the sequencer set.
func (s *SequencerSet) AppendSequencer(sequencer Sequencer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequencers = append(s.sequencers, sequencer)
}

func (s *SequencerSet) GetSequencers() []Sequencer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sequencers
}

// GetByHash gets the sequencer by hash. It returns an error if the hash is not found in the sequencer set.
func (s *SequencerSet) GetByHash(hash []byte) (Sequencer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, seq := range s.sequencers {
		if bytes.Equal(seq.MustHash(), hash) {
			return seq, true
		}
	}
	return Sequencer{}, false
}

// GetByAddress returns the sequencer with the given settlement address.
// used when handling events from the settlement, where the settlement address is used
func (s *SequencerSet) GetByAddress(settlementAddress string) (Sequencer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, seq := range s.sequencers {
		if seq.SettlementAddress == settlementAddress {
			return seq, true
		}
	}
	return Sequencer{}, false
}

// GetByConsAddress returns the sequencer with the given consensus address.
func (s *SequencerSet) GetByConsAddress(consAddr []byte) (Sequencer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, seq := range s.sequencers {
		if bytes.Equal(seq.val.Address, consAddr) {
			return seq, true
		}
	}
	return Sequencer{}, false
}

func (s *SequencerSet) String() string {
	return fmt.Sprintf("SequencerSet: %v", s.sequencers)
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
