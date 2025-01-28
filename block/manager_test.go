package block_test

import (
	"context"
	"crypto/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	"github.com/dymensionxyz/dymint/version"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"

	slregistry "github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/dymensionxyz/dymint/store"
)

// TODO: test loading sequencer while rotation in progress
// TODO: test sequencer after L2 handover but before last state update submitted
// TODO: test halt scenario
// TODO: TestApplyCachedBlocks_WithFraudCheck

func TestInitialState(t *testing.T) {
	version.DRS = "0"
	var err error
	assert := assert.New(t)
	genesis := testutil.GenerateGenesis(123)
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	raw, _ := key.GetPublic().Raw()
	pubkey := ed25519.PubKey(raw)
	sampleState := testutil.GenerateStateWithSequencer(1, 128, pubkey)

	conf := config.NodeConfig{
		BlockManagerConfig: testutil.GetManagerConfig(),
	}
	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(t, err)
	proxyApp := testutil.GetABCIProxyAppMock(logger.With("module", "proxy"))
	settlementlc := slregistry.GetClient(slregistry.Local)
	_ = settlementlc.Init(settlement.Config{}, genesis.ChainID, pubsubServer, logger)

	// Init empty store and full store
	emptyStore := store.New(store.NewDefaultInMemoryKVStore())
	fullStore := store.New(store.NewDefaultInMemoryKVStore())
	_, err = fullStore.SaveState(sampleState, nil)
	require.NoError(t, err)

	// Init p2p client
	privKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	p2pClient, err := p2p.NewClient(config.P2PConfig{
		ListenAddress:                config.DefaultListenAddress,
		GossipSubCacheSize:           50,
		BootstrapRetryTime:           30 * time.Second,
		BlockSyncRequestIntervalTime: 30 * time.Second,
		BlockSyncEnabled:             true,
		DiscoveryEnabled:             true,
	}, privKey, "TestChain", emptyStore, pubsubServer, datastore.NewMapDatastore(), logger)
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
		expectedInitialHeight   uint64
		expectedLastBlockHeight uint64
		expectedChainID         string
	}{
		{
			name:                    "empty store",
			store:                   emptyStore,
			genesis:                 genesis,
			expectedInitialHeight:   uint64(genesis.InitialHeight),
			expectedLastBlockHeight: 0,
			expectedChainID:         genesis.ChainID,
		},
		{
			name:                    "state in store",
			store:                   fullStore,
			genesis:                 genesis,
			expectedInitialHeight:   sampleState.InitialHeight,
			expectedLastBlockHeight: sampleState.Height(),
			expectedChainID:         sampleState.ChainID,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			agg, err := block.NewManager(key, conf, c.genesis, "", c.store, nil, proxyApp, settlementlc,
				nil, pubsubServer, p2pClient, nil, nil, logger)
			assert.NoError(err)
			assert.NotNil(agg)
			assert.Equal(c.expectedChainID, agg.State.ChainID)
			assert.Equal(c.expectedInitialHeight, agg.State.InitialHeight)
			assert.Equal(c.expectedLastBlockHeight, agg.State.Height())
		})
	}
}

//	should test that we are resuming publishing blocks after we are synced
//
// 1. Submit a batch and desync the manager
// 2. Fail to produce blocks
// 2. Sync the manager
// 3. Succeed to produce blocks
func TestProduceOnlyAfterSynced(t *testing.T) {
	// Init app
	app := testutil.GetAppMock(testutil.EndBlock)
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:         "mock",
			DrsVersion: 0,
		},
		ConsensusParamUpdates: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas:   40000000,
				MaxBytes: 500000,
			},
		},
	})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(t, err)
	proposerKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	manager, err := testutil.GetManagerWithProposerKey(testutil.GetManagerConfig(), proposerKey, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)

	t.Log("Taking the manager out of sync by submitting a batch")
	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	numBatchesToAdd := 2
	nextBatchStartHeight := manager.NextHeightToSubmit()
	var lastBlockHeaderHash [32]byte
	var batch *types.Batch
	for i := 0; i < numBatchesToAdd; i++ {
		batch, err = testutil.GenerateBatch(
			nextBatchStartHeight, nextBatchStartHeight+uint64(testutil.DefaultTestBatchSize-1), manager.LocalKey,
			lastBlockHeaderHash,
		)
		assert.NoError(t, err)
		daResultSubmitBatch := manager.DAClient.SubmitBatch(batch)
		assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)
		err = manager.SLClient.SubmitBatch(batch, manager.DAClient.GetClientType(), &daResultSubmitBatch)
		require.NoError(t, err)
		nextBatchStartHeight = batch.EndHeight() + 1
		lastBlockHeaderHash = batch.Blocks[len(batch.Blocks)-1].Header.Hash()
		// Wait until daHeight is updated
		time.Sleep(time.Millisecond * 500)
	}

	// Initially sync target is 0
	assert.Zero(t, manager.LastSettlementHeight.Load())
	assert.True(t, manager.State.Height() == 0)

	// enough time to sync and produce blocks
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
	defer cancel()
	// Capture the error returned by manager.Start.

	err = manager.Start(ctx)
	require.NoError(t, err)
	<-ctx.Done()

	assert.Equal(t, batch.EndHeight(), manager.LastSettlementHeight.Load())
	// validate that we produced blocks
	assert.Greater(t, manager.State.Height(), batch.EndHeight())
}

