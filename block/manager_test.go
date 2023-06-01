package block

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/avast/retry-go"

	"github.com/dymensionxyz/dymint/log/test"
	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	mockda "github.com/dymensionxyz/dymint/da/mock"
	nodemempool "github.com/dymensionxyz/dymint/node/mempool"
	slregistry "github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/dymensionxyz/dymint/store"
)

const (
	defaultBatchSize = 5
	batchLimitBytes  = 2000
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
	p2pClient, err := p2p.NewClient(config.P2PConfig{}, privKey, "TestChain", test.NewLogger(t))
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

			dalc := getMockDALC(100*time.Second, logger)
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
		resultSubmitBatch := manager.settlementClient.SubmitBatch(batch, manager.dalc.GetClientType(), &daResultSubmitBatch)
		assert.Equal(t, resultSubmitBatch.Code, settlement.StatusSuccess)
		nextBatchStartHeight = batch.EndHeight + 1
		// Wait until daHeight is updated
		time.Sleep(time.Millisecond * 500)
	}

	t.Log("Validating manager can't produce blocks")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	go manager.ProduceBlockLoop(ctx)
	select {
	case <-ctx.Done():
		assert.Equal(t, lastStoreHeight, manager.store.Height())
	}

	t.Log("Sync the manager")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	go manager.Start(ctx, false)
	select {
	case <-ctx.Done():
		assert.Greater(t, manager.store.Height(), lastStoreHeight)
		assert.Equal(t, batch.EndHeight, manager.store.Height())
	}

	t.Log("Validate blocks are produced")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	go manager.ProduceBlockLoop(ctx)
	select {
	case <-ctx.Done():
		assert.Greater(t, manager.store.Height(), batch.EndHeight)
	}
}

func TestPublishWhenSettlementLayerDisconnected(t *testing.T) {
	SLBatchRetryDelay = 1 * time.Second

	isSettlementError := atomic.Value{}
	isSettlementError.Store(true)
	manager, err := getManager(getManagerConfig(), &testutil.SettlementLayerClientSubmitBatchError{IsError: isSettlementError}, nil, 1, 1, 0, nil, nil)
	retry.DefaultAttempts = 2
	require.NoError(t, err)
	require.NotNil(t, manager)

	nextBatchStartHeight := atomic.LoadUint64(&manager.syncTarget) + 1
	batch, err := testutil.GenerateBatch(nextBatchStartHeight, nextBatchStartHeight+uint64(defaultBatchSize-1), manager.proposerKey)
	assert.NoError(t, err)
	daResultSubmitBatch := manager.dalc.SubmitBatch(batch)
	assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)
	resultSubmitBatch := manager.settlementClient.SubmitBatch(batch, manager.dalc.GetClientType(), &daResultSubmitBatch)
	assert.Equal(t, resultSubmitBatch.Code, settlement.StatusError)

	defer func() {
		err := recover().(error)
		assert.ErrorContains(t, err, connectionRefusedErrorMessage)
	}()
	manager.submitBatchToSL(&types.Batch{StartHeight: 1, EndHeight: 1}, nil)
}

func TestPublishWhenDALayerDisconnected(t *testing.T) {
	DABatchRetryDelay = 1 * time.Second
	manager, err := getManager(getManagerConfig(), nil, &testutil.DALayerClientSubmitBatchError{}, 1, 1, 0, nil, nil)
	retry.DefaultAttempts = 2
	require.NoError(t, err)
	require.NotNil(t, manager)

	nextBatchStartHeight := atomic.LoadUint64(&manager.syncTarget) + 1
	batch, err := testutil.GenerateBatch(nextBatchStartHeight, nextBatchStartHeight+uint64(defaultBatchSize-1), manager.proposerKey)
	assert.NoError(t, err)
	daResultSubmitBatch := manager.dalc.SubmitBatch(batch)
	assert.Equal(t, daResultSubmitBatch.Code, da.StatusError)

	_, err = manager.submitBatchToDA(context.Background(), batch)
	assert.ErrorContains(t, err, connectionRefusedErrorMessage)
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
	err = manager.produceBlock(context.Background())
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
	err = manager.produceBlock(context.Background())
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
			_ = manager.produceBlock(context.Background())
			assert.Equal(tc.expectedStoreHeight, manager.store.Height())
			assert.Equal(tc.expectedStateAppHash, manager.lastState.AppHash)
			storeState, err := manager.store.LoadState()
			require.NoError(err)
			assert.Equal(tc.expectedStateAppHash, storeState.AppHash)
		})
	}
}

