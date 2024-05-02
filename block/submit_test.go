package block_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/proxy"

	cosmosed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/dymensionxyz/dymint/config"
	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
)

var ctx = context.Background()

func TestBatchSubmissionHappyFlow(t *testing.T) {
	require := require.New(t)
	app := testutil.GetAppMock()
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)

	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	// Check initial assertions
	initialHeight := uint64(0)
	require.Zero(manager.Store.Height())
	require.Zero(manager.SyncTarget.Load())

	// Produce block and validate that we produced blocks
	err = manager.ProduceAndGossipBlock(ctx, true)
	require.NoError(err)
	assert.Greater(t, manager.Store.Height(), initialHeight)
	assert.Zero(t, manager.SyncTarget.Load())

	// submit and validate sync target
	manager.HandleSubmissionTrigger(ctx)
	assert.EqualValues(t, 1, manager.SyncTarget.Load())
}

func TestBatchSubmissionFailedSubmission(t *testing.T) {
	require := require.New(t)
	app := testutil.GetAppMock()

	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(err)

	lib2pPrivKey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	require.NoError(err)

	cosmosPrivKey := cosmosed25519.PrivKey{Key: priv}
	proposer := &types.Sequencer{
		PublicKey: cosmosPrivKey.PubKey(),
	}

	// Create a new mock LayerI
	mockLayerI := &mocks.MockLayerI{}
	mockLayerI.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockLayerI.On("Start").Return(nil)
	mockLayerI.On("GetProposer").Return(proposer)

	manager, err := testutil.GetManagerWithProposerKey(testutil.GetManagerConfig(), lib2pPrivKey, mockLayerI, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	// Check initial assertions
	initialHeight := uint64(0)
	require.Zero(manager.Store.Height())
	require.Zero(manager.SyncTarget.Load())

	// Produce block and validate that we produced blocks
	err = manager.ProduceAndGossipBlock(ctx, true)
	require.NoError(err)
	assert.Greater(t, manager.Store.Height(), initialHeight)
	assert.Zero(t, manager.SyncTarget.Load())

	// try to submit, we expect failure
	mockLayerI.On("SubmitBatch", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("Failed to submit batch")).Once()
	manager.HandleSubmissionTrigger(ctx)
	assert.EqualValues(t, 0, manager.SyncTarget.Load())

	// try to submit again, we expect success
	mockLayerI.On("SubmitBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	manager.HandleSubmissionTrigger(ctx)
	assert.EqualValues(t, 1, manager.SyncTarget.Load())
}

func TestBatchSubmissionAfterTimeout(t *testing.T) {
	const (
		// large batch size, so we expect the trigger to be the timeout
		batchSize     = 100000
		submitTimeout = 2 * time.Second
		blockTime     = 200 * time.Millisecond
		runTime       = submitTimeout + 1*time.Second
	)

	require := require.New(t)
	app := testutil.GetAppMock()
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)

	// Init manager with empty blocks feature enabled
	managerConfig := config.BlockManagerConfig{
		BlockTime:               blockTime,
		EmptyBlocksMaxTime:      0,
		BatchSubmitMaxTime:      submitTimeout,
		BlockBatchSize:          batchSize,
		BlockBatchMaxSizeBytes:  1000,
		GossipedBlocksCacheSize: 50,
	}

	manager, err := testutil.GetManager(managerConfig, nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	// Check initial height
	initialHeight := uint64(0)
	require.Equal(initialHeight, manager.Store.Height())
	require.Zero(manager.SyncTarget.Load())

	var wg sync.WaitGroup
	mCtx, cancel := context.WithTimeout(context.Background(), runTime)
	defer cancel()

	wg.Add(2) // Add 2 because we have 2 goroutines

	go func() {
		defer wg.Done() // Decrease counter when this goroutine finishes
		manager.ProduceBlockLoop(mCtx)
	}()

	go func() {
		defer wg.Done() // Decrease counter when this goroutine finishes
		manager.SubmitLoop(mCtx)
	}()

	<-mCtx.Done()
	wg.Wait() // Wait for all goroutines to finish
	require.True(manager.SyncTarget.Load() > 0)
}
