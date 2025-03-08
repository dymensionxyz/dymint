package store_test

import (
	"testing"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"

	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
)

func TestStoreLoad(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		blocks []*types.Block
	}{
		{"single block", []*types.Block{testutil.GetRandomBlock(1, 10)}},
		{"two consecutive blocks", []*types.Block{
			testutil.GetRandomBlock(1, 10),
			testutil.GetRandomBlock(2, 20),
		}},
		// TODO(tzdybal): this test needs extra handling because of lastCommits
		//{"blocks out of order", []*types.Block{
		//	getRandomBlock(2, 20),
		//	getRandomBlock(3, 30),
		//	getRandomBlock(4, 100),
		//	getRandomBlock(5, 10),
		//	getRandomBlock(1, 10),
		//}},
	}

	tmpDir := t.TempDir()

	for _, kv := range []store.KV{store.NewDefaultInMemoryKVStore(), store.NewDefaultKVStore(tmpDir, "db", "test")} {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				assert := assert.New(t)
				require := require.New(t)

				bstore := store.New(kv)

				lastCommit := &types.Commit{}
				for _, block := range c.blocks {
					commit := &types.Commit{Height: block.Header.Height, HeaderHash: block.Header.Hash()}
					block.LastCommit = *lastCommit
					_, err := bstore.SaveBlock(block, commit, nil)
					require.NoError(err)
					lastCommit = commit
				}

				for _, expected := range c.blocks {
					block, err := bstore.LoadBlock(expected.Header.Height)
					assert.NoError(err)
					assert.NotNil(block)
					assert.Equal(expected, block)
					assert.Equal(expected.Header.Height-1, block.LastCommit.Height)
					assert.Equal(expected.LastCommit.Height, block.LastCommit.Height)
					assert.Equal(expected.LastCommit.HeaderHash, block.LastCommit.HeaderHash)

					commit, err := bstore.LoadCommit(expected.Header.Height)
					assert.NoError(err)
					assert.NotNil(commit)
					assert.Equal(expected.Header.Height, commit.Height)
					headerHash := expected.Header.Hash()
					assert.Equal(headerHash, commit.HeaderHash)
				}
			})
		}
	}
}

func TestLoadState(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	kv := store.NewDefaultInMemoryKVStore()
	s1 := store.New(kv)
	expectedHeight := uint64(10)
	s := &types.State{}

	s.SetHeight(expectedHeight)
	_, err := s1.SaveState(s, nil)
	assert.NoError(err)

	s2 := store.New(kv)
	state, err := s2.LoadState()
	assert.NoError(err)

	assert.Equal(expectedHeight, state.Height())
}

func TestBlockResponses(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	kv := store.NewDefaultInMemoryKVStore()
	s := store.New(kv)

	expected := &tmstate.ABCIResponses{
		BeginBlock: &abcitypes.ResponseBeginBlock{
			Events: []abcitypes.Event{{
				Type: "test",
				Attributes: []abcitypes.EventAttribute{{
					Key:   []byte("foo"),
					Value: []byte("bar"),
					Index: false,
				}},
			}},
		},
		DeliverTxs: nil,
		EndBlock: &abcitypes.ResponseEndBlock{
			ValidatorUpdates: nil,
			ConsensusParamUpdates: &abcitypes.ConsensusParams{
				Block: &abcitypes.BlockParams{
					MaxBytes: 12345,
					MaxGas:   678909876,
				},
			},
		},
	}

	_, err := s.SaveBlockResponses(1, expected, nil)
	assert.NoError(err)

	resp, err := s.LoadBlockResponses(123)
	assert.Error(err)
	assert.Nil(resp)

	resp, err = s.LoadBlockResponses(1)
	assert.NoError(err)
	assert.NotNil(resp)
	assert.Equal(expected, resp)
}

func TestBatch(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	kv := store.NewDefaultInMemoryKVStore()
	s := store.New(kv)

	expected := &tmstate.ABCIResponses{
		BeginBlock: &abcitypes.ResponseBeginBlock{
			Events: []abcitypes.Event{{
				Type: "test",
				Attributes: []abcitypes.EventAttribute{{
					Key:   []byte("foo"),
					Value: []byte("bar"),
					Index: false,
				}},
			}},
		},
		DeliverTxs: nil,
		EndBlock: &abcitypes.ResponseEndBlock{
			ValidatorUpdates: nil,
			ConsensusParamUpdates: &abcitypes.ConsensusParams{
				Block: &abcitypes.BlockParams{
					MaxBytes: 12345,
					MaxGas:   678909876,
				},
			},
		},
	}

	batch := s.NewBatch()
	batch, err := s.SaveBlockResponses(1, expected, batch)
	assert.NoError(err)

	resp, err := s.LoadBlockResponses(1)
	assert.Error(err, gerrc.ErrNotFound)
	assert.Nil(resp)

	err = batch.Commit()
	assert.NoError(err)

	resp, err = s.LoadBlockResponses(1)
	assert.NoError(err)
	assert.NotNil(resp)
	assert.Equal(expected, resp)
}

// test for saving and loading cids for specific block heights in the store with and w/out batches
func TestBlockId(t *testing.T) {
	require := require.New(t)

	kv := store.NewDefaultInMemoryKVStore()
	s := store.New(kv)

	// Create a cid manually by specifying the 'prefix' parameters
	pref := &cid.Prefix{
		Codec:    cid.DagProtobuf,
		MhLength: -1,
		MhType:   mh.SHA2_256,
		Version:  1,
	}

	// And then feed it some data
	expectedCid, err := pref.Sum([]byte("test"))
	require.NoError(err)

	// store cid for height 1
	_, err = s.SaveBlockCid(1, expectedCid, nil)
	require.NoError(err)

	// retrieve cid for height 1
	resultCid, err := s.LoadBlockCid(1)
	require.NoError(err)

	require.Equal(expectedCid, resultCid)

	// repeat test using batch
	batch := s.NewBatch()

	// store cid for height 2
	batch, err = s.SaveBlockCid(2, expectedCid, batch)
	require.NoError(err)

	// retrieve cid for height 2
	_, err = s.LoadBlockCid(2)
	require.Error(err, gerrc.ErrNotFound)

	// commit
	batch.Commit()

	// retrieve cid for height 2
	resultCid, err = s.LoadBlockCid(2)
	require.NoError(err)
	require.Equal(expectedCid, resultCid)
}

func TestProposer(t *testing.T) {
	t.Parallel()

	expected := testutil.GenerateSequencer()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		s := store.New(store.NewDefaultInMemoryKVStore())

		_, err := s.SaveProposer(1, expected, nil)
		require.NoError(t, err)

		resp, err := s.LoadProposer(1)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, expected, resp)
	})

	t.Run("proposer not found", func(t *testing.T) {
		t.Parallel()

		s := store.New(store.NewDefaultInMemoryKVStore())

		_, err := s.SaveProposer(2, expected, nil)
		require.NoError(t, err)

		_, err = s.LoadProposer(2000)
		require.Error(t, err)
	})

	t.Run("empty proposer is invalid", func(t *testing.T) {
		t.Parallel()

		s := store.New(store.NewDefaultInMemoryKVStore())

		_, err := s.SaveProposer(3, types.Sequencer{}, nil)
		require.Error(t, err)

		_, err = s.LoadProposer(3)
		require.Error(t, err)
	})
}
