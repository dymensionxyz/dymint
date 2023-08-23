package block

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/mempool"
	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	tmcfg "github.com/tendermint/tendermint/config"

	"github.com/dymensionxyz/dymint/testutil"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
)

func TestCreateEmptyBlocksEnableDisable(t *testing.T) {
	const blockTime = 200 * time.Millisecond
	const EmptyBlocksMaxTime = blockTime * 10
	const runTime = 10 * time.Second

	assert := assert.New(t)
	require := require.New(t)
	app := testutil.GetAppMock()
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)

	// Init manager with empty blocks feature disabled
	managerConfigCreatesEmptyBlocks := getManagerConfig()
	managerConfigCreatesEmptyBlocks.BlockTime = blockTime
	managerConfigCreatesEmptyBlocks.EmptyBlocksMaxTime = 0 * time.Second
	managerWithEmptyBlocks, err := getManager(managerConfigCreatesEmptyBlocks, nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	// Init manager with empty blocks feature enabled
	managerConfig := getManagerConfig()
	managerConfig.BlockTime = blockTime
	managerConfig.EmptyBlocksMaxTime = EmptyBlocksMaxTime
	manager, err := getManager(managerConfig, nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	//Check initial height
	initialHeight := uint64(0)
	require.Equal(initialHeight, manager.store.Height())

	mCtx, cancel := context.WithTimeout(context.Background(), runTime)
	defer cancel()
	go manager.ProduceBlockLoop(mCtx)
	go managerWithEmptyBlocks.ProduceBlockLoop(mCtx)
	<-mCtx.Done()

	require.Greater(manager.store.Height(), initialHeight)
	require.Greater(managerWithEmptyBlocks.store.Height(), initialHeight)
	assert.Greater(managerWithEmptyBlocks.store.Height(), manager.store.Height())

	// Check that blocks are created with empty blocks feature disabled
	assert.LessOrEqual(manager.store.Height(), uint64(runTime/EmptyBlocksMaxTime))
	assert.LessOrEqual(managerWithEmptyBlocks.store.Height(), uint64(runTime/blockTime))

	for i := uint64(2); i < managerWithEmptyBlocks.store.Height(); i++ {
		prevBlock, err := managerWithEmptyBlocks.store.LoadBlock(i - 1)
		assert.NoError(err)

		block, err := managerWithEmptyBlocks.store.LoadBlock(i)
		assert.NoError(err)
		assert.NotZero(block.Header.Time)

		diff := time.Unix(0, int64(block.Header.Time)).Sub(time.Unix(0, int64(prevBlock.Header.Time)))
		assert.Greater(diff, blockTime-blockTime/10)
		assert.Less(diff, blockTime+blockTime/10)
	}

	for i := uint64(2); i < manager.store.Height(); i++ {
		prevBlock, err := manager.store.LoadBlock(i - 1)
		assert.NoError(err)

		block, err := manager.store.LoadBlock(i)
		assert.NoError(err)
		assert.NotZero(block.Header.Time)

		diff := time.Unix(0, int64(block.Header.Time)).Sub(time.Unix(0, int64(prevBlock.Header.Time)))
		assert.Greater(diff, manager.conf.EmptyBlocksMaxTime)
	}
}

func TestCreateEmptyBlocksNew(t *testing.T) {
	t.Skip("FIXME: fails to submit tx to test the empty blocks feature")
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
	managerConfig.BlockTime = 200 * time.Millisecond
	managerConfig.EmptyBlocksMaxTime = 1 * time.Second
	manager, err := getManager(managerConfig, nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	abciClient, err := clientCreator.NewABCIClient()
	require.NoError(err)
	require.NotNil(abciClient)
	require.NoError(abciClient.Start())
	defer abciClient.Stop()

	mempoolCfg := tmcfg.DefaultMempoolConfig()
	mempoolCfg.KeepInvalidTxsInCache = false
	mempoolCfg.Recheck = false

	mpool := mempoolv1.NewTxMempool(log.TestingLogger(), tmcfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(abciClient), 0)

	//Check initial height
	expectedHeight := uint64(0)
	assert.Equal(expectedHeight, manager.store.Height())

	mCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go manager.ProduceBlockLoop(mCtx)

	<-time.Tick(1 * time.Second)
	err = mpool.CheckTx([]byte{1, 2, 3, 4}, nil, mempool.TxInfo{})
	require.NoError(err)

	<-mCtx.Done()
	foundTx := false
	assert.LessOrEqual(manager.store.Height(), uint64(10))
	for i := uint64(2); i < manager.store.Height(); i++ {
		prevBlock, err := manager.store.LoadBlock(i - 1)
		assert.NoError(err)

		block, err := manager.store.LoadBlock(i)
		assert.NoError(err)
		assert.NotZero(block.Header.Time)

		diff := time.Unix(0, int64(block.Header.Time)).Sub(time.Unix(0, int64(prevBlock.Header.Time)))
		txsCount := len(block.Data.Txs)
		if txsCount == 0 {
			assert.Greater(diff, manager.conf.EmptyBlocksMaxTime)
			assert.Less(diff, manager.conf.EmptyBlocksMaxTime+1*time.Second)
		} else {
			foundTx = true
			assert.Less(diff, manager.conf.BlockTime+100*time.Millisecond)
		}

		fmt.Println("time diff:", diff, "tx len", 0)
	}
	assert.True(foundTx)
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
		BlockTime:              blockTime,
		EmptyBlocksMaxTime:     0,
		BatchSubmitMaxTime:     submitTimeout,
		BlockBatchSize:         batchSize,
		BlockBatchMaxSizeBytes: 1000,
	}

	manager, err := getManager(managerConfig, nil, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	//Check initial height
	initialHeight := uint64(0)
	require.Equal(initialHeight, manager.store.Height())
	require.True(manager.batchInProcess.Load() == false)

	mCtx, cancel := context.WithTimeout(context.Background(), runTime)
	defer cancel()
	go manager.ProduceBlockLoop(mCtx)
	go manager.SubmitLoop(mCtx)
	<-mCtx.Done()

	require.True(manager.batchInProcess.Load() == true)
}
