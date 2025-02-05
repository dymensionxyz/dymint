package block_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/da"
	blockmocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	"github.com/dymensionxyz/dymint/utils/event"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"
)

type mockError struct {
	name string
	data string
}

func (m mockError) Error() string {
	return "some string"
}

func (mockError) Unwrap() error {
	return gerrc.ErrFault
}

// TODO: should move to gerrc tests
func TestErrorIsErrFault(t *testing.T) {
	err := mockError{name: "test", data: "test"}

	if !errors.Is(err, gerrc.ErrFault) {
		t.Error("Expected Is to return true")
	}

	anotherErr := errors.New("some error")

	if errors.Is(anotherErr, gerrc.ErrFault) {
		t.Error("Expected Is to return false")
	}
}

func waitForUnhealthy(fraudEventReceived chan *events.DataHealthStatus) func(msg pubsub.Message) {
	return func(msg pubsub.Message) {
		event, ok := msg.Data().(*events.DataHealthStatus)
		if !ok {
			return
		}
		fraudEventReceived <- event
	}
}

func TestP2PBlockWithFraud(t *testing.T) {
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
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)
	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	// mock executor that returns ErrFault on ExecuteBlock
	mockExecutor := &blockmocks.MockExecutorI{}
	manager.Executor = mockExecutor
	mockExecutor.On("GetAppInfo").Return(&abci.ResponseInfo{
		LastBlockHeight: int64(0),
	}, nil)
	mockExecutor.On("ExecuteBlock", mock.Anything, mock.Anything).Return(nil, gerrc.ErrFault)

	// Channel to receive the fraud event
	fraudEventReceived := make(chan *events.DataHealthStatus, 1)
	_, manager.Cancel = context.WithCancel(context.Background())
	manager.FraudHandler = block.NewFreezeHandler(manager)

	go event.MustSubscribe(
		context.Background(),
		manager.Pubsub,
		"testFraudClient",
		events.QueryHealthStatus,
		waitForUnhealthy(fraudEventReceived),
		log.NewNopLogger(),
	)

	blocks, err := testutil.GenerateBlocks(manager.NextHeightToSubmit(), 1, manager.LocalKey, [32]byte{})
	assert.NoError(t, err)
	commits, err := testutil.GenerateCommits(blocks, manager.LocalKey)
	assert.NoError(t, err)

	t.Log("Submitting block")
	blockData := p2p.BlockData{Block: *blocks[0], Commit: *commits[0]}
	msg := pubsub.NewMessage(blockData, map[string][]string{p2p.EventTypeKey: {p2p.EventNewGossipedBlock}})
	manager.OnReceivedBlock(msg)

	select {
	case receivedEvent := <-fraudEventReceived:
		t.Log("Received fraud event")
		if receivedEvent.Error == nil {
			t.Error("there should be an error in the event")
		} else if !errors.Is(receivedEvent.Error, gerrc.ErrFault) {
			t.Errorf("Unexpected error received, expected: %v, got: %v", gerrc.ErrFault, receivedEvent.Error)
		}
	case <-time.After(2 * time.Second): // time for the fraud event
		t.Error("expected to receive a fraud event")
	}

	mockExecutor.AssertExpectations(t)
}

