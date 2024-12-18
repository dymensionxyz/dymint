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

type Sequencer struct {
	SettlementAddress string

	RewardAddr string

	WhitelistedRelayers []string

	val types.Validator
}

func NewSequencer(
	pubKey tmcrypto.PubKey,
	settlementAddress string,
	rewardAddr string,
	whitelistedRelayers []string,
) *Sequencer {
	if pubKey == nil { // when would this happen? should return error
		return nil
	}
	return &Sequencer{
		SettlementAddress:   settlementAddress,
		RewardAddr:          rewardAddr,
		WhitelistedRelayers: whitelistedRelayers,
		val:                 *types.NewValidator(pubKey, 1),
	}
}

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

func (s Sequencer) Hash() ([]byte, error) {
	vs, err := s.TMValset()
	if err != nil {
		return nil, fmt.Errorf("tm valset: %w", err)
	}
	return vs.Hash(), nil
}

func (s Sequencer) MustHash() []byte {
	h, err := s.Hash()
	if err != nil {
		panic(fmt.Errorf("hash: %w", err))
	}
	return h
}

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

// would make more sense as a .Sub oper
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

type Sequencers []Sequencer

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

func (s *SequencerSet) Set(sequencers Sequencers) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequencers = sequencers
}

func (s *SequencerSet) GetAll() Sequencers {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.sequencers) // whitelisted relayers and pubkey are not cloned
}

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

func NewSequencerFromValidator(val types.Validator) *Sequencer {
	return &Sequencer{
		SettlementAddress: "", //?
		val:               val,
	}
}
