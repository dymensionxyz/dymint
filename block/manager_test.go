package block

import (
	"context"
	"crypto/rand"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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

const (
	connectionRefusedErrorMessage = "connection refused"
	batchNotFoundErrorMessage     = "batch not found"
)

func TestInitialState(t *testing.T) {
	assert := assert.New(t)
	genesis := testutil.GenerateGenesis(123)
	sampleState := testutil.GenerateState(1, 128)
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	conf := getManagerConfig()
	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	proxyApp := testutil.GetABCIProxyAppMock(logger.With("module", "proxy"))
	settlementlc := slregistry.GetClient(slregistry.Mock)
	_ = settlementlc.Init(settlement.Config{}, pubsubServer, logger)

	// Init empty store and full store
	emptyStore := store.New(store.NewDefaultInMemoryKVStore())
	fullStore := store.New(store.NewDefaultInMemoryKVStore())
	_, err := fullStore.UpdateState(sampleState, nil)
	require.NoError(t, err)

	// Init p2p client
	privKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	p2pClient, err := p2p.NewClient(config.P2PConfig{}, privKey, "TestChain", logger)
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

			dalc := getMockDALC(logger)
			agg, err := NewManager(key, conf, c.genesis, c.store, nil, proxyApp, dalc, settlementlc,
				nil, pubsubServer, p2pClient, logger)
			assert.NoError(err)
			assert.NotNil(agg)
			assert.Equal(c.expectedChainID, agg.lastState.ChainID)
			assert.Equal(c.expectedInitialHeight, agg.lastState.InitialHeight)
			assert.Equal(c.expectedLastBlockHeight, agg.lastState.LastBlockHeight)
		})
	}
}

// TestProduceOnlyAfterSynced should test that we are resuming publishing blocks after we are synced
// 1. Submit a batch and outsync the manager
// 2. Fail to produce blocks
// 2. Sync the manager
// 3. Succeed to produce blocks
func TestProduceOnlyAfterSynced(t *testing.T) {
	manager, err := getManager(getManagerConfig(), nil, nil, 1, 1, 0, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)

	t.Log("Taking the manager out of sync by submitting a batch")
	lastStoreHeight := manager.store.Height()
	numBatchesToAdd := 2
	nextBatchStartHeight := atomic.LoadUint64(&manager.syncTarget) + 1
	var batch *types.Batch
	for i := 0; i < numBatchesToAdd; i++ {
		batch, err = testutil.GenerateBatch(nextBatchStartHeight, nextBatchStartHeight+uint64(defaultBatchSize-1), manager.proposerKey)
		assert.NoError(t, err)
		daResultSubmitBatch := manager.dalc.SubmitBatch(batch)
		assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)
		manager.settlementClient.SubmitBatch(batch, manager.dalc.GetClientType(), &daResultSubmitBatch)
		nextBatchStartHeight = batch.EndHeight + 1
		// Wait until daHeight is updated
		time.Sleep(time.Millisecond * 500)
	}

	t.Log("Validating manager can't produce blocks")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	go manager.ProduceBlockLoop(ctx)
	<-ctx.Done()
	assert.Equal(t, lastStoreHeight, manager.store.Height())
	// Wait until produce block loop is done
	time.Sleep(time.Second * 1)

	t.Log("Sync the manager")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	go manager.Start(ctx, false)
	<-ctx.Done()
	require.Greater(t, manager.store.Height(), lastStoreHeight)
	assert.Equal(t, batch.EndHeight, manager.store.Height())
	// Wait until manager is done
	time.Sleep(time.Second * 4)

	t.Log("Validate blocks are produced")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	go manager.ProduceBlockLoop(ctx)
	<-ctx.Done()
	assert.Greater(t, manager.store.Height(), batch.EndHeight)
}

