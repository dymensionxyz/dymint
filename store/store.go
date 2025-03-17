package store

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ipfs/go-cid"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	"go.uber.org/multierr"

	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
)

var (
	blockPrefix                 = [1]byte{1}
	indexPrefix                 = [1]byte{2}
	commitPrefix                = [1]byte{3}
	statePrefix                 = [1]byte{4}
	responsesPrefix             = [1]byte{5}
	proposerPrefix              = [1]byte{6}
	cidPrefix                   = [1]byte{7}
	sourcePrefix                = [1]byte{8}
	validatedHeightPrefix       = [1]byte{9}
	baseHeightPrefix            = [1]byte{10}
	blocksyncBaseHeightPrefix   = [1]byte{11}
	indexerBaseHeightPrefix     = [1]byte{12}
	drsVersionPrefix            = [1]byte{13}
	lastBlockSequencerSetPrefix = [1]byte{14}
	daPrefix                    = [1]byte{15}
)

// DefaultStore is a default store implementation.
type DefaultStore struct {
	db KV
}

var _ Store = &DefaultStore{}

// New returns new, default store.
func New(kv KV) *DefaultStore {
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

// SaveBlockSource saves block validation in Store.
func (s *DefaultStore) SaveBlockSource(height uint64, source types.BlockSource, batch KVBatch) (KVBatch, error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(source))
	if batch == nil {
		return nil, s.db.Set(getSourceKey(height), b)
	}
	err := batch.Set(getSourceKey(height), b)
	return batch, err
}

// LoadBlockSource returns block validation in Store.
func (s *DefaultStore) LoadBlockSource(height uint64) (types.BlockSource, error) {
	source, err := s.db.Get(getSourceKey(height))
	if err != nil {
		return types.BlockSource(0), fmt.Errorf("get block source for height %v: %w", height, err)
	}
	return types.BlockSource(binary.LittleEndian.Uint64(source)), nil
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

// SaveProposer stores the proposer for given block height in store.
func (s *DefaultStore) SaveProposer(height uint64, proposer types.Sequencer, batch KVBatch) (KVBatch, error) {
	pbProposer, err := proposer.ToProto()
	if err != nil {
		return batch, fmt.Errorf("marshal proposer to protobuf: %w", err)
	}
	blob, err := pbProposer.Marshal()
	if err != nil {
		return batch, fmt.Errorf("marshal proposer: %w", err)
	}

	if batch == nil {
		return nil, s.db.Set(getProposerKey(height), blob)
	}
	err = batch.Set(getProposerKey(height), blob)
	return batch, err
}

// LoadProposer loads proposer at given block height from store.
func (s *DefaultStore) LoadProposer(height uint64) (types.Sequencer, error) {
	blob, err := s.db.Get(getProposerKey(height))
	if err != nil {
		return types.Sequencer{}, fmt.Errorf("load proposer for height %v: %w", height, err)
	}

	pbProposer := new(pb.Sequencer)
	err = pbProposer.Unmarshal(blob)
	if err != nil {
		return types.Sequencer{}, fmt.Errorf("parsing blob as proposer: %w", err)
	}
	proposer, err := types.SequencerFromProto(pbProposer)
	if err != nil {
		return types.Sequencer{}, fmt.Errorf("unmarshal proposer from proto: %w", err)
	}

	return *proposer, nil
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

func (s *DefaultStore) SaveValidationHeight(height uint64, batch KVBatch) (KVBatch, error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, height)
	if batch == nil {
		return nil, s.db.Set(getValidatedHeightKey(), b)
	}
	err := batch.Set(getValidatedHeightKey(), b)
	return batch, err
}

func (s *DefaultStore) LoadValidationHeight() (uint64, error) {
	b, err := s.db.Get(getValidatedHeightKey())
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func (s *DefaultStore) RemoveBlockCid(height uint64) error {
	err := s.db.Delete(getCidKey(height))
	return err
}

func (s *DefaultStore) LoadDA(height uint64) (string, error) {
	resp, err := s.LoadBlockResponses(height)
	if err != nil {
		return "", err
	}
	return resp.EndBlock.RollappParamUpdates.Da, nil
}

func (s *DefaultStore) LoadDRSVersion(height uint64) (uint32, error) {
	resp, err := s.LoadBlockResponses(height)
	if err != nil {
		return 0, err
	}
	return resp.EndBlock.RollappParamUpdates.DrsVersion, nil
}

func (s *DefaultStore) LoadBaseHeight() (uint64, error) {
	b, err := s.db.Get(getBaseHeightKey())
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func (s *DefaultStore) SaveBaseHeight(height uint64) error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, height)
	return s.db.Set(getBaseHeightKey(), b)
}

func (s *DefaultStore) LoadBlockSyncBaseHeight() (uint64, error) {
	b, err := s.db.Get(getBlockSyncBaseHeightKey())
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func (s *DefaultStore) SaveBlockSyncBaseHeight(height uint64) error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, height)
	return s.db.Set(getBlockSyncBaseHeightKey(), b)
}

func (s *DefaultStore) LoadIndexerBaseHeight() (uint64, error) {
	b, err := s.db.Get(getIndexerBaseHeightKey())
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func (s *DefaultStore) SaveIndexerBaseHeight(height uint64) error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, height)
	return s.db.Set(getIndexerBaseHeightKey(), b)
}

func (s *DefaultStore) SaveLastBlockSequencerSet(sequencers types.Sequencers, batch KVBatch) (KVBatch, error) {
	pbSequencers, err := sequencers.ToProto()
	if err != nil {
		return batch, fmt.Errorf("marshal sequencers to protobuf: %w", err)
	}
	blob, err := pbSequencers.Marshal()
	if err != nil {
		return batch, fmt.Errorf("marshal sequencers: %w", err)
	}

	if batch == nil {
		return nil, s.db.Set(getLastBlockSequencerSetPrefixKey(), blob)
	}
	err = batch.Set(getLastBlockSequencerSetPrefixKey(), blob)
	return batch, err
}

func (s *DefaultStore) LoadLastBlockSequencerSet() (types.Sequencers, error) {
	blob, err := s.db.Get(getLastBlockSequencerSetPrefixKey())
	if err != nil {
		return nil, fmt.Errorf("load sequencers: %w", err)
	}

	pbSequencers := new(pb.SequencerSet)
	err = pbSequencers.Unmarshal(blob)
	if err != nil {
		return nil, fmt.Errorf("unmarshal sequencers: %w", err)
	}

	sequencers, err := types.SequencersFromProto(pbSequencers)
	if err != nil {
		return nil, fmt.Errorf("unmarshal sequencers from proto: %w", err)
	}

	return sequencers, nil
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

func getCidKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(cidPrefix[:], buf[:]...)
}

func getSourceKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(sourcePrefix[:], buf[:]...)
}

func getValidatedHeightKey() []byte {
	return validatedHeightPrefix[:]
}

func getDRSVersionKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(drsVersionPrefix[:], buf[:]...)
}

func getDAKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(daPrefix[:], buf[:]...)
}

func getProposerKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(proposerPrefix[:], buf[:]...)
}

func getBaseHeightKey() []byte {
	return baseHeightPrefix[:]
}

func getBlockSyncBaseHeightKey() []byte {
	return blocksyncBaseHeightPrefix[:]
}

func getIndexerBaseHeightKey() []byte {
	return indexerBaseHeightPrefix[:]
}

func getLastBlockSequencerSetPrefixKey() []byte {
	return lastBlockSequencerSetPrefix[:]
}
