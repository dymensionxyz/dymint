package block_test

import (
	"context"
	"testing"

	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
)

func TestPruningRetainHeight(t *testing.T) {
	require := require.New(t)
	app := testutil.GetAppMock()
	ctx := context.Background()
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)

	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	// Check initial assertions
	require.Zero(manager.State.Height())
	require.Zero(manager.LastSubmittedHeight.Load())

	batchSize := 10

	// Produce blocks
	for i := 0; i < batchSize; i++ {
		_, _, err = manager.ProduceAndGossipBlock(ctx, true)
		require.NoError(err)
	}
	// submit and validate sync target
	manager.CreateAndSubmitBatch(100000000)
	lastSubmitted := manager.LastSubmittedHeight.Load()
	assert.EqualValues(t, manager.State.Height(), lastSubmitted)
	assert.Equal(t, lastSubmitted, uint64(batchSize))

	// Produce new blocks
	for i := 0; i < batchSize; i++ {
		_, _, err = manager.ProduceAndGossipBlock(ctx, true)
		require.NoError(err)
	}
	validRetainHeight := manager.NextHeightToSubmit() // the max possible valid retain height
	for i := validRetainHeight; i < manager.State.Height(); i++ {
		expectedPruned := validRetainHeight - manager.State.BaseHeight
		pruned, err := manager.Store.PruneStore(manager.State.BaseHeight, validRetainHeight, log.NewNopLogger())
		if i <= validRetainHeight {
			require.NoError(err)
			assert.Equal(t, expectedPruned, pruned)
		} else {
			require.Error(gerrc.ErrInvalidArgument)
		}
	}

	err = manager.PruneBlocks(validRetainHeight)
	require.NoError(err)
}
