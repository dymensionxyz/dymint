package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ipfs/go-cid"
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
	sequencersPrefix = [1]byte{6}
	cidPrefix        = [1]byte{7}
	validationPrefix = [1]byte{8}
)

var (
	blockNonValidated = [1]byte{0}
	blockValidated    = [1]byte{1}
)

// DefaultStore is a default store implementation.
type DefaultStore struct {
	db KV
}

var _ Store = &DefaultStore{}

// New returns new, default store.
func New(kv KV) Store {
	return &DefaultStore{
		db: kv,
	}
}

// Close implements Store.
func (s *DefaultStore) Close() error {
	return s.db.Close()
}

// NewBatch creates a new db batch.
func (s *DefaultStore) NewBatch() KVBatch {
	return s.db.NewBatch()
}

// SaveBlock adds block to the store along with corresponding commit.
// Stored height is updated if block height is greater than stored value.
// In case a batch is provided, the block and commit are added to the batch and not saved.
func (s *DefaultStore) SaveBlock(block *types.Block, commit *types.Commit, batch KVBatch) (KVBatch, error) {
	hash := block.Header.Hash()
	blockBlob, err := block.MarshalBinary()
	if err != nil {
		return batch, fmt.Errorf("marshal Block to binary: %w", err)
	}

	commitBlob, err := commit.MarshalBinary()
	if err != nil {
		return batch, fmt.Errorf("marshal Commit to binary: %w", err)
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
		return batch, fmt.Errorf("create db batch: %w", err)
	}

	if err = bb.Commit(); err != nil {
		return nil, fmt.Errorf("commit db batch: %w", err)
	}

	return nil, nil
}

// LoadBlock returns block at given height, or error if it's not found in Store.
// TODO(tzdybal): what is more common access pattern? by height or by hash?
// currently, we're indexing height->hash, and store blocks by hash, but we might as well store by height
// and index hash->height
func (s *DefaultStore) LoadBlock(height uint64) (*types.Block, error) {
	h, err := s.loadHashFromIndex(height)
	if err != nil {
		return nil, fmt.Errorf("load hash from index: %w", err)
	}
	return s.LoadBlockByHash(h)
}

// LoadBlockByHash returns block with given block header hash, or error if it's not found in Store.
func (s *DefaultStore) LoadBlockByHash(hash [32]byte) (*types.Block, error) {
	blockData, err := s.db.Get(getBlockKey(hash))
	if err != nil {
		return nil, fmt.Errorf("load block data: %w", err)
	}
	block := new(types.Block)
	err = block.UnmarshalBinary(blockData)
	if err != nil {
		return nil, fmt.Errorf("unmarshal block data: %w", err)
	}

	return block, nil
}

// SaveBlockValidation saves block validation in Store.
func (s *DefaultStore) SaveBlockValidation(height uint64, validation bool, batch KVBatch) (KVBatch, error) {

	var data []byte
	if validation {
		data = blockValidated[:]
	} else {
		data = blockNonValidated[:]
	}
	err := batch.Set(getValidationKey(height), data)
	return batch, err
}

// LoadBlockValidation returns block validation in Store.
func (s *DefaultStore) LoadBlockValidation(height uint64) (bool, error) {

	data, err := s.db.Get(getValidationKey(height))
	if err != nil {
		return false, fmt.Errorf("retrieve block results from height %v: %w", height, err)
	}
	if bytes.Equal(data[:], blockValidated[:]) {
		return true, nil
	}
	return false, nil

}

// SaveBlockResponses saves block responses (events, tx responses, etc) in Store.
func (s *DefaultStore) SaveBlockResponses(height uint64, responses *tmstate.ABCIResponses, batch KVBatch) (KVBatch, error) {
	data, err := responses.Marshal()
	if err != nil {
		return batch, fmt.Errorf("marshal response: %w", err)
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
		return nil, fmt.Errorf("retrieve block results from height %v: %w", height, err)
	}
	var responses tmstate.ABCIResponses
	err = responses.Unmarshal(data)
	if err != nil {
		return &responses, fmt.Errorf("unmarshal data: %w", err)
	}
	return &responses, nil
}

// LoadCommit returns commit for a block at given height, or error if it's not found in Store.
func (s *DefaultStore) LoadCommit(height uint64) (*types.Commit, error) {
	hash, err := s.loadHashFromIndex(height)
	if err != nil {
		return nil, fmt.Errorf("load hash from index: %w", err)
	}
	return s.LoadCommitByHash(hash)
}

