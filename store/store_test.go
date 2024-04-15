package store_test

import (
	"os"
	"testing"

	abcitypes "github.com/tendermint/tendermint/abci/types"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"

	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreHeight(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		blocks   []*types.Block
		expected uint64
	}{
		{"single block", []*types.Block{testutil.GetRandomBlock(1, 0)}, 1},
		{"two consecutive blocks", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
		}, 2},
		{"blocks out of order", []*types.Block{
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
			testutil.GetRandomBlock(1, 0),
		}, 3},
		{"with a gap", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(9, 0),
			testutil.GetRandomBlock(10, 0),
		}, 10},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert := assert.New(t)
			bstore := store.New(store.NewDefaultInMemoryKVStore())
			assert.Equal(uint64(0), bstore.Height())

			for _, block := range c.blocks {
				_, err := bstore.SaveBlock(block, &types.Commit{}, nil)
				bstore.SetHeight(block.Header.Height)
				assert.NoError(err)
			}

			assert.Equal(c.expected, bstore.Height())
		})
	}
}

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

	tmpDir, err := os.MkdirTemp("", "optimint_test")
	require.NoError(t, err)
	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Log("failed to remove temporary directory", err)
		}
	}()

	for _, kv := range []store.KVStore{store.NewDefaultInMemoryKVStore(), store.NewDefaultKVStore(tmpDir, "db", "test")} {
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

func TestRestart(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	validatorSet := testutil.GetRandomValidatorSet()

	kv := store.NewDefaultInMemoryKVStore()
	s1 := store.New(kv)
	expectedHeight := uint64(10)
	_, err := s1.UpdateState(types.State{
		LastBlockHeight: int64(expectedHeight),
		LastStoreHeight: uint64(expectedHeight),
		NextValidators:  validatorSet,
		Validators:      validatorSet,
		LastValidators:  validatorSet,
	}, nil)
	assert.NoError(err)

	s2 := store.New(kv)
	_, err = s2.LoadState()
	assert.NoError(err)

	assert.Equal(expectedHeight, s2.Height())
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
	assert.EqualError(err, "retrieve block results from height 1: key not found") // TODO: use errors.Is
	assert.Nil(resp)

	err = batch.Commit()
	assert.NoError(err)

	resp, err = s.LoadBlockResponses(1)
	assert.NoError(err)
	assert.NotNil(resp)
	assert.Equal(expected, resp)
}
