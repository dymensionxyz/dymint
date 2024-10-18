package types

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/types"
)

// Sequencer is a struct that holds the sequencer's settlement address and tendermint validator.
// It's populated from the SL client. Uses tendermint's validator types for compatibility.
type Sequencer struct {
	// SettlementAddress is the address of the sequencer in the settlement layer (bech32 string).
	SettlementAddress string `json:"settlement_address"`
	// RewardAddr is the bech32-encoded sequencer's reward address.
	RewardAddr string `protobuf:"bytes,3,opt,name=reward_addr,json=rewardAddr,proto3" json:"reward_addr,omitempty"`
	// WhitelistedRelayers is a list of the whitelisted relayer addresses. Addresses are bech32-encoded strings.
	WhitelistedRelayers []string `protobuf:"bytes,4,rep,name=relayers,proto3" json:"relayers,omitempty"`

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

func (s Sequencer) TMValidator() (*types.Validator, error) {
	return &s.val, nil
}

func (s Sequencer) Equal(rhs Sequencer) bool {
	return bytes.Equal(s.FullHash(), rhs.FullHash())
}

func (s Sequencer) ConsAddress() string {
	return s.val.Address.String()
}

func (s Sequencer) PubKey() tmcrypto.PubKey {
	return s.val.PubKey
}

// AnyConsPubKey returns sequencer's consensus public key represented as Cosmos proto.Any.
func (s Sequencer) AnyConsPubKey() (*codectypes.Any, error) {
	val, err := s.TMValidator()
	if err != nil {
		return nil, fmt.Errorf("convert next squencer to tendermint validator: %w", err)
	}
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

// FullHash returns a "full" hash of the sequencer that includes all fields of the Sequencer type.
func (s Sequencer) FullHash() []byte {
	h := sha256.New()
	h.Write([]byte(s.SettlementAddress))
	h.Write([]byte(s.RewardAddr))
	for _, r := range s.WhitelistedRelayers {
		h.Write([]byte(r))
	}
	h.Write(s.Hash())
	return h.Sum(nil)
}

// Hash returns tendermint compatible hash of the sequencer
func (s Sequencer) Hash() []byte {
	tempProposerSet := types.NewValidatorSet([]*types.Validator{&s.val})
	return tempProposerSet.Hash()
}

func (s Sequencer) String() string {
	return fmt.Sprintf("Sequencer{SettlementAddress: %s Validator: %s}", s.SettlementAddress, s.val.String())
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
	return s.Proposer.Hash()
}

// SetProposerByHash sets the proposer by hash.
// It returns an error if the hash is not found in the sequencer set
// Used when updating proposer from the L2 blocks (nextSequencerHash header field)
func (s *SequencerSet) SetProposerByHash(hash []byte) error {
	for _, seq := range s.Sequencers {
		if bytes.Equal(seq.Hash(), hash) {
			s.SetProposer(&seq)
			return nil
		}
	}
	// can't find the proposer in the sequencer set
	// can happen in cases where the node is not synced with the SL and the sequencer array in the set is not updated
	return ErrMissingProposerPubKey
}

// GetByHash gets the sequencer by hash. It returns an error if the hash is not found in the sequencer set.
func (s *SequencerSet) GetByHash(hash []byte) (Sequencer, error) {
	for _, seq := range s.Sequencers {
		if bytes.Equal(seq.Hash(), hash) {
			return seq, nil
		}
	}
	// can't find the proposer in the sequencer set
	// can happen in cases where the node is not synced with the SL and the sequencer array in the set is not updated
	return Sequencer{}, ErrMissingProposerPubKey
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

// SequencerListRightOuterJoin returns a set of sequencers that are in B but not in A.
// CONTRACT: both A and B do not have duplicates!
//
// Example 1:
//
//	s1 =      {seq1, seq2, seq3}
//	s2 =      {      seq2, seq3, seq4}
//	s1 * s2 = {                  seq4}
func SequencerListRightOuterJoin(A, B []Sequencer) []Sequencer {
	lhsSet := make(map[string]struct{})
	for _, s := range A {
		lhsSet[string(s.FullHash())] = struct{}{}
	}
	var diff []Sequencer
	for _, s := range B {
		if _, ok := lhsSet[string(s.FullHash())]; !ok {
			diff = append(diff, s)
		}
	}
	return diff
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
