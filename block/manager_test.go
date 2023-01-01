package block

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/avast/retry-go"

	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	mockda "github.com/dymensionxyz/dymint/da/mock"
	slmock "github.com/dymensionxyz/dymint/settlement/mock"
	slregistry "github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/dymensionxyz/dymint/store"
)

const defaultBatchSize = 5
const connectionRefusedErrorMessage = "connection refused"
const batchNotFoundErrorMessage = "batch not found"

func TestInitialState(t *testing.T) {
	genesis := testutil.GenerateGenesis(123)
	sampleState := testutil.GenerateState(1, 128)
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	conf := getManagerConfig()
	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	proxyApp := testutil.GetABCIProxyAppMock(logger.With("module", "proxy"))
	settlementlc := slregistry.GetClient(slregistry.Mock)
	_ = settlementlc.Init(nil, pubsubServer, logger)

	// Init empty store and full store
	emptyStore := store.New(store.NewDefaultInMemoryKVStore())
	fullStore := store.New(store.NewDefaultInMemoryKVStore())
	_, err := fullStore.UpdateState(sampleState, nil)
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

			dalc := getMockDALC(100*time.Second, logger)
			agg, err := NewManager(key, conf, c.genesis, c.store, nil, proxyApp, dalc, settlementlc, nil, pubsubServer, logger)
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
	manager, err := getManager(nil, nil, 1, 1, int64(storeLastBlockHeight))
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Manager should produce blocks as it's the first to write batches.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	// Run syncTargetLoop so that we update the syncTarget.
	go manager.SyncTargetLoop(ctx)
	go manager.PublishBlockLoop(ctx)
	select {
	case <-ctx.Done():
		// Validate some blocks produced
		assert.Greater(t, manager.store.Height(), storeLastBlockHeight)
	}
	// As the publishBlock function doesn't stop upon context termination (only PublishBlockLoop),
	// wait for it to finish before taking the manager out of sync.
	time.Sleep(time.Second)

	// Take the manager out of sync.
	t.Log("Taking the manager out of sync by submitting a batch")
	startHeight := atomic.LoadUint64(&manager.syncTarget) + 1
	endHeight := startHeight + uint64(defaultBatchSize-1)*2
	batch := testutil.GenerateBatch(startHeight, endHeight)
	daResult := &da.ResultSubmitBatch{
		BaseResult: da.BaseResult{
			DAHeight: 1,
		},
	}
	resultSubmitBatch := manager.settlementClient.SubmitBatch(batch, manager.dalc.GetClientType(), daResult)
	assert.Equal(t, resultSubmitBatch.Code, settlement.StatusSuccess)

	// Validate blocks are not produced.
	t.Log("Validating blocks are not produced")
	storeHeight := manager.store.Height()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	go manager.PublishBlockLoop(ctx)
	select {
	case <-ctx.Done():
		assert.Equal(t, storeHeight, manager.store.Height())
	}
}

// TestPublishAfterSynced should test that we are resuming publishing blocks after we are synced
// 1. Validate blocks are not produced by adding a batch and outsyncing the manager
// 2. Sync the manager
// 3. Validate blocks are produced.
func TestPublishAfterSynced(t *testing.T) {
	manager, err := getManager(nil, nil, 1, 1, 0)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Validate blocks are not produced by adding a batch and outsyncing the manager.
	// Submit batch
	lastStoreHeight := manager.store.Height()
	numBatchesToAdd := 2
	nextBatchStartHeight := atomic.LoadUint64(&manager.syncTarget) + 1
	var batch *types.Batch
	for i := 0; i < numBatchesToAdd; i++ {
		batch = testutil.GenerateBatch(nextBatchStartHeight, nextBatchStartHeight+uint64(defaultBatchSize-1))
		daResultSubmitBatch := manager.dalc.SubmitBatch(batch)
		assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)
		resultSubmitBatch := manager.settlementClient.SubmitBatch(batch, manager.dalc.GetClientType(), &daResultSubmitBatch)
		assert.Equal(t, resultSubmitBatch.Code, settlement.StatusSuccess)
		nextBatchStartHeight = batch.EndHeight + 1
		// Wait until daHeight is updated
		time.Sleep(time.Millisecond * 500)
	}

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

func TestPublishWhenSettlementLayerDisconnected(t *testing.T) {
	manager, err := getManager(&SettlementLayerClientSubmitBatchError{}, nil, 1, 1, 0)
	retry.DefaultAttempts = 2
	require.NoError(t, err)
	require.NotNil(t, manager)

	nextBatchStartHeight := atomic.LoadUint64(&manager.syncTarget) + 1
	var batch = testutil.GenerateBatch(nextBatchStartHeight, nextBatchStartHeight+uint64(defaultBatchSize-1))
	daResultSubmitBatch := manager.dalc.SubmitBatch(batch)
	assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)
	resultSubmitBatch := manager.settlementClient.SubmitBatch(batch, manager.dalc.GetClientType(), &daResultSubmitBatch)
	assert.Equal(t, resultSubmitBatch.Code, settlement.StatusError)

	defer func() {
		err := recover().(error)
		assert.ErrorContains(t, err, connectionRefusedErrorMessage)
	}()
	manager.submitBatchToSL(context.Background(), nil, nil)
}