func TestRetrieveDaBatchesFailed(t *testing.T) {
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)
	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	batch := &settlement.Batch{
		MetaData: &settlement.BatchMetaData{
			DA: &da.DASubmitMetaData{
				Client: da.Mock,
				Height: 1,
			},
		},
	}

	err = manager.ApplyBatchFromSL(batch)
	t.Log(err)
	assert.ErrorIs(t, err, da.ErrBlobNotFound)
}

func TestProduceNewBlock(t *testing.T) {
	// Init app
	app := testutil.GetAppMock(testutil.Commit, testutil.EndBlock)
	commitHash := [32]byte{1}
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]})
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:         "mock",
			DrsVersion: 0,
		},
		ConsensusParamUpdates: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas:   40000000,
				MaxBytes: 500000,
			},
		},
	})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(t, err)
	// Init manager
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	// Produce block
	_, _, err = manager.ProduceApplyGossipBlock(context.Background(), block.ProduceBlockOptions{AllowEmpty: true})
	require.NoError(t, err)
	// Validate state is updated with the commit hash
	assert.Equal(t, uint64(1), manager.State.Height())
	assert.Equal(t, commitHash, manager.State.AppHash)
}

func TestProducePendingBlock(t *testing.T) {
	// Init app
	app := testutil.GetAppMock(testutil.Commit, testutil.EndBlock)
	commitHash := [32]byte{1}
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]})
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:         "mock",
			DrsVersion: 0,
		},
		ConsensusParamUpdates: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas:   40000000,
				MaxBytes: 500000,
			},
		},
	})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(t, err)
	// Init manager
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	// Generate b and commit and save it to the store
	b := testutil.GetRandomBlock(1, 3)
	copy(b.Header.SequencerHash[:], manager.State.GetProposerHash())
	copy(b.Header.NextSequencersHash[:], manager.State.GetProposerHash())

	_, err = manager.Store.SaveBlock(b, &b.LastCommit, nil)
	require.NoError(t, err)
	// Produce b
	_, _, err = manager.ProduceApplyGossipBlock(context.Background(), block.ProduceBlockOptions{AllowEmpty: true})
	require.NoError(t, err)

	// Validate state is updated with the b that was saved in the store
	// hacky way to validate the b was indeed contain txs
	assert.NotEqual(t, manager.State.LastResultsHash, testutil.GetEmptyLastResultsHash())
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
	app := testutil.GetAppMock(testutil.Info, testutil.Commit, testutil.EndBlock)
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)
	// Create a new mock store which should succeed to save the first block
	mockStore := testutil.NewMockStore()
	// Init manager
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, proxyApp, mockStore)
	require.NoError(err)

	cases := []struct {
		name                 string
		shoudFailOnSaveState bool
		LastAppBlockHeight   int64
		AppCommitHash        [32]byte
		LastAppCommitHash    [32]byte
		expectedStoreHeight  uint64
		expectedStateAppHash [32]byte
	}{
		{
			name:                 "ProduceFirstBlockSuccessfully",
			shoudFailOnSaveState: false,
			AppCommitHash:        [32]byte{1},
			expectedStoreHeight:  1,
			expectedStateAppHash: [32]byte{1},
		},
		{
			name:                 "ProduceSecondBlockFailOnUpdateState",
			shoudFailOnSaveState: true,
			AppCommitHash:        [32]byte{2},
			expectedStoreHeight:  1, // height not changed on failed save state
			expectedStateAppHash: [32]byte{1},
		},
		{
			name:                 "ProduceSecondBlockSuccessfullyFromApp",
			shoudFailOnSaveState: false,
			LastAppCommitHash:    [32]byte{2}, // loading state from app
			LastAppBlockHeight:   2,
			expectedStoreHeight:  2,
			expectedStateAppHash: [32]byte{2},
		},
		{
			name:                 "ProduceThirdBlockFailOnUpdateStoreHeight",
			shoudFailOnSaveState: true,
			AppCommitHash:        [32]byte{3},
			expectedStoreHeight:  2, // height not changed on failed save state
			expectedStateAppHash: [32]byte{2},
		},
		{
			name:                 "ProduceThirdBlockSuccessfully",
			shoudFailOnSaveState: false,
			LastAppCommitHash:    [32]byte{3},
			LastAppBlockHeight:   3,
			expectedStoreHeight:  3,
			expectedStateAppHash: [32]byte{3},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: tc.AppCommitHash[:]}).Once()
			app.On("Info", mock.Anything).Return(abci.ResponseInfo{
				LastBlockHeight:  tc.LastAppBlockHeight,
				LastBlockAppHash: tc.LastAppCommitHash[:],
			})
			app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
				RollappParamUpdates: &abci.RollappParams{
					Da:         "mock",
					DrsVersion: 0,
				},
				ConsensusParamUpdates: &abci.ConsensusParams{
					Block: &abci.BlockParams{
						MaxGas:   40000000,
						MaxBytes: 500000,
					},
				},
			})
			mockStore.ShouldFailUpdateStateWithBatch = tc.shoudFailOnSaveState
			_, _, _ = manager.ProduceApplyGossipBlock(context.Background(), block.ProduceBlockOptions{AllowEmpty: true})
			storeState, err := manager.Store.LoadState()
			assert.NoError(err)
			manager.State = storeState
			require.Equal(tc.expectedStoreHeight, storeState.Height(), tc.name)
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
	app := testutil.GetAppMock(testutil.EndBlock)
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:         "mock",
			DrsVersion: 0,
		},
		ConsensusParamUpdates: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas:   40000000,
				MaxBytes: 500000,
			},
		},
	})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)
	// Init manager
	managerConfig := testutil.GetManagerConfig()
	managerConfig.BatchSubmitBytes = batchLimitBytes // enough for 2 block, not enough for 10 blocks
	manager, err := testutil.GetManager(managerConfig, nil, 1, 1, 0, proxyApp, nil)
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
				_, _, err := manager.ProduceApplyGossipBlock(ctx, block.ProduceBlockOptions{AllowEmpty: true})
				assert.NoError(err)
			}

			// Call createNextDABatch function
			startHeight := manager.NextHeightToSubmit()
			endHeight := startHeight + uint64(tc.blocksToProduce) - 1
			batch, err := manager.CreateBatch(manager.Conf.BatchSubmitBytes, startHeight, endHeight)
			assert.NoError(err)

			assert.Equal(batch.StartHeight(), startHeight)
			assert.LessOrEqual(batch.SizeBytes(), int(managerConfig.BatchSubmitBytes))

			if !tc.expectedToBeTruncated {
				assert.Equal(batch.EndHeight(), endHeight)
			} else {
				assert.Equal(batch.EndHeight(), batch.StartHeight()+batch.NumBlocks()-1)
				assert.Less(batch.EndHeight(), endHeight)

				// validate next added block to batch would have been actually too big
				// First relax the byte limit so we could produce larger batch
				manager.Conf.BatchSubmitBytes = 10 * manager.Conf.BatchSubmitBytes
				newBatch, err := manager.CreateBatch(manager.Conf.BatchSubmitBytes, startHeight, batch.EndHeight()+1)
				assert.Greater(newBatch.SizeBytes(), batchLimitBytes)

				assert.NoError(err)
			}
		})
	}
}

