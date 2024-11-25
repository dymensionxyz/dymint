package types

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"slices"
	"sync"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/types"
)

// Sequencer is a struct that holds the sequencer's information and tendermint validator.
// It is populated from the Hub on start and is periodically updated from the Hub polling.
// Uses tendermint's validator types for compatibility.
type Sequencer struct {
	// SettlementAddress is the address of the sequencer in the settlement layer (bech32 string)
	SettlementAddress string
	// RewardAddr is the bech32-encoded sequencer's reward address
	RewardAddr string
	// WhitelistedRelayers is a list of the whitelisted relayer addresses. Addresses are bech32-encoded strings.
	WhitelistedRelayers []string

	// val is a tendermint validator type for compatibility. Holds the public key and cons address.
	val types.Validator
}

func NewSequencer(
	pubKey tmcrypto.PubKey,
	settlementAddress string,
	rewardAddr string,
	whitelistedRelayers []string,
) *Sequencer {
	if pubKey == nil {
		return nil
	}
	return &Sequencer{
		SettlementAddress:   settlementAddress,
		RewardAddr:          rewardAddr,
		WhitelistedRelayers: whitelistedRelayers,
		val:                 *types.NewValidator(pubKey, 1),
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

// AnyConsPubKey returns sequencer's consensus public key represented as Cosmos proto.Any.
func (s Sequencer) AnyConsPubKey() (*codectypes.Any, error) {
	val := s.TMValidator()
	pubKey, err := cryptocodec.FromTmPubKeyInterface(val.PubKey)
	if err != nil {
		return nil, fmt.Errorf("convert tendermint pubkey to cosmos: %w", err)
	}
	anyPK, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		return nil, fmt.Errorf("convert cosmos pubkey to any: %w", err)
	}
	return anyPK, nil
}

// MustFullHash returns a "full" hash of the sequencer that includes all fields of the Sequencer type.
func (s Sequencer) MustFullHash() []byte {
	h := sha256.New()
	h.Write([]byte(s.SettlementAddress))
	h.Write([]byte(s.RewardAddr))
	for _, r := range s.WhitelistedRelayers {
		h.Write([]byte(r))
	}
	h.Write(s.MustHash())
	return h.Sum(nil)
}

// SequencerListRightOuterJoin returns a set of sequencers that are in B but not in A.
// Sequencer is identified by a hash of all of it's fields.
//
// Example 1:
//
//	s1 =      {seq1, seq2, seq3}
//	s2 =      {      seq2, seq3, seq4}
//	s1 * s2 = {                  seq4}
func SequencerListRightOuterJoin(A, B Sequencers) Sequencers {
	lhsSet := make(map[string]struct{})
	for _, s := range A {
		lhsSet[string(s.MustFullHash())] = struct{}{}
	}
	var diff Sequencers
	for _, s := range B {
		if _, ok := lhsSet[string(s.MustFullHash())]; !ok {
			diff = append(diff, s)
		}
	}
	return diff
}

func (s Sequencer) String() string {
	return fmt.Sprintf("Sequencer{SettlementAddress: %s RewardAddr: %s WhitelistedRelayers: %v Validator: %s}", s.SettlementAddress, s.RewardAddr, s.WhitelistedRelayers, s.val.String())
}

// Sequencers is a list of sequencers.
type Sequencers []Sequencer

// SequencerSet is a set of rollapp sequencers. It holds the entire set of sequencers
// that were ever associated with the rollapp (including bonded/unbonded/unbonding).
// It is populated from the Hub on start and is periodically updated from the Hub polling.
// This type is thread-safe.
type SequencerSet struct {
	mu         sync.RWMutex
	sequencers Sequencers
}

func NewSequencerSet(s ...Sequencer) *SequencerSet {
	return &SequencerSet{
		mu:         sync.RWMutex{},
		sequencers: s,
	}
}

// Set sets the sequencers of the sequencer set.
func (s *SequencerSet) Set(sequencers Sequencers) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequencers = sequencers
}

func (s *SequencerSet) GetAll() Sequencers {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.sequencers)
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
