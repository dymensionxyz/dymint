package store_test

import (
	"testing"

	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/stretchr/testify/assert"
)

func TestStorePruning(t *testing.T) {
	t.Parallel()

	pruningHeight := uint64(3)

	cases := []struct {
		name           string
		blocks         []*types.Block
		pruningHeight  uint64
		expectedBase   uint64
		expectedHeight uint64
		shouldError    bool
	}{
		{"blocks with pruning", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
			testutil.GetRandomBlock(4, 0),
			testutil.GetRandomBlock(5, 0),
		}, pruningHeight, pruningHeight, 5, false},
		{"blocks out of order", []*types.Block{
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
			testutil.GetRandomBlock(1, 0),
		}, pruningHeight, pruningHeight, 3, false},
		{"with a gap", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(9, 0),
			testutil.GetRandomBlock(10, 0),
		}, pruningHeight, pruningHeight, 10, false},
		{"pruning beyond latest height", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
		}, pruningHeight, 1, 2, true}, // should error because pruning height > latest height
		{"pruning height 0", []*types.Block{
			testutil.GetRandomBlock(1, 0),
			testutil.GetRandomBlock(2, 0),
			testutil.GetRandomBlock(3, 0),
		}, 0, 1, 3, true},
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

			_, err := bstore.PruneBlocks(int64(c.pruningHeight))
			if c.shouldError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(pruningHeight, bstore.Base())
				assert.Equal(c.expectedHeight, bstore.Height())
				assert.Equal(c.expectedBase, bstore.Base())

				// Check if pruned blocks are really removed from the store
				for h := uint64(1); h < pruningHeight; h++ {
					_, err := bstore.LoadBlock(h)
					assert.Error(err, "Block at height %d should be pruned", h)

					_, err = bstore.LoadBlockResponses(h)
					assert.Error(err, "BlockResponse at height %d should be pruned", h)

					_, err = bstore.LoadCommit(h)
					assert.Error(err, "Commit at height %d should be pruned", h)
				}
			}
		})
	}
}

// TODO: prune twice
