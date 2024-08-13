package block_test

import (
	"context"
	"testing"

	"github.com/dymensionxyz/dymint/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		_, _, err = manager.ProduceApplyGossipBlock(ctx, true)
		require.NoError(err)
	}
	// submit and validate sync target
	manager.CreateAndSubmitBatch(100000000, false)
	lastSubmitted := manager.LastSubmittedHeight.Load()
	assert.EqualValues(t, manager.State.Height(), lastSubmitted)
	assert.Equal(t, lastSubmitted, uint64(batchSize))

	// Produce new blocks
	for i := 0; i < batchSize; i++ {
		_, _, err = manager.ProduceApplyGossipBlock(ctx, true)
		require.NoError(err)
	}

	validRetainHeight := lastSubmitted + 1 // the max possible valid retain height
	for i := validRetainHeight + 1; i < manager.State.Height(); i++ {
		err = manager.PruneBlocks(i)
		require.Error(err) // cannot prune blocks before they have been submitted
	}

	err = manager.PruneBlocks(validRetainHeight)
	require.NoError(err)
}
