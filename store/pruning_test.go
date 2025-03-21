package store_test

import (
	"testing"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proto/tendermint/state"
	"golang.org/x/exp/rand"

	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
)

func TestStorePruning(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		blocks []*types.Block
		from   uint64
		to     uint64
		pruned uint64
	}{
		{"blocks with pruning", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
			testutil.GetRandomBlock(4, 0),
			testutil.GetRandomBlock(5, 0),
		}, 3, 5, 2},
		{"blocks out of order", []*types.Block{
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(5, 0),
		}, 3, 5, 1},
		{"with a gap", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(9, 0),
			testutil.GetRandomBlock(10, 0),
		}, 3, 5, 0},
		{"pruning height 0", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
		}, 0, 1, 0},
		{"pruning same height", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
		}, 3, 3, 0},
		{"to height exceeds actual block cnt", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
		}, 2, 5, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert := assert.New(t)
			bstore := store.New(store.NewDefaultInMemoryKVStore())

			savedBlockHeights := make(map[uint64]bool)
			savedRespHeights := make(map[uint64]bool)
			savedSeqHeights := make(map[uint64]bool)
			savedCidHeights := make(map[uint64]bool)
			savedDRSHeights := make(map[uint64]bool)

			bstore.SaveBaseHeight(c.from)
			for _, block := range c.blocks {

				_, err := bstore.SaveBlock(block, &types.Commit{Height: block.Header.Height}, nil)
				assert.NoError(err)
				savedBlockHeights[block.Header.Height] = true

				// generate and store block responses randomly for block heights
				if randBool() {
					_, err = bstore.SaveBlockResponses(block.Header.Height, &state.ABCIResponses{}, nil)
					savedRespHeights[block.Header.Height] = true
					assert.NoError(err)
				}

				// generate and store sequencers randomly for block heights
				if randBool() {
					_, err = bstore.SaveProposer(block.Header.Height, testutil.GenerateSequencer(), nil)
					savedSeqHeights[block.Header.Height] = true
					assert.NoError(err)
				}

				// generate and store cids randomly for block heights
				if randBool() {
					// generate cid from block
					blockBytes, err := block.MarshalBinary()
					assert.NoError(err)
					// Create a cid manually by specifying the 'prefix' parameters
					pref := &cid.Prefix{
						Codec:    cid.DagProtobuf,
						MhLength: -1,
						MhType:   mh.SHA2_256,
						Version:  1,
					}
					cid, err := pref.Sum(blockBytes)
					assert.NoError(err)
					_, err = bstore.SaveBlockCid(block.Header.Height, cid, nil)
					assert.NoError(err)
					savedCidHeights[block.Header.Height] = true
				}

			}

			// Validate everything is saved
			for k := range savedBlockHeights {
				_, err := bstore.LoadBlock(k)
				assert.NoError(err)
			}

			for k := range savedRespHeights {
				_, err := bstore.LoadBlockResponses(k)
				assert.NoError(err)
			}

			for k := range savedSeqHeights {
				_, err := bstore.LoadProposer(k)
				assert.NoError(err)
			}

			for k := range savedCidHeights {
				_, err := bstore.LoadBlockCid(k)
				assert.NoError(err)
			}

			for k := range savedDRSHeights {
				_, err := bstore.LoadDRSVersion(k)
				assert.NoError(err)
			}

			pruned, err := bstore.PruneStore(c.to, log.NewNopLogger())
			assert.Equal(c.pruned, pruned)
			assert.NoError(err)

			// Validate only blocks in the range are pruned
			for k := range savedBlockHeights {
				if k >= c.from && k < c.to { // k < c.to is the exclusion test
					_, err := bstore.LoadBlock(k)
					assert.Error(err, "Block at height %d should be pruned", k)

					_, err = bstore.LoadCommit(k)
					assert.Error(err, "Commit at height %d should be pruned", k)

				} else {
					_, err := bstore.LoadBlock(k)
					assert.NoError(err)

					_, err = bstore.LoadCommit(k)
					assert.NoError(err)

				}
			}

			// Validate only block responses in the range are pruned
			for k := range savedRespHeights {
				if k >= c.from && k < c.to { // k < c.to is the exclusion test
					_, err = bstore.LoadBlockResponses(k)
					assert.Error(err, "Block response at height %d should be pruned", k)
				} else {
					_, err = bstore.LoadBlockResponses(k)
					assert.NoError(err)
				}
			}

			// Validate only sequencers in the range are pruned
			for k := range savedSeqHeights {
				if k >= c.from && k < c.to { // k < c.to is the exclusion test
					_, err = bstore.LoadProposer(k)
					assert.Error(err, "Block cid at height %d should be pruned", k)
				} else {
					_, err = bstore.LoadProposer(k)
					assert.NoError(err)
				}
			}

			// Validate only block drs in the range are pruned
			for k := range savedDRSHeights {
				if k >= c.from && k < c.to { // k < c.to is the exclusion test
					_, err = bstore.LoadDRSVersion(k)
					assert.Error(err, "DRS version at height %d should be pruned", k)
				} else {
					_, err = bstore.LoadDRSVersion(k)
					assert.NoError(err)
				}
			}
		})
	}
}

// TODO: prune twice

func randBool() bool {
	return rand.Intn(2) == 0
}
