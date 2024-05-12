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

func TestBatchSubmissionHappyFlow(t *testing.T) {
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
	initialHeight := uint64(0)
	require.Zero(manager.State.Height())
	require.Zero(manager.SyncTarget.Load())

	// Produce block and validate that we produced blocks
	_, _, err = manager.ProduceAndGossipBlock(ctx, true)
	require.NoError(err)
	assert.Greater(t, manager.State.Height(), initialHeight)
	assert.Zero(t, manager.SyncTarget.Load())

	// submit and validate sync target
	manager.HandleSubmissionTrigger()
	assert.EqualValues(t, manager.State.Height(), manager.SyncTarget.Load())
}

func TestBatchSubmissionFailedSubmission(t *testing.T) {
	require := require.New(t)
	app := testutil.GetAppMock()
	ctx := context.Background()

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
	require.Zero(manager.State.Height())
	require.Zero(manager.SyncTarget.Load())

	// Produce block and validate that we produced blocks
	_, _, err = manager.ProduceAndGossipBlock(ctx, true)
	require.NoError(err)
	assert.Greater(t, manager.State.Height(), initialHeight)
	assert.Zero(t, manager.SyncTarget.Load())

	// try to submit, we expect failure
	mockLayerI.On("SubmitBatch", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("Failed to submit batch")).Once()
	assert.Error(t, manager.HandleSubmissionTrigger())

	// try to submit again, we expect success
	mockLayerI.On("SubmitBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	manager.HandleSubmissionTrigger()
	assert.EqualValues(t, manager.State.Height(), manager.SyncTarget.Load())
}

// TestSubmissionByTime tests the submission trigger by time
func TestSubmissionByTime(t *testing.T) {
	const (
		// large batch size, so we expect the trigger to be the timeout
		submitTimeout = 1 * time.Second
		blockTime     = 200 * time.Millisecond
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
		MaxIdleTime:             0,
		MaxSupportedBatchSkew:   10,
		BatchSubmitMaxTime:      submitTimeout,
		BlockBatchMaxSizeBytes:  1000,
		GossipedBlocksCacheSize: 50,
	}

	manager, err := testutil.GetManager(managerConfig, nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	// Check initial height
	initialHeight := uint64(0)
	require.Equal(initialHeight, manager.State.Height())
	require.Zero(manager.SyncTarget.Load())

	var wg sync.WaitGroup
	mCtx, cancel := context.WithTimeout(context.Background(), 2*submitTimeout)
	defer cancel()

	wg.Add(2) // Add 2 because we have 2 goroutines

	go func() {
		manager.ProduceBlockLoop(mCtx)
		wg.Done() // Decrease counter when this goroutine finishes
	}()

	go func() {
		manager.SubmitLoop(mCtx)
		wg.Done() // Decrease counter when this goroutine finishes
	}()

	wg.Wait() // Wait for all goroutines to finish
	require.True(manager.SyncTarget.Load() > 0)
}

// TestSubmissionByBatchSize tests the submission trigger by batch size
func TestSubmissionByBatchSize(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	cases := []struct {
		name                   string
		blockBatchMaxSizeBytes uint64
		expectedSubmission     bool
	}{
		{
			name:                   "block batch max size is fulfilled",
			blockBatchMaxSizeBytes: 2000,
			expectedSubmission:     true,
		},
		{
			name:                   "block batch max size is not fulfilled",
			blockBatchMaxSizeBytes: 100000,
			expectedSubmission:     false,
		},
	}

	for _, c := range cases {
		managerConfig := testutil.GetManagerConfig()
		managerConfig.BlockBatchMaxSizeBytes = c.blockBatchMaxSizeBytes
		manager, err := testutil.GetManager(managerConfig, nil, nil, 1, 1, 0, nil, nil)
		require.NoError(err)

		// validate initial accumulated is zero
		require.Equal(manager.AccumulatedBatchSize.Load(), uint64(0))
		assert.Equal(manager.State.Height(), uint64(0))

		var wg sync.WaitGroup
		wg.Add(2) // Add 2 because we have 2 goroutines

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		go func() {
			manager.ProduceBlockLoop(ctx)
			wg.Done() // Decrease counter when this goroutine finishes
		}()

		go func() {
			manager.SubmitLoop(ctx)
			wg.Done() // Decrease counter when this goroutine finishes
		}()

		// wait for block to be produced but not for submission threshold
		time.Sleep(200 * time.Millisecond)
		// assert block produced but nothing submitted yet
		assert.Greater(manager.State.Height(), uint64(0))
		assert.Greater(manager.AccumulatedBatchSize.Load(), uint64(0))
		assert.Zero(manager.SyncTarget.Load())

		wg.Wait() // Wait for all goroutines to finish

		if c.expectedSubmission {
			assert.Positive(manager.SyncTarget.Load())
		} else {
			assert.Zero(manager.SyncTarget.Load())
		}
	}
}