func TestDAFetch(t *testing.T) {
	require := require.New(t)
	// Setup app
	app := testutil.GetAppMock(testutil.Info, testutil.Commit, testutil.EndBlock)
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:         "mock",
			DrsVersion: 0,
		},
		ConsensusParamUpdates: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas:   40000000,
				MaxBytes: 500000,
			},
		},
	})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)
	// Create a new mock store which should succeed to save the first block
	mockStore := testutil.NewMockStore()
	// Init manager
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, proxyApp, mockStore)
	require.NoError(err)
	commitHash := [32]byte{}

	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]})

	nextBatchStartHeight := manager.NextHeightToSubmit()
	batch, err := testutil.GenerateBatch(
		nextBatchStartHeight,
		nextBatchStartHeight+uint64(testutil.DefaultTestBatchSize-1),
		manager.LocalKey,
		[32]byte{},
	)
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
			err:        da.ErrDAMismatch,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]}).Once()
			app.On("Info", mock.Anything).Return(abci.ResponseInfo{
				LastBlockHeight:  int64(batch.EndHeight() + 1),
				LastBlockAppHash: commitHash[:],
			})

			var bds []rollapp.BlockDescriptor
			for _, block := range batch.Blocks {
				bds = append(bds, rollapp.BlockDescriptor{
					Height: block.Header.Height,
				})
			}
			slBatch := &settlement.Batch{
				MetaData: &settlement.BatchMetaData{
					DA: c.daMetaData,
				},
				BlockDescriptors: bds,
				EndHeight:        batch.EndHeight(),
			}
			err := manager.ApplyBatchFromSL(slBatch)
			require.Equal(c.err, err)
		})
	}
}

func TestManager_updateTargetHeight(t *testing.T) {
	tests := []struct {
		name            string
		TargetHeight    uint64
		h               uint64
		expTargetHeight uint64
	}{
		{
			name:            "no update target height",
			TargetHeight:    100,
			h:               99,
			expTargetHeight: 100,
		}, {
			name:            "update target height",
			TargetHeight:    100,
			h:               101,
			expTargetHeight: 101,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &block.Manager{
				TargetHeight: atomic.Uint64{},
			}
			m.TargetHeight.Store(tt.TargetHeight)
			m.UpdateTargetHeight(tt.h)
			assert.Equal(t, tt.expTargetHeight, m.TargetHeight.Load())
		})
	}
}