func TestPublishWhenDALayerDisconnected(t *testing.T) {
	manager, err := getManager(nil, &DALayerClientSubmitBatchError{}, 1, 1, 0)
	retry.DefaultAttempts = 2
	require.NoError(t, err)
	require.NotNil(t, manager)

	nextBatchStartHeight := atomic.LoadUint64(&manager.syncTarget) + 1
	var batch = testutil.GenerateBatch(nextBatchStartHeight, nextBatchStartHeight+uint64(defaultBatchSize-1))
	daResultSubmitBatch := manager.dalc.SubmitBatch(batch)
	assert.Equal(t, daResultSubmitBatch.Code, da.StatusError)

	_, err = manager.submitBatchToDA(context.Background(), nil)
	assert.ErrorContains(t, err, connectionRefusedErrorMessage)
}

func TestRetrieveDaBatchesFailed(t *testing.T) {
	manager, err := getManager(nil, &DALayerClientRetrieveBatchesError{}, 1, 1, 0)
	require.NoError(t, err)
	require.NotNil(t, manager)

	err = manager.processNextDABatch(context.Background(), 1)
	assert.ErrorContains(t, err, batchNotFoundErrorMessage)
}

func getManager(settlementlc settlement.LayerClient, dalc da.DataAvailabilityLayerClient, genesisHeight int64, storeInitialHeight int64, storeLastBlockHeight int64) (*Manager, error) {
	genesis := testutil.GenerateGenesis(genesisHeight)
	// Change the LastBlockHeight to avoid calling InitChainSync within the manager
	// And updating the state according to the genesis.
	state := testutil.GenerateState(storeInitialHeight, storeLastBlockHeight)
	store := store.New(store.NewDefaultInMemoryKVStore())
	if _, err := store.UpdateState(state, nil); err != nil {
		return nil, err
	}
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	conf := getManagerConfig()
	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()

	if settlementlc == nil {
		settlementlc = slregistry.GetClient(slregistry.Mock)
	}
	_ = initSettlementLayerMock(settlementlc, defaultBatchSize, uint64(state.LastBlockHeight), uint64(state.LastBlockHeight)+1, pubsubServer, logger)

	if dalc == nil {
		dalc = &mockda.DataAvailabilityLayerClient{}
	}
	initDALCMock(dalc, conf.DABlockTime, logger)

	proxyApp := testutil.GetABCIProxyAppMock(logger.With("module", "proxy"))
	if err := proxyApp.Start(); err != nil {
		return nil, err
	}

	mp := mempoolv1.NewTxMempool(logger, tmcfg.DefaultMempoolConfig(), proxyApp.Mempool(), 0)
	manager, err := NewManager(key, conf, genesis, store, mp, proxyApp, dalc, settlementlc, nil, pubsubServer, logger)
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
func initSettlementLayerMock(settlementlc settlement.LayerClient, batchSize uint64, latestHeight uint64, batchOffsetHeight uint64, pubsubServer *pubsub.Server, logger log.Logger) error {
	conf := slmock.Config{
		Config: &settlement.Config{
			BatchSize: batchSize,
		},
	}
	byteconf, _ := json.Marshal(conf)
	return settlementlc.Init(byteconf, pubsubServer, logger)
}

func getManagerConfig() config.BlockManagerConfig {
	return config.BlockManagerConfig{
		BlockTime:         100 * time.Millisecond,
		DABlockTime:       100 * time.Millisecond,
		BatchSyncInterval: 1 * time.Second,
		BlockBatchSize:    defaultBatchSize,
		DAStartHeight:     0,
		NamespaceID:       [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
}

type SettlementLayerClientSubmitBatchError struct {
	slmock.SettlementLayerClient
}

func (s *SettlementLayerClientSubmitBatchError) SubmitBatch(_ *types.Batch, _ da.Client, _ *da.ResultSubmitBatch) *settlement.ResultSubmitBatch {
	return &settlement.ResultSubmitBatch{
		BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: connectionRefusedErrorMessage},
	}
}

type DALayerClientSubmitBatchError struct {
	mockda.DataAvailabilityLayerClient
}

func (s *DALayerClientSubmitBatchError) SubmitBatch(_ *types.Batch) da.ResultSubmitBatch {
	return da.ResultSubmitBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: connectionRefusedErrorMessage}}
}

type DALayerClientRetrieveBatchesError struct {
	mockda.DataAvailabilityLayerClient
}

func (m *DALayerClientRetrieveBatchesError) RetrieveBatches(_ uint64) da.ResultRetrieveBatch {
	return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: batchNotFoundErrorMessage}}
}
