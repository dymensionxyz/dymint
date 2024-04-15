package store

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"

	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	"go.uber.org/multierr"

	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
)

var (
	blockPrefix      = [1]byte{1}
	indexPrefix      = [1]byte{2}
	commitPrefix     = [1]byte{3}
	statePrefix      = [1]byte{4}
	responsesPrefix  = [1]byte{5}
	validatorsPrefix = [1]byte{6}
)

// DefaultStore is a default store implmementation.
type DefaultStore struct {
	db KVStore

	height     uint64
	baseHeight uint64
}

var _ Store = &DefaultStore{}

// New returns new, default store.
func New(kv KVStore) Store {
	return &DefaultStore{
		db: kv,
	}
}

// NewBatch creates a new db batch.
func (s *DefaultStore) NewBatch() Batch {
	return s.db.NewBatch()
}

// SetHeight sets the height saved in the Store if it is higher than the existing height
func (s *DefaultStore) SetHeight(height uint64) {
	storeHeight := atomic.LoadUint64(&s.height)
	if height > storeHeight {
		_ = atomic.CompareAndSwapUint64(&s.height, storeHeight, height)
	}
}

// Height returns height of the highest block saved in the Store.
func (s *DefaultStore) Height() uint64 {
	return atomic.LoadUint64(&s.height)
}

// SetBase sets the height saved in the Store of the earliest block
func (s *DefaultStore) SetBase(height uint64) {
	baseHeight := atomic.LoadUint64(&s.baseHeight)
	if height > baseHeight {
		_ = atomic.CompareAndSwapUint64(&s.baseHeight, baseHeight, height)
	}
}

// Base returns height of the earliest block saved in the Store.
func (s *DefaultStore) Base() uint64 {
	return atomic.LoadUint64(&s.baseHeight)
}

// SaveBlock adds block to the store along with corresponding commit.
// Stored height is updated if block height is greater than stored value.
// In case a batch is provided, the block and commit are added to the batch and not saved.
func (s *DefaultStore) SaveBlock(block *types.Block, commit *types.Commit, batch Batch) (Batch, error) {
	hash := block.Header.Hash()
	blockBlob, err := block.MarshalBinary()
	if err != nil {
		return batch, fmt.Errorf("failed to marshal Block to binary: %w", err)
	}

	commitBlob, err := commit.MarshalBinary()
	if err != nil {
		return batch, fmt.Errorf("failed to marshal Commit to binary: %w", err)
	}

	// Not sure it's neeeded, as it's not used anywhere
	if batch != nil {
		err = multierr.Append(err, batch.Set(getBlockKey(hash), blockBlob))
		err = multierr.Append(err, batch.Set(getCommitKey(hash), commitBlob))
		err = multierr.Append(err, batch.Set(getIndexKey(block.Header.Height), hash[:]))
		return batch, err
	}

	bb := s.db.NewBatch()
	defer bb.Discard()

	err = multierr.Append(err, bb.Set(getBlockKey(hash), blockBlob))
	err = multierr.Append(err, bb.Set(getCommitKey(hash), commitBlob))
	err = multierr.Append(err, bb.Set(getIndexKey(block.Header.Height), hash[:]))
	if err != nil {
		return batch, fmt.Errorf("failed to create db batch: %w", err)
	}

	if err = bb.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit db batch: %w", err)
	}

	return bb, nil
}

// LoadBlock returns block at given height, or error if it's not found in Store.
// TODO(tzdybal): what is more common access pattern? by height or by hash?
// currently, we're indexing height->hash, and store blocks by hash, but we might as well store by height
// and index hash->height
func (s *DefaultStore) LoadBlock(height uint64) (*types.Block, error) {
	h, err := s.loadHashFromIndex(height)
	if err != nil {
		return nil, fmt.Errorf("failed to load hash from index: %w", err)
	}
	return s.LoadBlockByHash(h)
}

// LoadBlockByHash returns block with given block header hash, or error if it's not found in Store.
func (s *DefaultStore) LoadBlockByHash(hash [32]byte) (*types.Block, error) {
	blockData, err := s.db.Get(getBlockKey(hash))
	if err != nil {
		return nil, fmt.Errorf("failed to load block data: %w", err)
	}
	block := new(types.Block)
	err = block.UnmarshalBinary(blockData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal block data: %w", err)
	}

	return block, nil
}

// SaveBlockResponses saves block responses (events, tx responses, validator set updates, etc) in Store.
func (s *DefaultStore) SaveBlockResponses(height uint64, responses *tmstate.ABCIResponses, batch Batch) (Batch, error) {
	data, err := responses.Marshal()
	if err != nil {
		return batch, fmt.Errorf("failed to marshal response: %w", err)
	}
	if batch == nil {
		return nil, s.db.Set(getResponsesKey(height), data)
	}
	err = batch.Set(getResponsesKey(height), data)
	return batch, err
}

