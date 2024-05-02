package block_test

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	uevent "github.com/dymensionxyz/dymint/utils/event"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p/core/crypto"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	slregistry "github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/dymensionxyz/dymint/store"
)

func TestInitialState(t *testing.T) {
	var err error
	assert := assert.New(t)
	genesis := testutil.GenerateGenesis(123)
	sampleState := testutil.GenerateState(1, 128)
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	conf := testutil.GetManagerConfig()
	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(t, err)
	proxyApp := testutil.GetABCIProxyAppMock(logger.With("module", "proxy"))
	settlementlc := slregistry.GetClient(slregistry.Local)
	_ = settlementlc.Init(settlement.Config{}, pubsubServer, logger)

	// Init empty store and full store
	emptyStore := store.New(store.NewDefaultInMemoryKVStore())
	fullStore := store.New(store.NewDefaultInMemoryKVStore())
	_, err = fullStore.UpdateState(sampleState, nil)
	require.NoError(t, err)

	// Init p2p client
	privKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	p2pClient, err := p2p.NewClient(config.P2PConfig{
		GossipCacheSize: 50,
		BoostrapTime:    30 * time.Second,
	}, privKey, "TestChain", logger)
	assert.NoError(err)
	assert.NotNil(p2pClient)

	err = p2pClient.Start(context.Background())
	defer func() {
		_ = p2pClient.Close()
	}()
	assert.NoError(err)

	cases := []struct {
		name                    string
		store                   store.Store
		genesis                 *tmtypes.GenesisDoc
		expectedInitialHeight   int64
		expectedLastBlockHeight int64
		expectedChainID         string
	}{
		{
			name:                    "empty store",
			store:                   emptyStore,
			genesis:                 genesis,
			expectedInitialHeight:   genesis.InitialHeight,
			expectedLastBlockHeight: 0,
			expectedChainID:         genesis.ChainID,
		},
		{
			name:                    "state in store",
			store:                   fullStore,
			genesis:                 genesis,
			expectedInitialHeight:   sampleState.InitialHeight,
			expectedLastBlockHeight: sampleState.LastBlockHeight,
			expectedChainID:         sampleState.ChainID,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dalc := testutil.GetMockDALC(logger)
			agg, err := block.NewManager(key, conf, c.genesis, c.store, nil, proxyApp, dalc, settlementlc,
				nil, pubsubServer, p2pClient, logger)
			assert.NoError(err)
			assert.NotNil(agg)
			assert.Equal(c.expectedChainID, agg.LastState.ChainID)
			assert.Equal(c.expectedInitialHeight, agg.LastState.InitialHeight)
			assert.Equal(c.expectedLastBlockHeight, agg.LastState.LastBlockHeight)
		})
	}
}

// TestProduceOnlyAfterSynced should test that we are resuming publishing blocks after we are synced
// 1. Submit a batch and outsync the manager
// 2. Fail to produce blocks
// 2. Sync the manager
// 3. Succeed to produce blocks
func TestProduceOnlyAfterSynced(t *testing.T) {
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, nil, 1, 1, 0, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)

	t.Log("Taking the manager out of sync by submitting a batch")

	syncTarget := manager.SyncTarget.Load()
	numBatchesToAdd := 2
	nextBatchStartHeight := syncTarget + 1
	var batch *types.Batch
	for i := 0; i < numBatchesToAdd; i++ {
		batch, err = testutil.GenerateBatch(nextBatchStartHeight, nextBatchStartHeight+uint64(testutil.DefaultTestBatchSize-1), manager.ProposerKey)
		assert.NoError(t, err)
		daResultSubmitBatch := manager.DAClient.SubmitBatch(batch)
		assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)
		err = manager.SLClient.SubmitBatch(batch, manager.DAClient.GetClientType(), &daResultSubmitBatch)
		require.NoError(t, err)
		nextBatchStartHeight = batch.EndHeight + 1
		// Wait until daHeight is updated
		time.Sleep(time.Millisecond * 500)
	}

	// Initially sync target is 0
	assert.Zero(t, manager.SyncTarget.Load())
	assert.True(t, manager.Store.Height() == 0)

	// enough time to sync and produce blocks
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
	defer cancel()
	// Capture the error returned by manager.Start.
	errChan := make(chan error, 1)
	go func() {
		errChan <- manager.Start(ctx, true)
		err := <-errChan
		// Check for error from manager.Start.
		assert.NoError(t, err, "Manager start should not produce an error")
	}()
	<-ctx.Done()
	assert.Equal(t, batch.EndHeight, manager.SyncTarget.Load())
	// validate that we produced blocks
	assert.Greater(t, manager.Store.Height(), batch.EndHeight)
}

