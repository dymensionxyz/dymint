package block

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	mempoolv1 "github.com/celestiaorg/optimint/mempool/v1"
	"github.com/celestiaorg/optimint/settlement"
	"github.com/celestiaorg/optimint/testutil"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/celestiaorg/optimint/config"
	"github.com/celestiaorg/optimint/da"
	mockda "github.com/celestiaorg/optimint/da/mock"
	slmock "github.com/celestiaorg/optimint/settlement/mock"
	slregistry "github.com/celestiaorg/optimint/settlement/registry"
	"github.com/celestiaorg/optimint/store"
)

const defaultBatchSize = 5

func TestInitialState(t *testing.T) {
	genesis := testutil.GenerateGenesis(123)
	sampleState := testutil.GenerateState(1, 128)
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	conf := getManagerConfig()
	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	settlementlc := slregistry.GetClient(slregistry.ClientMock)
	_ = settlementlc.Init(nil, pubsubServer, logger)

	// Init empty store and full store
	emptyStore := store.New(store.NewDefaultInMemoryKVStore())
	fullStore := store.New(store.NewDefaultInMemoryKVStore())
	err := fullStore.UpdateState(sampleState)
	require.NoError(t, err)

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
			assert := assert.New(t)
			dalc := getMockDALC(logger)
			agg, err := NewManager(key, conf, c.genesis, c.store, nil, nil, dalc, settlementlc, nil, pubsubServer, logger)
			assert.NoError(err)
			assert.NotNil(agg)
			assert.Equal(c.expectedChainID, agg.lastState.ChainID)
			assert.Equal(c.expectedInitialHeight, agg.lastState.InitialHeight)
			assert.Equal(c.expectedLastBlockHeight, agg.lastState.LastBlockHeight)
		})
	}
}

// TestWaitUntilSynced tests that we don't start producing blocks until we're synced.
// 1. Validate blocks are produced.
// 2. Add a batch which takes the manager out of sync
// 3. Validate blocks are not produced.
func TestWaitUntilSynced(t *testing.T) {
	storeLastBlockHeight := uint64(0)
	manager, err := getManager(1, 1, int64(storeLastBlockHeight))
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Manager should produce blocks as it's the first to write batches.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	go manager.PublishBlockLoop(ctx)
	select {
	case <-ctx.Done():
		// Validate some blocks produced
		assert.Greater(t, manager.store.Height(), storeLastBlockHeight)
	}
	// Add a batch which takes the manager out of sync
	startHeight := atomic.LoadUint64(&manager.syncTarget) + 1
	batch := testutil.GenerateBatch(startHeight, startHeight+uint64(defaultBatchSize-1))
	daResult := &da.ResultSubmitBatch{
		BaseResult: da.BaseResult{
			DAHeight: 1,
		},
	}
	resultSubmitBatch := manager.settlementlc.SubmitBatch(batch, daResult)
	assert.Equal(t, resultSubmitBatch.Code, settlement.StatusSuccess)

	// Validate blocks are not produced.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	storeHeight := manager.store.Height()
	go manager.PublishBlockLoop(ctx)
	select {
	case <-ctx.Done():
		assert.Equal(t, manager.store.Height(), storeHeight)
	}
}

// TestPublishAfterSynced should test that we are resuming publishing blocks after we are synced
// 1. Validate blocks are not produced by adding a batch and outsyncing the manager
// 2. Sync the manager
// 3. Validate blocks are produced.
func TestPublishAfterSynced(t *testing.T) {
	manager, err := getManager(1, 1, 0)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Validate blocks are not produced by adding a batch and outsyncing the manager.
	// Submit batch
	lastStoreHeight := manager.store.Height()
	nextBatchStartHeight := atomic.LoadUint64(&manager.syncTarget) + 1
	batch := testutil.GenerateBatch(nextBatchStartHeight, nextBatchStartHeight+uint64(defaultBatchSize-1))
	daResultSubmitBatch := manager.dalc.SubmitBatch(batch)
	assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)
	resultSubmitBatch := manager.settlementlc.SubmitBatch(batch, &daResultSubmitBatch)
	assert.Equal(t, resultSubmitBatch.Code, settlement.StatusSuccess)

	// Check manager is out of sync
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	go manager.PublishBlockLoop(ctx)
	select {
	case <-ctx.Done():
		assert.Equal(t, manager.store.Height(), lastStoreHeight)
	}

	// Sync the manager
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	go manager.SyncTargetLoop(ctx)
	go manager.RetriveLoop(ctx)
	go manager.ApplyBlockLoop(ctx)
	select {
	case <-ctx.Done():
		assert.Greater(t, manager.store.Height(), lastStoreHeight)
		assert.Equal(t, manager.store.Height(), batch.EndHeight)
	}

	// Validate blocks are produced
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	go manager.PublishBlockLoop(ctx)
	select {
	case <-ctx.Done():
		assert.Greater(t, manager.store.Height(), batch.EndHeight)
	}

}

func getManager(genesisHeight int64, storeInitialHeight int64, storeLastBlockHeight int64) (*Manager, error) {
	genesis := testutil.GenerateGenesis(genesisHeight)
	// Change the LastBlockHeight to avoid calling InitChainSync within the manager
	// And updating the state according to the genesis.
	state := testutil.GenerateState(storeInitialHeight, storeLastBlockHeight)
	store := store.New(store.NewDefaultInMemoryKVStore())
	if err := store.UpdateState(state); err != nil {
		return nil, err
	}
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	conf := getManagerConfig()
	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	settlementlc := getSettlementLayerMock(defaultBatchSize, uint64(state.LastBlockHeight), uint64(state.LastBlockHeight)+1, pubsubServer, logger)
	proxyApp := testutil.GetABCIProxyAppMock(logger.With("module", "proxy"))
	if err := proxyApp.Start(); err != nil {
		return nil, err
	}
	mp := mempoolv1.NewTxMempool(logger, tmcfg.DefaultMempoolConfig(), proxyApp.Mempool(), 0)
	manager, err := NewManager(key, conf, genesis, store, mp, proxyApp.Consensus(), getMockDALC(logger), settlementlc, nil, pubsubServer, logger)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

// TODO(omritoptix): Possible move out to a generic testutil
func getMockDALC(logger log.Logger) da.DataAvailabilityLayerClient {
	dalc := &mockda.DataAvailabilityLayerClient{}
	_ = dalc.Init(nil, store.NewDefaultInMemoryKVStore(), logger)
	_ = dalc.Start()
	return dalc
}

// TODO(omritoptix): Possible move out to a generic testutil
func getSettlementLayerMock(batchSize uint64, latestHeight uint64, batchOffsetHeight uint64, pubsubServer *pubsub.Server, logger log.Logger) settlement.LayerClient {
	conf := slmock.Config{
		AutoUpdateBatches: false,
		BatchSize:         batchSize,
		LatestHeight:      latestHeight,
		BatchOffsetHeight: batchOffsetHeight,
	}
	byteconf, _ := json.Marshal(conf)
	settlementlc := slregistry.GetClient(slregistry.ClientMock)
	_ = settlementlc.Init(byteconf, pubsubServer, logger)
	return settlementlc
}

func getManagerConfig() config.BlockManagerConfig {
	return config.BlockManagerConfig{
		BlockTime:         10 * time.Second,
		BatchSyncInterval: 1 * time.Second,
		BlockBatchSize:    defaultBatchSize,
		DAStartHeight:     0,
		NamespaceID:       [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
}
