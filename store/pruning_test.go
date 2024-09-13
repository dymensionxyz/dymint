package store_test

import (
	"testing"

	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
)

func TestStorePruning(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		blocks      []*types.Block
		from        uint64
		to          uint64
		shouldError bool
	}{
		{"blocks with pruning", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
			testutil.GetRandomBlock(4, 0),
			testutil.GetRandomBlock(5, 0),
		}, 3, 5, false},
		{"blocks out of order", []*types.Block{
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(5, 0),
		}, 3, 5, false},
		{"with a gap", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(9, 0),
			testutil.GetRandomBlock(10, 0),
		}, 3, 5, false},
		{"pruning height 0", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
		}, 0, 1, true},
		{"pruning same height", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
		}, 3, 3, true},
		{"to height exceeds actual block cnt", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
		}, 2, 5, false}, // it shouldn't error it should just no-op
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert := assert.New(t)
			bstore := store.New(store.NewDefaultInMemoryKVStore())

			savedHeights := make(map[uint64]bool)
			for _, block := range c.blocks {
				_, err := bstore.SaveBlock(block, &types.Commit{}, nil)
				assert.NoError(err)
				savedHeights[block.Header.Height] = true
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

				// TODO: add block responses and commits
			}

			// And then feed it some data
			// expectedCid, err := pref.Sum(block)
			// Validate all blocks are saved
			for k := range savedHeights {
				_, err := bstore.LoadBlock(k)
				assert.NoError(err)
			}

			_, err := bstore.PruneBlocks(c.from, c.to)
			if c.shouldError {
				assert.Error(err)
				return
			}

			assert.NoError(err)

			// Validate only blocks in the range are pruned
			for k := range savedHeights {
				if k >= c.from && k < c.to { // k < c.to is the exclusion test
					_, err := bstore.LoadBlock(k)
					assert.Error(err, "Block at height %d should be pruned", k)

					_, err = bstore.LoadBlockResponses(k)
					assert.Error(err, "BlockResponse at height %d should be pruned", k)

					_, err = bstore.LoadCommit(k)
					assert.Error(err, "Commit at height %d should be pruned", k)

					_, err = bstore.LoadBlockCid(k)
					assert.Error(err, "Cid at height %d should be pruned", k)

				} else {
					_, err := bstore.LoadBlock(k)
					assert.NoError(err)

					_, err = bstore.LoadBlockCid(k)
					assert.NoError(err)

				}
			}
		})
	}
}

// TODO: prune twice