// Test batch retry halts upon new batch acceptance
// 1. Produce blocks with settlement layer batch submission error
// 2. Emit an event that a new batch was accepted
// 3. Validate new batches was submitted
func TestBatchRetryWhileBatchAccepted(t *testing.T) {
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
	managerConfig.BlockBatchSize = 1
	isSettlementError := atomic.Value{}
	isSettlementError.Store(true)
	settlementLayerClient := &testutil.SettlementLayerClientSubmitBatchError{IsError: isSettlementError}
	manager, err := getManager(managerConfig, settlementLayerClient, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)
	// Produce blocks with settlement layer batch submission error
	blockLoopContext, blockLoopCancel := context.WithCancel(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	defer blockLoopCancel()
	defer cancel()
	go manager.ProduceBlockLoop(blockLoopContext)
	go manager.SyncTargetLoop(ctx)
	time.Sleep(200 * time.Millisecond)
	assert.Equal(uint64(0), atomic.LoadUint64(&settlementLayerClient.BatchCounter))
	// Cancel block production to not interfere with the isBatchInProcess flag
	blockLoopCancel()
	time.Sleep(100 * time.Millisecond)
	// Emit an event that a new batch was accepted and wait for it to be processed
	eventData := &settlement.EventDataNewSettlementBatchAccepted{EndHeight: 1, StateIndex: 1}
	manager.pubsub.PublishWithEvents(ctx, eventData, map[string][]string{settlement.EventTypeKey: {settlement.EventNewSettlementBatchAccepted}})
	time.Sleep(200 * time.Millisecond)
	// Change settlement layer to accept batches and validate new batches was submitted
	settlementLayerClient.IsError.Store(false)
	blockLoopContext, blockLoopCancel = context.WithCancel(context.Background())
	defer blockLoopCancel()
	go manager.ProduceBlockLoop(blockLoopContext)
	time.Sleep(1 * time.Second)
	assert.Greater(atomic.LoadUint64(&settlementLayerClient.BatchCounter), uint64(0))

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
	managerConfig.BlockBatchSizeBytes = batchLimitBytes //enough for 2 block, not enough for 10 blocks
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
				manager.produceBlock(ctx)
			}

			// Call createNextDABatch function
			startHeight := atomic.LoadUint64(&manager.syncTarget) + 1
			endHeight := startHeight + uint64(tc.blocksToProduce) - 1
			batch, err := manager.createNextDABatch(startHeight, endHeight)
			assert.NoError(err)

			assert.Equal(batch.StartHeight, startHeight)
			assert.LessOrEqual(batch.ToProto().Size(), int(managerConfig.BlockBatchSizeBytes))

			if !tc.expectedToBeTruncated {
				assert.Equal(batch.EndHeight, endHeight)
			} else {
				assert.Equal(batch.EndHeight, batch.StartHeight+uint64(len(batch.Blocks))-1)
				assert.Less(batch.EndHeight, endHeight)

				//validate next added block to batch would have been actually too big
				//First relax the byte limit so we could proudce larger batch
				manager.conf.BlockBatchSizeBytes = 10 * manager.conf.BlockBatchSizeBytes
				newBatch, err := manager.createNextDABatch(startHeight, batch.EndHeight+1)
				assert.Greater(newBatch.ToProto().Size(), batchLimitBytes)

				assert.NoError(err)
			}
		})
	}
}

/* -------------------------------------------------------------------------- */
/*                                    utils                                   */
/* -------------------------------------------------------------------------- */

