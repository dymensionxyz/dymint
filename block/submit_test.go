package block_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"

	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	slmocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/version"
)

// TestBatchOverhead tests the scenario where we have a single block that is very large, and occupies the entire batch size.
// This test is to ensure the value of 90% for types.MaxBlockSizeAdjustment is valid
// 2 use cases:
// 1. single block with single large tx
// 2. single block with multiple small tx
func TestBatchOverhead(t *testing.T) {
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)

	maxBatchSize := uint64(10_000)                                            // 10KB
	maxTxData := uint64(float64(maxBatchSize) * types.MaxBlockSizeAdjustment) // 90% of maxBatchSize

	// first batch with single block with single large tx
	tcases := []struct {
		name string
		nTxs int
	}{
		{
			name: "single block with single large tx",
			nTxs: 1,
		},
		{
			name: "single block with multiple small tx",
			nTxs: 100,
		},
	}

	for _, tcase := range tcases {
		blocks, err := testutil.GenerateBlocks(1, 1, manager.LocalKey)
		require.NoError(t, err)
		block := blocks[0]

		mallete := func(nTxs int) {
			txSize := maxTxData / uint64(nTxs)

			block.Data = types.Data{
				Txs: make(types.Txs, nTxs),
			}

			for i := 0; i < nTxs; i++ {
				block.Data.Txs[0] = testutil.GetRandomBytes(txSize)
			}
		}

		mallete(tcase.nTxs)

		commits, err := testutil.GenerateCommits(blocks, manager.LocalKey)
		require.NoError(t, err)
		commit := commits[0]

		batch := types.Batch{
			Blocks:  blocks,
			Commits: commits,
		}

		batchSize := batch.ToProto().Size()

		var blocksize, commitSize int
		blocksize = block.ToProto().Size()
		commitSize = commit.ToProto().Size()

		// we assert that the batch size is not larger than the maxBatchSize
		assert.LessOrEqual(t, batchSize, int(maxBatchSize), tcase.name)

		t.Log("Batch size:", batchSize, "Max batch size:", maxBatchSize, tcase.name)
		t.Log("Commit size:", commitSize, tcase.name)
		t.Log("Block size:", blocksize, tcase.name)
		t.Log("Tx size:", maxTxData, "num of txs:", len(blocks[0].Data.Txs), tcase.name)
		t.Log("Overhead:", batchSize-int(maxTxData), tcase.name)
	}
}

func TestBatchSubmissionHappyFlow(t *testing.T) {
	require := require.New(t)
	app := testutil.GetAppMock(testutil.EndBlock)
	ctx := context.Background()
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{RollappConsensusParamUpdates: &abci.RollappConsensusParams{
		Da:     "mock",
		Commit: version.Commit,
		Block: &abci.BlockParams{
			MaxBytes: 500000,
			MaxGas:   40000000,
		},
	}})
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	// Check initial assertions
	initialHeight := uint64(0)
	require.Zero(manager.State.Height())
	require.Zero(manager.LastSubmittedHeight.Load())

	// Produce block and validate that we produced blocks
	_, _, err = manager.ProduceApplyGossipBlock(ctx, true)
	require.NoError(err)
	assert.Greater(t, manager.State.Height(), initialHeight)
	assert.Zero(t, manager.LastSubmittedHeight.Load())

	// submit and validate sync target
	manager.CreateAndSubmitBatch(manager.Conf.BatchSubmitBytes, false)
	assert.EqualValues(t, manager.State.Height(), manager.LastSubmittedHeight.Load())
}

