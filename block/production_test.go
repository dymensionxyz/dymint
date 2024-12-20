package block_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	rdktypes "github.com/dymensionxyz/dymension-rdk/x/sequencers/types"
	"github.com/gogo/protobuf/proto"
	prototypes "github.com/gogo/protobuf/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"

	block2 "github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/mempool"
	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	slmocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	uchannel "github.com/dymensionxyz/dymint/utils/channel"
	uevent "github.com/dymensionxyz/dymint/utils/event"
	"github.com/dymensionxyz/dymint/version"
)

// TODO: test producing lastBlock
// TODO: test using already produced lastBlock
func TestCreateEmptyBlocksEnableDisable(t *testing.T) {
	version.DRS = "0"
	const blockTime = 200 * time.Millisecond
	const MaxIdleTime = blockTime * 10
	const runTime = 10 * time.Second

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

	// Init manager with empty blocks feature disabled
	managerConfigCreatesEmptyBlocks := testutil.GetManagerConfig()
	managerConfigCreatesEmptyBlocks.BlockTime = blockTime
	managerConfigCreatesEmptyBlocks.MaxIdleTime = 0 * time.Second
	managerWithEmptyBlocks, err := testutil.GetManager(managerConfigCreatesEmptyBlocks, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	// Init manager with empty blocks feature enabled
	managerConfig := testutil.GetManagerConfig()
	managerConfig.BlockTime = blockTime
	managerConfig.MaxIdleTime = MaxIdleTime
	managerConfig.MaxProofTime = MaxIdleTime
	manager, err := testutil.GetManager(managerConfig, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	// Check initial height
	initialHeight := uint64(0)
	require.Equal(initialHeight, manager.State.Height())

	mCtx, cancel := context.WithTimeout(context.Background(), runTime)
	defer cancel()
	bytesProduced1 := make(chan int)
	bytesProduced2 := make(chan int)
	go manager.ProduceBlockLoop(mCtx, bytesProduced1)
	go managerWithEmptyBlocks.ProduceBlockLoop(mCtx, bytesProduced2)
	uchannel.DrainForever(bytesProduced1, bytesProduced2)
	<-mCtx.Done()

	require.Greater(manager.State.Height(), initialHeight)
	require.Greater(managerWithEmptyBlocks.State.Height(), initialHeight)
	assert.Greater(managerWithEmptyBlocks.State.Height(), manager.State.Height())

	// Check that blocks are created with empty blocks feature disabled
	assert.LessOrEqual(manager.State.Height(), uint64(runTime/MaxIdleTime))
	assert.LessOrEqual(managerWithEmptyBlocks.State.Height(), uint64(runTime/blockTime))

	for i := uint64(2); i < managerWithEmptyBlocks.State.Height(); i++ {
		prevBlock, err := managerWithEmptyBlocks.Store.LoadBlock(i - 1)
		assert.NoError(err)

		block, err := managerWithEmptyBlocks.Store.LoadBlock(i)
		assert.NoError(err)
		assert.NotZero(block.Header.GetTimestamp())

		diff := block.Header.GetTimestamp().Sub(prevBlock.Header.GetTimestamp())
		assert.Greater(diff, blockTime-blockTime/10)
		assert.Less(diff, blockTime+blockTime/10)
	}

	for i := uint64(2); i < manager.State.Height(); i++ {
		prevBlock, err := manager.Store.LoadBlock(i - 1)
		assert.NoError(err)

		block, err := manager.Store.LoadBlock(i)
		assert.NoError(err)
		assert.NotZero(block.Header.GetTimestamp())

		diff := block.Header.GetTimestamp().Sub(prevBlock.Header.GetTimestamp())
		assert.Greater(diff, manager.Conf.MaxIdleTime)
	}
}

func TestCreateEmptyBlocksNew(t *testing.T) {
	// TODO(https://github.com/dymensionxyz/dymint/issues/352)
	t.Skip("TODO: fails to submit tx to test the empty blocks feature")
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
	managerConfig.BlockTime = 200 * time.Millisecond
	managerConfig.MaxIdleTime = 1 * time.Second
	manager, err := testutil.GetManager(managerConfig, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	abciClient, err := clientCreator.NewABCIClient()
	require.NoError(err)
	require.NotNil(abciClient)
	require.NoError(abciClient.Start())

	defer func() {
		// Capture the error returned by abciClient.Stop and handle it.
		err := abciClient.Stop()
		if err != nil {
			t.Logf("Error stopping ABCI client: %v", err)
		}
	}()

	mempoolCfg := tmcfg.DefaultMempoolConfig()
	mempoolCfg.KeepInvalidTxsInCache = false
	mempoolCfg.Recheck = false

	mpool := mempoolv1.NewTxMempool(log.TestingLogger(), tmcfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(abciClient), 0)

	// Check initial height
	expectedHeight := uint64(0)
	assert.Equal(expectedHeight, manager.State.Height())

	mCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	bytesProduced := make(chan int)
	go manager.ProduceBlockLoop(mCtx, bytesProduced)
	uchannel.DrainForever(bytesProduced)

	<-time.Tick(1 * time.Second)
	err = mpool.CheckTx([]byte{1, 2, 3, 4}, nil, mempool.TxInfo{})
	require.NoError(err)

	<-mCtx.Done()
	foundTx := false
	assert.LessOrEqual(manager.State.Height(), uint64(10))
	for i := uint64(2); i < manager.State.Height(); i++ {
		prevBlock, err := manager.Store.LoadBlock(i - 1)
		assert.NoError(err)

		block, err := manager.Store.LoadBlock(i)
		assert.NoError(err)
		assert.NotZero(block.Header.GetTimestamp())

		diff := block.Header.GetTimestamp().Sub(prevBlock.Header.GetTimestamp())
		txsCount := len(block.Data.Txs)
		if txsCount == 0 {
			assert.Greater(diff, manager.Conf.MaxIdleTime)
			assert.Less(diff, manager.Conf.MaxIdleTime+1*time.Second)
		} else {
			foundTx = true
			assert.Less(diff, manager.Conf.BlockTime+100*time.Millisecond)
		}

		fmt.Println("time diff:", diff, "tx len", 0)
	}
	assert.True(foundTx)
}

// TestStopBlockProduction tests the block production stops when submitter is full
// and resumes when submitter is ready to accept more batches
func TestStopBlockProduction(t *testing.T) {
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

	managerConfig := testutil.GetManagerConfig()
	managerConfig.BatchSubmitBytes = 1000 // small batch size to fill up quickly
	manager, err := testutil.GetManager(managerConfig, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	assert.Equal(manager.State.Height(), uint64(0))

	// subscribe to health status event
	eventReceivedCh := make(chan error)
	cb := func(event pubsub.Message) {
		eventReceivedCh <- event.Data().(*events.DataHealthStatus).Error
	}
	go uevent.MustSubscribe(context.Background(), manager.Pubsub, "HealthStatusHandler", events.QueryHealthStatus, cb, log.TestingLogger())

	var wg sync.WaitGroup
	wg.Add(2) // Add 2 because we have 2 goroutines

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	bytesProducedC := make(chan int)

	go func() {
		manager.ProduceBlockLoop(ctx, bytesProducedC)
		wg.Done() // Decrease counter when this goroutine finishes
	}()

	// validate block production works
	time.Sleep(400 * time.Millisecond)
	assert.Greater(manager.State.Height(), uint64(0))

	// we don't read from the submit channel, so we assume it get full
	// we expect the block production to stop and unhealthy event to be emitted
	select {
	case <-ctx.Done():
		t.Error("expected unhealthy event")
	case err := <-eventReceivedCh:
		assert.Error(err)
	}

	stoppedHeight := manager.State.Height()

	// make sure block production is stopped
	time.Sleep(400 * time.Millisecond)
	assert.Equal(stoppedHeight, manager.State.Height())

	// consume the signal
	<-bytesProducedC

	// check for health status event and block production to continue
	select {
	case <-ctx.Done():
		t.Error("expected health event")
	case err := <-eventReceivedCh:
		assert.NoError(err)
	}

	// make sure block production is resumed
	time.Sleep(400 * time.Millisecond)
	assert.Greater(manager.State.Height(), stoppedHeight)
}

func TestUpdateInitialSequencerSet(t *testing.T) {

	require := require.New(t)
	app := testutil.GetAppMock(testutil.EndBlock)
	ctx := context.Background()
	version.DRS = "0"
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

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(err)

	lib2pPrivKey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	require.NoError(err)

	proposer := testutil.GenerateSequencer()
	sequencer := testutil.GenerateSequencer()

	// Create a new mock ClientI
	slmock := &slmocks.MockClientI{}
	slmock.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	slmock.On("Start").Return(nil)
	slmock.On("GetProposer").Return(proposer)
	slmock.On("GetAllSequencers").Return([]types.Sequencer{proposer, sequencer}, nil)
	slmock.On("GetRollapp").Return(&types.Rollapp{}, nil)

	manager, err := testutil.GetManagerWithProposerKey(testutil.GetManagerConfig(), lib2pPrivKey, slmock, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	// Check initial assertions
	require.Zero(manager.State.Height())
	require.Zero(manager.LastSettlementHeight.Load())

	// Simulate updating the sequencer set from SL on start
	err = manager.UpdateSequencerSetFromSL()
	require.NoError(err)

	// Produce block and validate results. We expect to have two consensus messages for the two sequencers
	// since the store for the last block sequencer set is empty.
	block, _, err := manager.ProduceApplyGossipBlock(ctx, block2.ProduceBlockOptions{AllowEmpty: true})
	require.NoError(err)
	assert.Greater(t, manager.State.Height(), uint64(0))
	assert.Zero(t, manager.LastSettlementHeight.Load())

	// Validate the last block sequencer set is persisted in the store
	actualSeqSet, err := manager.Store.LoadLastBlockSequencerSet()
	require.NoError(err)
	require.Len(actualSeqSet, 2)
	require.Equal(actualSeqSet[0], proposer)
	require.Equal(actualSeqSet[1], sequencer)

	// Validate that the block has expected consensus msgs
	require.Len(block.Data.ConsensusMessages, 2)

	// Construct expected messages
	signer, err := block2.ConsensusMsgSigner(new(rdktypes.ConsensusMsgUpsertSequencer))
	require.NoError(err)

	// Msg 1
	anyPK1, err := proposer.AnyConsPubKey()
	require.NoError(err)
	expectedConsMsg1 := &rdktypes.ConsensusMsgUpsertSequencer{
		Signer:     signer.String(),
		Operator:   proposer.SettlementAddress,
		ConsPubKey: anyPK1,
		RewardAddr: proposer.RewardAddr,
		Relayers:   proposer.WhitelistedRelayers,
	}
	expectedConsMsgBytes1, err := proto.Marshal(expectedConsMsg1)
	require.NoError(err)
	anyMsg1 := &prototypes.Any{
		TypeUrl: "rollapp.sequencers.types.ConsensusMsgUpsertSequencer",
		Value:   expectedConsMsgBytes1,
	}

	// Msg 2
	anyPK2, err := sequencer.AnyConsPubKey()
	require.NoError(err)
	expectedConsMsg2 := &rdktypes.ConsensusMsgUpsertSequencer{
		Signer:     signer.String(),
		Operator:   sequencer.SettlementAddress,
		ConsPubKey: anyPK2,
		RewardAddr: sequencer.RewardAddr,
		Relayers:   sequencer.WhitelistedRelayers,
	}
	expectedConsMsgBytes2, err := proto.Marshal(expectedConsMsg2)
	require.NoError(err)
	anyMsg2 := &prototypes.Any{
		TypeUrl: "rollapp.sequencers.types.ConsensusMsgUpsertSequencer",
		Value:   expectedConsMsgBytes2,
	}

	// Verify the result
	require.True(proto.Equal(anyMsg1, block.Data.ConsensusMessages[0]))
	require.True(proto.Equal(anyMsg2, block.Data.ConsensusMessages[1]))

	// Produce one more block and validate results. We expect to have zero consensus messages
	// since there are no sequencer set updates.
	block, _, err = manager.ProduceApplyGossipBlock(ctx, block2.ProduceBlockOptions{AllowEmpty: true})
	require.NoError(err)
	assert.Greater(t, manager.State.Height(), uint64(1))
	assert.Zero(t, manager.LastSettlementHeight.Load())

	// Validate the last block sequencer set is persisted in the store
	actualSeqSet, err = manager.Store.LoadLastBlockSequencerSet()
	require.NoError(err)
	require.Len(actualSeqSet, 2)
	require.Equal(actualSeqSet[0], proposer)
	require.Equal(actualSeqSet[1], sequencer)

	// Validate that the block has expected consensus msgs
	require.Len(block.Data.ConsensusMessages, 0)
}

func TestUpdateExistingSequencerSet(t *testing.T) {
	require := require.New(t)
	app := testutil.GetAppMock(testutil.EndBlock)
	ctx := context.Background()
	version.DRS = "0"
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

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(err)

	lib2pPrivKey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	require.NoError(err)

	proposer := testutil.GenerateSequencer()
	sequencer := testutil.GenerateSequencer()

	// Create a new mock ClientI
	slmock := &slmocks.MockClientI{}
	slmock.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	slmock.On("Start").Return(nil)
	slmock.On("GetProposer").Return(proposer)
	slmock.On("GetRollapp").Return(&types.Rollapp{}, nil)

	manager, err := testutil.GetManagerWithProposerKey(testutil.GetManagerConfig(), lib2pPrivKey, slmock, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	// Set the initial sequencer set
	manager.Sequencers.Set([]types.Sequencer{proposer, sequencer})
	_, err = manager.Store.SaveLastBlockSequencerSet([]types.Sequencer{proposer, sequencer}, nil)
	require.NoError(err)

	// Check initial assertions
	require.Zero(manager.State.Height())
	require.Zero(manager.LastSettlementHeight.Load())
	// Memory has the initial sequencer set
	initialMemorySequencers := manager.Sequencers.GetAll()
	require.Len(initialMemorySequencers, 2)
	require.Equal(initialMemorySequencers[0], proposer)
	require.Equal(initialMemorySequencers[1], sequencer)
	// Store has the initial sequencer set
	initialStoreSequencers, err := manager.Store.LoadLastBlockSequencerSet()
	require.NoError(err)
	require.Len(initialStoreSequencers, 2)
	require.Equal(initialStoreSequencers[0], proposer)
	require.Equal(initialStoreSequencers[1], sequencer)

	// Update one of the sequencers and pass the update to the manager.
	// We expect that the manager will update the sequencer set in memory and
	// generate a new consensus msg during block production.
	updatedSequencer := sequencer
	const newSequencerRewardAddr = "dym1mk7pw34ypusacm29m92zshgxee3yreums8avur"
	updatedSequencer.RewardAddr = newSequencerRewardAddr
	// GetAllSequencers now return an updated sequencer
	slmock.On("GetAllSequencers").Return([]types.Sequencer{proposer, updatedSequencer}, nil)

	// Simulate updating the sequencer set from SL
	err = manager.UpdateSequencerSetFromSL()
	require.NoError(err)

	// The number of sequencers is the same. However, the second sequencer is modified.
	sequencers := manager.Sequencers.GetAll()
	require.Len(sequencers, 2)
	require.Equal(sequencers[0], proposer)
	// Now Sequencers[1] is a sequencer with the new reward address
	require.NotEqual(sequencer, sequencers[1])
	require.Equal(updatedSequencer, sequencers[1])

	// Produce block and validate results. We expect to have one consensus message for the updated sequencer.
	block, _, err := manager.ProduceApplyGossipBlock(ctx, block2.ProduceBlockOptions{AllowEmpty: true})
	require.NoError(err)
	assert.Greater(t, manager.State.Height(), uint64(0))
	assert.Zero(t, manager.LastSettlementHeight.Load())

	// Validate that the block has expected consensus msgs: one msg for one new sequencer
	require.Len(block.Data.ConsensusMessages, 1)

	// Construct the expected message
	signer, err := block2.ConsensusMsgSigner(new(rdktypes.ConsensusMsgUpsertSequencer))
	require.NoError(err)
	anyPK, err := updatedSequencer.AnyConsPubKey()
	require.NoError(err)
	expectedConsMsg := &rdktypes.ConsensusMsgUpsertSequencer{
		Signer:     signer.String(),
		Operator:   updatedSequencer.SettlementAddress,
		ConsPubKey: anyPK,
		RewardAddr: updatedSequencer.RewardAddr,
		Relayers:   updatedSequencer.WhitelistedRelayers,
	}
	expectedConsMsgBytes, err := proto.Marshal(expectedConsMsg)
	require.NoError(err)
	anyMsg1 := &prototypes.Any{
		TypeUrl: "rollapp.sequencers.types.ConsensusMsgUpsertSequencer",
		Value:   expectedConsMsgBytes,
	}

	// Verify the result
	require.True(proto.Equal(anyMsg1, block.Data.ConsensusMessages[0]))

	// Validate the last block sequencer set is persisted in the store
	actualStoreSequencers, err := manager.Store.LoadLastBlockSequencerSet()
	require.NoError(err)
	require.Len(actualStoreSequencers, 2)
	require.Equal(actualStoreSequencers[0], proposer)
	require.Equal(actualStoreSequencers[1], updatedSequencer)
}
