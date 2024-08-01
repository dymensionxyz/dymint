package block_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/version"

	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	"github.com/dymensionxyz/dymint/node/events"
	uchannel "github.com/dymensionxyz/dymint/utils/channel"
	uevent "github.com/dymensionxyz/dymint/utils/event"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcfg "github.com/tendermint/tendermint/config"

	"github.com/dymensionxyz/dymint/testutil"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"
)

// TODO: test producing lastBlock
// TODO: test using already produced lastBlock
func TestCreateEmptyBlocksEnableDisable(t *testing.T) {
	const blockTime = 200 * time.Millisecond
	const MaxIdleTime = blockTime * 10
	const runTime = 10 * time.Second

	assert := assert.New(t)
	require := require.New(t)
	app := testutil.GetAppMock(testutil.EndBlock)
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{RollappConsensusParamUpdates: &abci.RollappConsensusParams{
		Da:     "mock",
		Commit: version.Commit,
	}})
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
		assert.NotZero(block.Header.Time)

		diff := time.Unix(0, int64(block.Header.Time)).Sub(time.Unix(0, int64(prevBlock.Header.Time)))
		assert.Greater(diff, blockTime-blockTime/10)
		assert.Less(diff, blockTime+blockTime/10)
	}

	for i := uint64(2); i < manager.State.Height(); i++ {
		prevBlock, err := manager.Store.LoadBlock(i - 1)
		assert.NoError(err)

		block, err := manager.Store.LoadBlock(i)
		assert.NoError(err)
		assert.NotZero(block.Header.Time)

		diff := time.Unix(0, int64(block.Header.Time)).Sub(time.Unix(0, int64(prevBlock.Header.Time)))
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
		assert.NotZero(block.Header.Time)

		diff := time.Unix(0, int64(block.Header.Time)).Sub(time.Unix(0, int64(prevBlock.Header.Time)))
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
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{RollappConsensusParamUpdates: &abci.RollappConsensusParams{
		Da:     "mock",
		Commit: version.Commit,
	}})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)

	managerConfig := testutil.GetManagerConfig()
	managerConfig.BatchMaxSizeBytes = 1000 // small batch size to fill up quickly
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