func TestRetrieveDaBatchesFailed(t *testing.T) {
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, &testutil.DALayerClientRetrieveBatchesError{}, 1, 1, 0, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)

	t.Log(manager.LastState.SLStateIndex)
	daMetaData := &da.DASubmitMetaData{
		Client: da.Mock,
		Height: 1,
	}
	err = manager.ProcessNextDABatch(daMetaData)
	t.Log(err)
	assert.ErrorIs(t, err, da.ErrBlobNotFound)
}

func TestProduceNewBlock(t *testing.T) {
	// Init app
	app := testutil.GetAppMock(testutil.Commit)
	commitHash := [32]byte{1}
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(t, err)
	// Init manager
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	// Produce block
	err = manager.ProduceAndGossipBlock(context.Background(), true)
	require.NoError(t, err)
	// Validate state is updated with the commit hash
	assert.Equal(t, uint64(1), manager.Store.Height())
	assert.Equal(t, commitHash, manager.LastState.AppHash)
}

func TestProducePendingBlock(t *testing.T) {
	// Init app
	app := testutil.GetAppMock(testutil.Commit)
	commitHash := [32]byte{1}
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(t, err)
	// Init manager
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	// Generate block and commit and save it to the store
	blocks, err := testutil.GenerateBlocks(1, 1, manager.ProposerKey)
	require.NoError(t, err)
	block := blocks[0]
	_, err = manager.Store.SaveBlock(block, &block.LastCommit, nil)
	require.NoError(t, err)
	// Produce block
	err = manager.ProduceAndGossipBlock(context.Background(), true)
	require.NoError(t, err)
	// Validate state is updated with the block that was saved in the store
	assert.Equal(t, block.Header.Hash(), *(*[32]byte)(manager.LastState.LastBlockID.Hash))
}

// TestBlockProductionNodeHealth tests the different scenarios of block production when the node health is toggling.
// The test does the following:
// 1. Send healthy event and validate blocks are produced
// 2. Send unhealthy event and validate blocks are not produced
// 3. Send another unhealthy event and validate blocks are still not produced
// 4. Send healthy event and validate blocks are produced
func TestBlockProductionNodeHealth(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	// Setup app
	app := testutil.GetAppMock()
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)
	// Init manager
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	cases := []struct {
		name                  string
		healthStatusEvent     map[string][]string
		healthStatusEventData interface{}
		shouldProduceBlocks   bool
	}{
		{
			name:                  "HealthyEventBlocksProduced",
			healthStatusEvent:     events.HealthStatusList,
			healthStatusEventData: &events.DataHealthStatus{},
			shouldProduceBlocks:   true,
		},
		{
			name:                  "UnhealthyEventBlocksNotProduced",
			healthStatusEvent:     events.HealthStatusList,
			healthStatusEventData: &events.DataHealthStatus{Error: errors.New("unhealthy")},
			shouldProduceBlocks:   false,
		},
		{
			name:                  "UnhealthyEventBlocksStillNotProduced",
			healthStatusEvent:     events.HealthStatusList,
			healthStatusEventData: &events.DataHealthStatus{Error: errors.New("unhealthy")},
			shouldProduceBlocks:   false,
		},
		{
			name:                  "HealthyEventBlocksProduced",
			healthStatusEvent:     events.HealthStatusList,
			healthStatusEventData: &events.DataHealthStatus{},
			shouldProduceBlocks:   true,
		},
	}
	// Start the manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = manager.Start(ctx, true)
	require.NoError(err)
	time.Sleep(100 * time.Millisecond)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			uevent.MustPublish(context.Background(), manager.Pubsub, c.healthStatusEventData, c.healthStatusEvent)
			assert.NoError(err, "PublishWithEvents should not produce an error")
			time.Sleep(500 * time.Millisecond)
			blockHeight := manager.Store.Height()
			time.Sleep(500 * time.Millisecond)
			if c.shouldProduceBlocks {
				assert.Greater(manager.Store.Height(), blockHeight)
			} else {
				assert.Equal(blockHeight, manager.Store.Height())
			}
		})
	}
}