func getManager(conf config.BlockManagerConfig, settlementlc settlement.LayerI, dalc da.DataAvailabilityLayerClient, genesisHeight int64, storeInitialHeight int64, storeLastBlockHeight int64, proxyAppConns proxy.AppConns, mockStore store.Store) (*Manager, error) {
	genesis := testutil.GenerateGenesis(genesisHeight)
	// Change the LastBlockHeight to avoid calling InitChainSync within the manager
	// And updating the state according to the genesis.
	state := testutil.GenerateState(storeInitialHeight, storeLastBlockHeight)
	var managerStore store.Store
	if mockStore == nil {
		managerStore = store.New(store.NewDefaultInMemoryKVStore())
	} else {
		managerStore = mockStore
	}
	if _, err := managerStore.UpdateState(state, nil); err != nil {
		return nil, err
	}

	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()

	// Init the settlement layer mock
	if settlementlc == nil {
		settlementlc = slregistry.GetClient(slregistry.Mock)
	}
	//TODO(omritoptix): Change the initialization. a bit dirty.
	proposerKey, proposerPubKey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	pubKeybytes, err := proposerPubKey.Raw()
	if err != nil {
		return nil, err
	}

	err = initSettlementLayerMock(settlementlc, hex.EncodeToString(pubKeybytes), pubsubServer, logger)
	if err != nil {
		return nil, err
	}

	if dalc == nil {
		dalc = &mockda.DataAvailabilityLayerClient{}
	}
	initDALCMock(dalc, conf.DABlockTime, logger)

	var proxyApp proxy.AppConns
	if proxyAppConns == nil {
		proxyApp = testutil.GetABCIProxyAppMock(logger.With("module", "proxy"))
		if err := proxyApp.Start(); err != nil {
			return nil, err
		}
	} else {
		proxyApp = proxyAppConns
	}

	mp := mempoolv1.NewTxMempool(logger, tmcfg.DefaultMempoolConfig(), proxyApp.Mempool(), 0)
	mpIDs := nodemempool.NewMempoolIDs()

	// Init p2p client and validator
	p2pKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	p2pClient, err := p2p.NewClient(config.P2PConfig{}, p2pKey, "TestChain", logger)
	if err != nil {
		return nil, err
	}
	p2pValidator := p2p.NewValidator(logger, pubsubServer)
	p2pClient.SetTxValidator(p2pValidator.TxValidator(mp, mpIDs))
	p2pClient.SetBlockValidator(p2pValidator.BlockValidator())

	if err = p2pClient.Start(context.Background()); err != nil {
		return nil, err
	}

	manager, err := NewManager(proposerKey, conf, genesis, managerStore, mp, proxyApp, dalc, settlementlc, nil,
		pubsubServer, p2pClient, logger)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

// TODO(omritoptix): Possible move out to a generic testutil
func getMockDALC(daBlockTime time.Duration, logger log.Logger) da.DataAvailabilityLayerClient {
	dalc := &mockda.DataAvailabilityLayerClient{}
	initDALCMock(dalc, daBlockTime, logger)
	return dalc
}

// TODO(omritoptix): Possible move out to a generic testutil
func initDALCMock(dalc da.DataAvailabilityLayerClient, daBlockTime time.Duration, logger log.Logger) {
	_ = dalc.Init([]byte(daBlockTime.String()), store.NewDefaultInMemoryKVStore(), logger)
	_ = dalc.Start()
}

// TODO(omritoptix): Possible move out to a generic testutil
func initSettlementLayerMock(settlementlc settlement.LayerI, proposer string, pubsubServer *pubsub.Server, logger log.Logger) error {
	err := settlementlc.Init(settlement.Config{ProposerPubKey: proposer}, pubsubServer, logger)
	if err != nil {
		return err
	}
	err = settlementlc.Start()
	if err != nil {
		return err
	}
	return nil
}

func getManagerConfig() config.BlockManagerConfig {
	return config.BlockManagerConfig{
		BlockTime:         100 * time.Millisecond,
		DABlockTime:       100 * time.Millisecond,
		BatchSyncInterval: 1 * time.Second,
		BlockBatchSize:    defaultBatchSize,
		DAStartHeight:     0,
		NamespaceID:       "0102030405060708",
	}
}