// LoadBlockResponses returns block results at given height, or error if it's not found in Store.
func (s *DefaultStore) LoadBlockResponses(height uint64) (*tmstate.ABCIResponses, error) {
	data, err := s.db.Get(getResponsesKey(height))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve block results from height %v: %w", height, err)
	}
	var responses tmstate.ABCIResponses
	err = responses.Unmarshal(data)
	if err != nil {
		return &responses, fmt.Errorf("failed to unmarshal data: %w", err)
	}
	return &responses, nil
}

// LoadCommit returns commit for a block at given height, or error if it's not found in Store.
func (s *DefaultStore) LoadCommit(height uint64) (*types.Commit, error) {
	hash, err := s.loadHashFromIndex(height)
	if err != nil {
		return nil, fmt.Errorf("failed to load hash from index: %w", err)
	}
	return s.LoadCommitByHash(hash)
}

// LoadCommitByHash returns commit for a block with given block header hash, or error if it's not found in Store.
func (s *DefaultStore) LoadCommitByHash(hash [32]byte) (*types.Commit, error) {
	commitData, err := s.db.Get(getCommitKey(hash))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve commit from hash %v: %w", hash, err)
	}
	commit := new(types.Commit)
	err = commit.UnmarshalBinary(commitData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Commit into object: %w", err)
	}
	return commit, nil
}

// UpdateState updates state saved in Store. Only one State is stored.
// If there is no State in Store, state will be saved.
func (s *DefaultStore) UpdateState(state types.State, batch Batch) (Batch, error) {
	pbState, err := state.ToProto()
	if err != nil {
		return batch, fmt.Errorf("failed to marshal state to JSON: %w", err)
	}
	data, err := pbState.Marshal()
	if err != nil {
		return batch, err
	}

	if batch == nil {
		return nil, s.db.Set(getStateKey(), data)
	}
	err = batch.Set(getStateKey(), data)
	return batch, err
}

// LoadState returns last state saved with UpdateState.
func (s *DefaultStore) LoadState() (types.State, error) {
	blob, err := s.db.Get(getStateKey())
	if err != nil {
		return types.State{}, types.ErrNoStateFound
	}
	var pbState pb.State
	err = pbState.Unmarshal(blob)
	if err != nil {
		return types.State{}, fmt.Errorf("failed to unmarshal state from store: %w", err)
	}

	var state types.State
	err = state.FromProto(&pbState)
	if err != nil {
		return types.State{}, fmt.Errorf("failed to unmarshal state from proto: %w", err)
	}

	atomic.StoreUint64(&s.height, state.LastStoreHeight)
	atomic.StoreUint64(&s.baseHeight, state.BaseHeight)
	return state, nil
}

// SaveValidators stores validator set for given block height in store.
func (s *DefaultStore) SaveValidators(height uint64, validatorSet *tmtypes.ValidatorSet, batch Batch) (Batch, error) {
	pbValSet, err := validatorSet.ToProto()
	if err != nil {
		return batch, fmt.Errorf("failed to marshal ValidatorSet to protobuf: %w", err)
	}
	blob, err := pbValSet.Marshal()
	if err != nil {
		return batch, fmt.Errorf("failed to marshal ValidatorSet: %w", err)
	}

	if batch == nil {
		return nil, s.db.Set(getValidatorsKey(height), blob)
	}
	err = batch.Set(getValidatorsKey(height), blob)
	return batch, err
}

// LoadValidators loads validator set at given block height from store.
func (s *DefaultStore) LoadValidators(height uint64) (*tmtypes.ValidatorSet, error) {
	blob, err := s.db.Get(getValidatorsKey(height))
	if err != nil {
		return nil, fmt.Errorf("failed to load Validators for height %v: %w", height, err)
	}
	var pbValSet tmproto.ValidatorSet
	err = pbValSet.Unmarshal(blob)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal to protobuf: %w", err)
	}

	return tmtypes.ValidatorSetFromProto(&pbValSet)
}

func (s *DefaultStore) loadHashFromIndex(height uint64) ([32]byte, error) {
	blob, err := s.db.Get(getIndexKey(height))

	var hash [32]byte
	if err != nil {
		return hash, fmt.Errorf("failed to load block hash for height %v: %w", height, err)
	}
	if len(blob) != len(hash) {
		return hash, errors.New("invalid hash length")
	}
	copy(hash[:], blob)
	return hash, nil
}

func getBlockKey(hash [32]byte) []byte {
	return append(blockPrefix[:], hash[:]...)
}

func getCommitKey(hash [32]byte) []byte {
	return append(commitPrefix[:], hash[:]...)
}

func getIndexKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(indexPrefix[:], buf[:]...)
}

func getStateKey() []byte {
	return statePrefix[:]
}

func getResponsesKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(responsesPrefix[:], buf[:]...)
}

func getValidatorsKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(validatorsPrefix[:], buf[:]...)
}