func TestRetrieveDaBatchesFailed(t *testing.T) {
	manager, err := getManager(getManagerConfig(), nil, &testutil.DALayerClientRetrieveBatchesError{}, 1, 1, 0, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)

	err = manager.processNextDABatch(context.Background(), 1)
	assert.ErrorContains(t, err, batchNotFoundErrorMessage)
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
	manager, err := getManager(getManagerConfig(), nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	// Produce block
	err = manager.produceBlock(context.Background(), true)
	require.NoError(t, err)
	// Validate state is updated with the commit hash
	assert.Equal(t, uint64(1), manager.store.Height())
	assert.Equal(t, commitHash, manager.lastState.AppHash)
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
	manager, err := getManager(getManagerConfig(), nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	// Generate block and commit and save it to the store
	blocks, err := testutil.GenerateBlocks(1, 1, manager.proposerKey)
	require.NoError(t, err)
	block := blocks[0]
	_, err = manager.store.SaveBlock(block, &block.LastCommit, nil)
	require.NoError(t, err)
	// Produce block
	err = manager.produceBlock(context.Background(), true)
	require.NoError(t, err)
	// Validate state is updated with the block that was saved in the store
	assert.Equal(t, block.Header.Hash(), *(*[32]byte)(manager.lastState.LastBlockID.Hash))
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
	manager, err := getManager(getManagerConfig(), nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	cases := []struct {
		name                  string
		healthStatusEvent     map[string][]string
		healthStatusEventData interface{}
		shouldProduceBlocks   bool
	}{
		{
			name:                  "HealthyEventBlocksProduced",
			healthStatusEvent:     map[string][]string{events.EventNodeTypeKey: {events.EventHealthStatus}},
			healthStatusEventData: &events.EventDataHealthStatus{Healthy: true, Error: nil},
			shouldProduceBlocks:   true,
		},
		{
			name:                  "UnhealthyEventBlocksNotProduced",
			healthStatusEvent:     map[string][]string{events.EventNodeTypeKey: {events.EventHealthStatus}},
			healthStatusEventData: &events.EventDataHealthStatus{Healthy: false, Error: errors.New("Unhealthy")},
			shouldProduceBlocks:   false,
		},
		{
			name:                  "UnhealthyEventBlocksStillNotProduced",
			healthStatusEvent:     map[string][]string{events.EventNodeTypeKey: {events.EventHealthStatus}},
			healthStatusEventData: &events.EventDataHealthStatus{Healthy: false, Error: errors.New("Unhealthy")},
			shouldProduceBlocks:   false,
		},
		{
			name:                  "HealthyEventBlocksProduced",
			healthStatusEvent:     map[string][]string{events.EventNodeTypeKey: {events.EventHealthStatus}},
			healthStatusEventData: &events.EventDataHealthStatus{Healthy: true, Error: nil},
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
			manager.pubsub.PublishWithEvents(context.Background(), c.healthStatusEventData, c.healthStatusEvent)
			time.Sleep(500 * time.Millisecond)
			blockHeight := manager.store.Height()
			time.Sleep(500 * time.Millisecond)
			if c.shouldProduceBlocks {
				assert.Greater(manager.store.Height(), blockHeight)
			} else {
				assert.Equal(blockHeight, manager.store.Height())
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
	assert := assert.New(t)
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
	manager, err := getManager(getManagerConfig(), nil, nil, 1, 1, 0, proxyApp, mockStore)
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
			app.On("Info", mock.Anything).Return(abci.ResponseInfo{LastBlockHeight: tc.LastAppBlockHeight,
				LastBlockAppHash: tc.LastAppCommitHash[:]}).Once()
			mockStore.ShouldFailSetHeight = tc.shouldFailSetSetHeight
			mockStore.ShoudFailUpdateState = tc.shouldFailUpdateState
			_ = manager.produceBlock(context.Background(), true)
			assert.Equal(tc.expectedStoreHeight, manager.store.Height())
			assert.Equal(tc.expectedStateAppHash, manager.lastState.AppHash)
			storeState, err := manager.store.LoadState()
			require.NoError(err)
			assert.Equal(tc.expectedStateAppHash, storeState.AppHash)
		})
	}
}

func TestCreateNextDABatchWithBytesLimit(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	app := testutil.GetAppMock()
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)
	// Init manager
	managerConfig := getManagerConfig()
	managerConfig.BlockBatchSize = 1000
	managerConfig.BlockBatchMaxSizeBytes = batchLimitBytes //enough for 2 block, not enough for 10 blocks
	manager, err := getManager(managerConfig, nil, nil, 1, 1, 0, proxyApp, nil)
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
				manager.produceBlock(ctx, true)
			}

			// Call createNextDABatch function
			startHeight := atomic.LoadUint64(&manager.syncTarget) + 1
			endHeight := startHeight + uint64(tc.blocksToProduce) - 1
			batch, err := manager.createNextDABatch(startHeight, endHeight)
			assert.NoError(err)

			assert.Equal(batch.StartHeight, startHeight)
			assert.LessOrEqual(batch.ToProto().Size(), int(managerConfig.BlockBatchMaxSizeBytes))

			if !tc.expectedToBeTruncated {
				assert.Equal(batch.EndHeight, endHeight)
			} else {
				assert.Equal(batch.EndHeight, batch.StartHeight+uint64(len(batch.Blocks))-1)
				assert.Less(batch.EndHeight, endHeight)

				//validate next added block to batch would have been actually too big
				//First relax the byte limit so we could proudce larger batch
				manager.conf.BlockBatchMaxSizeBytes = 10 * manager.conf.BlockBatchMaxSizeBytes
				newBatch, err := manager.createNextDABatch(startHeight, batch.EndHeight+1)
				assert.Greater(newBatch.ToProto().Size(), batchLimitBytes)

				assert.NoError(err)
			}
		})
	}
}