// LoadCommitByHash returns commit for a block with given block header hash, or error if it's not found in Store.
func (s *DefaultStore) LoadCommitByHash(hash [32]byte) (*types.Commit, error) {
	commitData, err := s.db.Get(getCommitKey(hash))
	if err != nil {
		return nil, fmt.Errorf("retrieve commit from hash %v: %w", hash, err)
	}
	commit := new(types.Commit)
	err = commit.UnmarshalBinary(commitData)
	if err != nil {
		return nil, fmt.Errorf("marshal Commit into object: %w", err)
	}
	return commit, nil
}

// SaveState updates state saved in Store. Only one State is stored.
// If there is no State in Store, state will be saved.
func (s *DefaultStore) SaveState(state *types.State, batch KVBatch) (KVBatch, error) {
	pbState, err := state.ToProto()
	if err != nil {
		return batch, fmt.Errorf("marshal state to JSON: %w", err)
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
func (s *DefaultStore) LoadState() (*types.State, error) {
	blob, err := s.db.Get(getStateKey())
	if err != nil {
		return nil, types.ErrNoStateFound
	}
	var pbState pb.State
	err = pbState.Unmarshal(blob)
	if err != nil {
		return nil, fmt.Errorf("unmarshal state from store: %w", err)
	}

	var state types.State
	err = state.FromProto(&pbState)
	if err != nil {
		return nil, fmt.Errorf("unmarshal state from proto: %w", err)
	}

	return &state, nil
}

// SaveSequencers stores sequencerSet for given block height in store.
func (s *DefaultStore) SaveSequencers(height uint64, sequencerSet *types.SequencerSet, batch KVBatch) (KVBatch, error) {
	pbValSet, err := sequencerSet.ToProto()
	if err != nil {
		return batch, fmt.Errorf("marshal sequencerSet to protobuf: %w", err)
	}
	blob, err := pbValSet.Marshal()
	if err != nil {
		return batch, fmt.Errorf("marshal sequencerSet: %w", err)
	}

	if batch == nil {
		return nil, s.db.Set(getSequencersKey(height), blob)
	}
	err = batch.Set(getSequencersKey(height), blob)
	return batch, err
}

// LoadSequencers loads sequencer set at given block height from store.
func (s *DefaultStore) LoadSequencers(height uint64) (*types.SequencerSet, error) {
	blob, err := s.db.Get(getSequencersKey(height))
	if err != nil {
		return nil, fmt.Errorf("load sequencers for height %v: %w", height, err)
	}
	var pbValSet pb.SequencerSet
	err = pbValSet.Unmarshal(blob)
	if err != nil {
		// migration support: try to unmarshal as old ValidatorSet
		return parseAsValidatorSet(blob)
	}

	var ss types.SequencerSet
	err = ss.FromProto(pbValSet)
	if err != nil {
		return nil, fmt.Errorf("unmarshal from proto: %w", err)
	}

	return &ss, nil
}

func parseAsValidatorSet(blob []byte) (*types.SequencerSet, error) {
	var (
		ss          types.SequencerSet
		pbValSetOld tmproto.ValidatorSet
	)
	err := pbValSetOld.Unmarshal(blob)
	if err != nil {
		return nil, fmt.Errorf("unmarshal protobuf: %w", err)
	}
	pbValSet, err := tmtypes.ValidatorSetFromProto(&pbValSetOld)
	if err != nil {
		return nil, fmt.Errorf("unmarshal to ValidatorSet: %w", err)
	}
	ss.LoadFromValSet(pbValSet)
	return &ss, nil
}

func (s *DefaultStore) loadHashFromIndex(height uint64) ([32]byte, error) {
	blob, err := s.db.Get(getIndexKey(height))

	var hash [32]byte
	if err != nil {
		return hash, fmt.Errorf("load block hash for height %v: %w", height, err)
	}
	if len(blob) != len(hash) {
		return hash, errors.New("invalid hash length")
	}
	copy(hash[:], blob)
	return hash, nil
}

func (s *DefaultStore) SaveBlockCid(height uint64, cid cid.Cid, batch KVBatch) (KVBatch, error) {
	if batch == nil {
		return nil, s.db.Set(getCidKey(height), []byte(cid.String()))
	}
	err := batch.Set(getCidKey(height), []byte(cid.String()))
	return batch, err
}

func (s *DefaultStore) LoadBlockCid(height uint64) (cid.Cid, error) {
	cidBytes, err := s.db.Get(getCidKey(height))
	if err != nil {
		return cid.Undef, fmt.Errorf("load cid for height %v: %w", height, err)
	}
	parsedCid, err := cid.Parse(string(cidBytes))
	if err != nil {
		return cid.Undef, fmt.Errorf("parse cid: %w", err)
	}
	return parsedCid, nil
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

func getSequencersKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(sequencersPrefix[:], buf[:]...)
}

func getCidKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(cidPrefix[:], buf[:]...)
}

func getValidationKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(validationPrefix[:], buf[:]...)
}