func TestLocalBlockWithFraud(t *testing.T) {
	t.Skip("TODO: this not actually testing a local block")
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
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)
	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	numBatchesToAdd := 2
	nextBatchStartHeight := manager.NextHeightToSubmit()
	var batch *types.Batch
	for i := 0; i < numBatchesToAdd; i++ {
		batch, err = testutil.GenerateBatch(
			nextBatchStartHeight,
			nextBatchStartHeight+uint64(testutil.DefaultTestBatchSize-1),
			manager.LocalKey,
			[32]byte{},
		)
		assert.NoError(t, err)

		// Save one block on state to enforce local block application
		_, err = manager.Store.SaveBlock(batch.Blocks[0], batch.Commits[0], nil)
		require.NoError(t, err)

		daResultSubmitBatch := manager.DAClient.SubmitBatch(batch)
		assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)

		err = manager.SLClient.SubmitBatch(batch, manager.DAClient.GetClientType(), &daResultSubmitBatch)
		require.NoError(t, err)

		nextBatchStartHeight = batch.EndHeight() + 1

		time.Sleep(time.Millisecond * 500)
	}

	// mock executor that returns ErrFault on ExecuteBlock
	mockExecutor := &blockmocks.MockExecutorI{}
	manager.Executor = mockExecutor
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockExecutor.On("InitChain", mock.Anything, mock.Anything, mock.Anything).Return(&abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)
	mockExecutor.On("GetAppInfo").Return(&abci.ResponseInfo{
		LastBlockHeight: int64(batch.EndHeight()),
	}, nil)
	mockExecutor.On("UpdateStateAfterInitChain", mock.Anything, mock.Anything).Return(nil)
	mockExecutor.On("UpdateMempoolAfterInitChain", mock.Anything).Return(nil)
	mockExecutor.On("ExecuteBlock", mock.Anything, mock.Anything).Return(nil, gerrc.ErrFault)

	// Channel to receive the fraud event
	fraudEventReceived := make(chan *events.DataHealthStatus, 1)
	_, manager.Cancel = context.WithCancel(context.Background())
	manager.FraudHandler = block.NewFreezeHandler(manager)

	go event.MustSubscribe(
		context.Background(),
		manager.Pubsub,
		"testFraudClient",
		events.QueryHealthStatus,
		waitForUnhealthy(fraudEventReceived),
		log.NewNopLogger(),
	)

	// Initially sync target is 0
	assert.Zero(t, manager.LastSettlementHeight.Load())
	assert.True(t, manager.State.Height() == 0)

	// enough time to sync and produce blocks
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	// Capture the error returned by manager.Start.

	errChan := make(chan error, 1)
	go func() {
		errChan <- manager.Start(ctx)
		err := <-errChan
		require.Truef(t, errors.Is(err, gerrc.ErrFault), "expected error to be %v, got: %v", gerrc.ErrFault, err)
	}()
	<-ctx.Done()

	select {
	case receivedEvent := <-fraudEventReceived:
		t.Log("Received fraud event")
		if receivedEvent.Error == nil {
			t.Error("there should be an error in the event")
		} else if !errors.Is(receivedEvent.Error, gerrc.ErrFault) {
			t.Errorf("Unexpected error received, expected: %v, got: %v", gerrc.ErrFault, receivedEvent.Error)
		}
	case <-time.After(2 * time.Second): // time for the fraud event
		t.Error("expected to receive a fraud event")
	}

	assert.Equal(t, batch.EndHeight(), manager.LastSettlementHeight.Load())
	mockExecutor.AssertExpectations(t)
}

// TestApplyBatchFromSLWithFraud tests that the ApplyBatchFromSL function returns an error if the batch is fraudulent
func TestApplyBatchFromSLWithFraud(t *testing.T) {
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
	commitHash := [32]byte{1}
	manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
	manager.Retriever = manager.DAClient.(da.BatchRetriever)
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]})

	// submit batch
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

	// Mock Executor to return ErrFraud
	mockExecutor := &blockmocks.MockExecutorI{}
	manager.Executor = mockExecutor
	mockExecutor.On("GetAppInfo").Return(&abci.ResponseInfo{
		LastBlockHeight: int64(batch.EndHeight()),
	}, nil)
	mockExecutor.On("ExecuteBlock", mock.Anything, mock.Anything).Return(nil, gerrc.ErrFault)
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]}).Once()
	app.On("Info", mock.Anything).Return(abci.ResponseInfo{
		LastBlockHeight:  int64(batch.EndHeight()),
		LastBlockAppHash: commitHash[:],
	})

	var bds []rollapp.BlockDescriptor
	for _, block := range batch.Blocks {
		bds = append(bds, rollapp.BlockDescriptor{
			Height: block.Header.Height,
		})
	}
	slBatch := &settlement.Batch{
		MetaData:         daResultSubmitBatch.SubmitMetaData,
		BlockDescriptors: bds,
	}

	// Call ApplyBatchFromSL
	err = manager.ApplyBatchFromSL(slBatch)

	// Verify
	require.True(errors.Is(err, gerrc.ErrFault))
	mockExecutor.AssertExpectations(t)
}