// Test that in case we fail after the proxy app commit, next time we won't commit again to the proxy app
// and only update the store height and app hash. This test does the following:
// 1. Produce first block successfully
// 2. Produce second block and fail on update state
// 3. Produce second block again successfully despite the failure
// 4. Produce third block and fail on update store height
// 5. Produce third block successfully
func TestProduceBlockFailAfterCommit(t *testing.T) {
	require := require.New(t)
	// Setup app
	app := testutil.GetAppMock(testutil.Info, testutil.Commit)
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)
	// Create a new mock store which should succeed to save the first block
	mockStore := testutil.NewMockStore()
	// Init manager
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, nil, 1, 1, 0, proxyApp, mockStore)
	require.NoError(err)

	cases := []struct {
		name                   string
		shouldFailSetSetHeight bool
		shouldFailUpdateState  bool
		LastAppBlockHeight     int64
		AppCommitHash          [32]byte
		LastAppCommitHash      [32]byte
		expectedStoreHeight    uint64
		expectedStateAppHash   [32]byte
	}{
		{
			name:                   "ProduceFirstBlockSuccessfully",
			shouldFailSetSetHeight: false,
			shouldFailUpdateState:  false,
			AppCommitHash:          [32]byte{1},
			LastAppCommitHash:      [32]byte{0},
			LastAppBlockHeight:     0,
			expectedStoreHeight:    1,
			expectedStateAppHash:   [32]byte{1},
		},
		{
			name:                   "ProduceSecondBlockFailOnUpdateState",
			shouldFailSetSetHeight: false,
			shouldFailUpdateState:  true,
			AppCommitHash:          [32]byte{2},
			LastAppCommitHash:      [32]byte{},
			LastAppBlockHeight:     0,
			expectedStoreHeight:    1,
			expectedStateAppHash:   [32]byte{1},
		},
		{
			name:                   "ProduceSecondBlockSuccessfully",
			shouldFailSetSetHeight: false,
			shouldFailUpdateState:  false,
			AppCommitHash:          [32]byte{},
			LastAppCommitHash:      [32]byte{2},
			LastAppBlockHeight:     2,
			expectedStoreHeight:    2,
			expectedStateAppHash:   [32]byte{2},
		},
		{
			name:                   "ProduceThirdBlockFailOnUpdateStoreHeight",
			shouldFailSetSetHeight: true,
			shouldFailUpdateState:  false,
			AppCommitHash:          [32]byte{3},
			LastAppCommitHash:      [32]byte{2},
			LastAppBlockHeight:     2,
			expectedStoreHeight:    2,
			expectedStateAppHash:   [32]byte{3},
		},
		{
			name:                   "ProduceThirdBlockSuccessfully",
			shouldFailSetSetHeight: false,
			shouldFailUpdateState:  false,
			AppCommitHash:          [32]byte{},
			LastAppCommitHash:      [32]byte{3},
			LastAppBlockHeight:     3,
			expectedStoreHeight:    3,
			expectedStateAppHash:   [32]byte{3},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: tc.AppCommitHash[:]}).Once()
			app.On("Info", mock.Anything).Return(abci.ResponseInfo{
				LastBlockHeight:  tc.LastAppBlockHeight,
				LastBlockAppHash: tc.LastAppCommitHash[:],
			})
			mockStore.ShouldFailSetHeight = tc.shouldFailSetSetHeight
			mockStore.ShoudFailUpdateState = tc.shouldFailUpdateState
			_ = manager.ProduceAndGossipBlock(context.Background(), true)
			require.Equal(tc.expectedStoreHeight, manager.Store.Height(), tc.name)
			require.Equal(tc.expectedStateAppHash, manager.LastState.AppHash, tc.name)
			storeState, err := manager.Store.LoadState()
			require.NoError(err)
			require.Equal(tc.expectedStateAppHash, storeState.AppHash, tc.name)

			app.On("Commit", mock.Anything).Unset()
			app.On("Info", mock.Anything).Unset()
		})
	}
}