func TestBatchSubmissionFailedSubmission(t *testing.T) {
	require := require.New(t)
	app := testutil.GetAppMock(testutil.EndBlock)
	ctx := context.Background()
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{RollappConsensusParamUpdates: &abci.RollappConsensusParams{
		Da:     "mock",
		Commit: version.Commit,
		Block: &abci.BlockParams{
			MaxBytes: 500000,
			MaxGas:   40000000,
		},
	}})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(err)

	lib2pPrivKey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	require.NoError(err)

	proposerKey := tmed25519.PrivKey(priv)
	proposer := *types.NewSequencer(proposerKey.PubKey(), "")

	// Create a new mock ClientI
	slmock := &slmocks.MockClientI{}
	slmock.On("Init", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	slmock.On("Start").Return(nil)
	slmock.On("GetProposer").Return(proposer)

	manager, err := testutil.GetManagerWithProposerKey(testutil.GetManagerConfig(), lib2pPrivKey, slmock, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	// Check initial assertions
	initialHeight := uint64(0)
	require.Zero(manager.State.Height())
	require.Zero(manager.LastSubmittedHeight.Load())

	// Produce block and validate that we produced blocks
	_, _, err = manager.ProduceApplyGossipBlock(ctx, true)
	require.NoError(err)
	assert.Greater(t, manager.State.Height(), initialHeight)
	assert.Zero(t, manager.LastSubmittedHeight.Load())

	// try to submit, we expect failure
	slmock.On("SubmitBatch", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("submit batch")).Once()
	_, err = manager.CreateAndSubmitBatch(manager.Conf.BatchSubmitBytes, false)
	assert.Error(t, err)

	// try to submit again, we expect success
	slmock.On("SubmitBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	manager.CreateAndSubmitBatch(manager.Conf.BatchSubmitBytes, false)
	assert.EqualValues(t, manager.State.Height(), manager.LastSubmittedHeight.Load())
}

// TestSubmissionByTime tests the submission trigger by time
func TestSubmissionByTime(t *testing.T) {
	const (
		// large batch size, so we expect the trigger to be the timeout
		submitTimeout = 1 * time.Second
		blockTime     = 200 * time.Millisecond
	)

	require := require.New(t)
	app := testutil.GetAppMock(testutil.EndBlock)
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{RollappConsensusParamUpdates: &abci.RollappConsensusParams{
		Da:     "mock",
		Commit: version.Commit,
		Block: &abci.BlockParams{
			MaxBytes: 500000,
			MaxGas:   40000000,
		},
	}})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(err)

	// Init manager with empty blocks feature enabled
	managerConfig := config.BlockManagerConfig{
		BlockTime:        blockTime,
		MaxIdleTime:      0,
		BatchSkewBlocks:  10,
		BatchSubmitTime:  submitTimeout,
		BatchSubmitBytes: 1000,
	}

	manager, err := testutil.GetManager(managerConfig, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(err)

	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	// Check initial height
	initialHeight := uint64(0)
	require.Equal(initialHeight, manager.State.Height())
	require.Zero(manager.LastSubmittedHeight.Load())

	var wg sync.WaitGroup
	mCtx, cancel := context.WithTimeout(context.Background(), 2*submitTimeout)
	defer cancel()

	wg.Add(2) // Add 2 because we have 2 goroutines

	bytesProducedC := make(chan int)
	go func() {
		manager.ProduceBlockLoop(mCtx, bytesProducedC)
		wg.Done() // Decrease counter when this goroutine finishes
	}()

	go func() {
		manager.SubmitLoop(mCtx, bytesProducedC)
		wg.Done() // Decrease counter when this goroutine finishes
	}()

	wg.Wait() // Wait for all goroutines to finish
	require.True(0 < manager.LastSubmittedHeight.Load())
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
		app := testutil.GetAppMock(testutil.EndBlock)
		app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{RollappConsensusParamUpdates: &abci.RollappConsensusParams{
			Da:     "mock",
			Commit: version.Commit,
			Block: &abci.BlockParams{
				MaxBytes: 500000,
				MaxGas:   40000000,
			},
		}})
		// Create proxy app
		clientCreator := proxy.NewLocalClientCreator(app)
		proxyApp := proxy.NewAppConns(clientCreator)
		err := proxyApp.Start()
		require.NoError(err)

		managerConfig := testutil.GetManagerConfig()
		managerConfig.BatchSubmitBytes = c.blockBatchMaxSizeBytes
		manager, err := testutil.GetManager(managerConfig, nil, 1, 1, 0, proxyApp, nil)
		require.NoError(err)

		manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
		manager.Retriever = manager.DAClient.(da.BatchRetriever)

		assert.Equal(manager.State.Height(), uint64(0))

		submissionByBatchSize(manager, assert, c.expectedSubmission)
	}
}

func submissionByBatchSize(manager *block.Manager, assert *assert.Assertions, expectedSubmission bool) {
	var wg sync.WaitGroup
	wg.Add(2) // Add 2 because we have 2 goroutines

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	bytesProducedC := make(chan int)

	go func() {
		manager.ProduceBlockLoop(ctx, bytesProducedC)
		wg.Done() // Decrease counter when this goroutine finishes
	}()

	go func() {
		assert.Zero(manager.LastSubmittedHeight.Load())
		manager.SubmitLoop(ctx, bytesProducedC)
		wg.Done() // Decrease counter when this goroutine finishes
	}()

	// wait for block to be produced but not for submission threshold
	time.Sleep(200 * time.Millisecond)
	// assert block produced but nothing submitted yet
	assert.Greater(manager.State.Height(), uint64(0))

	wg.Wait() // Wait for all goroutines to finish

	if expectedSubmission {
		assert.Positive(manager.LastSubmittedHeight.Load())
	} else {
		assert.Zero(manager.LastSubmittedHeight.Load())
	}
}