func TestCreateNextDABatchWithBytesLimit(t *testing.T) {
	const batchLimitBytes = 2000

	assert := assert.New(t)
	require := require.New(t)
	app := testutil.GetAppMock()
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)
	// Init manager
	managerConfig := testutil.GetManagerConfig()
	managerConfig.BlockBatchSize = 1000
	managerConfig.BlockBatchMaxSizeBytes = batchLimitBytes // enough for 2 block, not enough for 10 blocks
	manager, err := testutil.GetManager(managerConfig, nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testCases := []struct {
		name                  string
		blocksToProduce       int
		expectedToBeTruncated bool
	}{
		{
			name:                  "batch fits in bytes limit",
			blocksToProduce:       2,
			expectedToBeTruncated: false,
		},
		{
			name:                  "batch exceeds bytes limit",
			blocksToProduce:       10,
			expectedToBeTruncated: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Produce blocks
			for i := 0; i < tc.blocksToProduce; i++ {
				err := manager.ProduceAndGossipBlock(ctx, true)
				assert.NoError(err)
			}

			// Call createNextDABatch function
			startHeight := manager.SyncTarget.Load() + 1
			endHeight := startHeight + uint64(tc.blocksToProduce) - 1
			batch, err := manager.CreateNextDABatch(startHeight, endHeight)
			assert.NoError(err)

			assert.Equal(batch.StartHeight, startHeight)
			assert.LessOrEqual(batch.ToProto().Size(), int(managerConfig.BlockBatchMaxSizeBytes))

			if !tc.expectedToBeTruncated {
				assert.Equal(batch.EndHeight, endHeight)
			} else {
				assert.Equal(batch.EndHeight, batch.StartHeight+uint64(len(batch.Blocks))-1)
				assert.Less(batch.EndHeight, endHeight)

				// validate next added block to batch would have been actually too big
				// First relax the byte limit so we could proudce larger batch
				manager.Conf.BlockBatchMaxSizeBytes = 10 * manager.Conf.BlockBatchMaxSizeBytes
				newBatch, err := manager.CreateNextDABatch(startHeight, batch.EndHeight+1)
				assert.Greater(newBatch.ToProto().Size(), batchLimitBytes)

				assert.NoError(err)
			}
		})
	}
}

func TestDAFetch(t *testing.T) {
	require := require.New(t)
	// Setup app
	app := testutil.GetAppMock(testutil.Info, testutil.Commit)
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)
	// Create a new mock store which should succeed to save the first block
	mockStore := testutil.NewMockStore()
	// Init manager
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, nil, 1, 1, 0, proxyApp, mockStore)
	require.NoError(err)
	commitHash := [32]byte{1}

	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]})

	syncTarget := manager.SyncTarget.Load()
	nextBatchStartHeight := syncTarget + 1
	batch, err := testutil.GenerateBatch(nextBatchStartHeight, nextBatchStartHeight+uint64(testutil.DefaultTestBatchSize-1), manager.ProposerKey)
	require.NoError(err)
	daResultSubmitBatch := manager.DAClient.SubmitBatch(batch)
	require.Equal(daResultSubmitBatch.Code, da.StatusSuccess)
	err = manager.SLClient.SubmitBatch(batch, manager.DAClient.GetClientType(), &daResultSubmitBatch)
	require.NoError(err)

	cases := []struct {
		name       string
		manager    *block.Manager
		daMetaData *da.DASubmitMetaData
		batch      *types.Batch
		err        error
	}{
		{
			name:       "valid DA",
			manager:    manager,
			daMetaData: daResultSubmitBatch.SubmitMetaData,
			batch:      batch,
			err:        nil,
		},
		{
			name:       "wrong DA",
			manager:    manager,
			daMetaData: &da.DASubmitMetaData{Client: da.Celestia, Height: daResultSubmitBatch.SubmitMetaData.Height},
			batch:      batch,
			err:        block.ErrWrongDA,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]}).Once()
			app.On("Info", mock.Anything).Return(abci.ResponseInfo{
				LastBlockHeight:  int64(batch.EndHeight),
				LastBlockAppHash: commitHash[:],
			})
			err := manager.ProcessNextDABatch(c.daMetaData)
			require.Equal(c.err, err)
		})
	}
}
