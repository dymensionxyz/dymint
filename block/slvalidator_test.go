package block_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/da"
	mock_settlement "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	"github.com/dymensionxyz/dymint/version"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"
)

func TestStateUpdateValidator_ValidateStateUpdate(t *testing.T) {
	version.DRS = "0"
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

	doubleSigned, err := testutil.GenerateBlocks(1, 10, proposerKey, [32]byte{})
	require.NoError(t, err)

	// Test cases
	testCases := []struct {
		name               string
		p2pBlocks          bool
		doubleSignedBlocks []*types.Block
		stateUpdateFraud   string
		expectedErrType    error
		last               bool
	}{
		{
			name:               "Successful validation applied from DA",
			p2pBlocks:          false,
			doubleSignedBlocks: nil,
			stateUpdateFraud:   "",
			expectedErrType:    nil,
		},
		{
			name:               "Successful validation applied from P2P",
			p2pBlocks:          true,
			doubleSignedBlocks: nil,
			stateUpdateFraud:   "",
			expectedErrType:    nil,
		},
		{
			name:               "Failed validation blocks not matching",
			p2pBlocks:          true,
			stateUpdateFraud:   "",
			doubleSignedBlocks: doubleSigned,
			expectedErrType:    &types.ErrStateUpdateDoubleSigningFraud{},
		},
		{
			name:               "Failed validation wrong state roots",
			p2pBlocks:          true,
			stateUpdateFraud:   "stateroot",
			doubleSignedBlocks: nil,
			expectedErrType:    &types.ErrStateUpdateStateRootNotMatchingFraud{},
		},
		{
			name:               "Failed validation batch num blocks",
			p2pBlocks:          true,
			stateUpdateFraud:   "batchnumblocks",
			doubleSignedBlocks: nil,
			expectedErrType:    &types.ErrStateUpdateNumBlocksNotMatchingFraud{},
		},
		{
			name:               "Failed validation batch num bds",
			p2pBlocks:          true,
			stateUpdateFraud:   "batchnumbds",
			doubleSignedBlocks: nil,
			expectedErrType:    &types.ErrStateUpdateNumBlocksNotMatchingFraud{},
		},
		{
			name:               "Failed validation wrong timestamps",
			p2pBlocks:          true,
			stateUpdateFraud:   "timestamp",
			doubleSignedBlocks: doubleSigned,
			expectedErrType:    &types.ErrStateUpdateTimestampNotMatchingFraud{},
		},
		{
			name:               "Failed validation wrong height",
			p2pBlocks:          true,
			stateUpdateFraud:   "height",
			doubleSignedBlocks: doubleSigned,
			expectedErrType:    &types.ErrStateUpdateHeightNotMatchingFraud{},
		},
		{
			name:               "Failed validation drs version",
			p2pBlocks:          true,
			stateUpdateFraud:   "drs",
			doubleSignedBlocks: nil,
			expectedErrType:    &types.ErrStateUpdateDRSVersionFraud{},
		},
		{
			name:               "Failed validation next sequencer",
			p2pBlocks:          true,
			stateUpdateFraud:   "nextsequencer",
			doubleSignedBlocks: nil,
			expectedErrType:    &types.ErrInvalidNextSequencersHashFraud{},
			last:               true,
		},
		{
			name:               "Failed validation wrong DA",
			p2pBlocks:          true,
			stateUpdateFraud:   "da",
			doubleSignedBlocks: nil,
			expectedErrType:    &types.ErrStateUpdateDAFraud{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create manager
			manager, err := testutil.GetManagerWithProposerKey(testutil.GetManagerConfig(), proposerKey, nil, 1, 1, 0, proxyApp, nil)
			require.NoError(t, err)
			require.NotNil(t, manager)

			// Create DA
			manager.DAClients[da.Mock] = testutil.GetMockDALC(log.TestingLogger())

			// Generate batch
			var batch *types.Batch
			batch, err = testutil.GenerateBatch(1, 10, proposerKey, [32]byte{})
			assert.NoError(t, err)

			// Submit batch to DA
			daResultSubmitBatch := manager.GetActiveDAClient().SubmitBatch(batch)
			assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)

			// Create block descriptors
			bds, err := getBlockDescriptors(batch)
			require.NoError(t, err)

			// create the batch in settlement
			slBatch := getSLBatch(bds, daResultSubmitBatch.SubmitMetaData, 1, 10, manager.State.GetProposer().SettlementAddress)

			// Create the StateUpdateValidator
			validator := block.NewSettlementValidator(testutil.NewLogger(t), manager)

			// in case double signing generate commits for these blocks
			if tc.doubleSignedBlocks != nil {
				batch.Blocks = tc.doubleSignedBlocks
				batch.Commits, err = testutil.GenerateCommits(batch.Blocks, proposerKey)
				require.NoError(t, err)
			}

			// call manager flow for p2p received blocks
			if tc.p2pBlocks {
				for i, block := range batch.Blocks {
					blockData := p2p.BlockData{Block: *block, Commit: *batch.Commits[i]}
					msg := pubsub.NewMessage(blockData, map[string][]string{p2p.EventTypeKey: {p2p.EventNewGossipedBlock}})
					manager.OnReceivedBlock(msg)
				}
				// otherwise load them from DA
			} else {
				if !tc.last {
					manager.ApplyBatchFromSL(slBatch.Batch)
				}
			}

			// set fraud data
			switch tc.stateUpdateFraud {
			case "drs":
				// set different bd drs version
				slBatch.BlockDescriptors[0].DrsVersion = 2
			case "batchnumblocks":
				// set wrong numblocks in state update
				slBatch.NumBlocks = 11
			case "batchnumbds":
				// add more block descriptors than blocks
				bds = append(bds, rollapp.BlockDescriptor{})
				slBatch.BlockDescriptors = bds
			case "stateroot":
				// post empty state root
				slBatch.BlockDescriptors[0].StateRoot = []byte{}
			case "timestamp":
				// add wrong timestamp
				slBatch.BlockDescriptors[0].Timestamp = slBatch.BlockDescriptors[0].Timestamp.Add(time.Second)
			case "height":
				// add blockdescriptor with wrong height
				slBatch.BlockDescriptors[0].Height = 2
			case "nextsequencer":
				nextProposer := testutil.GenerateSequencer()
				slBatch.NextSequencer = nextProposer.SettlementAddress
				manager.State.Proposer.Store(&nextProposer)
				manager.State.SetProposer(&nextProposer)
				seqs := manager.Sequencers.GetAll()
				seqs = append(seqs, nextProposer)
				manager.Sequencers.Set(seqs)
			case "da":
				slBatch.MetaData.Client = "celestia"
			}

			// validate the state update
			err = validator.ValidateStateUpdate(slBatch)

			// Check the result
			if tc.expectedErrType == nil {
				assert.NoError(t, err)
			} else {
				require.Equal(t, reflect.ValueOf(tc.expectedErrType).Type(), reflect.TypeOf(err))
			}
		})
	}
}

func TestStateUpdateValidator_ValidateDAFraud(t *testing.T) {
	// Init app
	app := testutil.GetAppMock(testutil.EndBlock)
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:         "celestia",
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

	// Generate batch
	batch, err := testutil.GenerateBatch(1, 10, proposerKey, [32]byte{})
	require.NoError(t, err)

	// Batch data to be included in a blob
	batchData, err := batch.MarshalBinary()
	require.NoError(t, err)

	// Batch data to be included in a fraud blob
	randomData := []byte{1, 2, 3, 4}

	// Test cases
	testCases := []struct {
		name              string
		checkAvailability bool
		blobData          [][]byte
		expectedErrType   error
	}{
		{
			name:              "Successful DA Blob",
			checkAvailability: false,
			blobData:          [][]byte{batchData},
			expectedErrType:   nil,
		},
		{
			name:              "Blob not valid",
			checkAvailability: false,
			blobData:          [][]byte{randomData},
			expectedErrType:   &types.ErrStateUpdateBlobCorruptedFraud{},
		},
		{
			name:              "Blob unavailable",
			checkAvailability: true,
			blobData:          [][]byte{batchData},
			expectedErrType:   &types.ErrStateUpdateBlobNotAvailableFraud{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create manager
			manager, err := testutil.GetManagerWithProposerKey(testutil.GetManagerConfig(), proposerKey, nil, 1, 1, 0, proxyApp, nil)
			require.NoError(t, err)
			require.NotNil(t, manager)

			// Create Mock DA
			mockDA, err := testutil.NewMockDA(t)
			require.NoError(t, err)

			// Start DA client
			manager.DAClients[da.Celestia] = mockDA.DaClient
			err = manager.DAClients[da.Celestia].Start()
			require.NoError(t, err)

			// RPC calls necessary for blob submission
			mockDA.MockRPC.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockDA.IDS, nil).Once().Run(func(args mock.Arguments) {})
			mockDA.MockRPC.On("GetByHeight", mock.Anything, mock.Anything).Return(mockDA.Header, nil).Once().Run(func(args mock.Arguments) { time.Sleep(5 * time.Millisecond) })
			mockDA.MockRPC.On("GetProofs", mock.Anything, mock.Anything, mock.Anything).Return(mockDA.BlobProofs, nil).Once().Run(func(args mock.Arguments) { time.Sleep(5 * time.Millisecond) })
			mockDA.MockRPC.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]bool{true}, nil).Once().Run(func(args mock.Arguments) { time.Sleep(5 * time.Millisecond) })

			// Submit batch to DA
			daResultSubmitBatch := manager.DAClients[da.Celestia].SubmitBatch(batch)
			assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)

			// RPC calls for successful blob retrieval
			if !tc.checkAvailability {
				mockDA.MockRPC.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(tc.blobData, nil).Run(func(args mock.Arguments) { time.Sleep(5 * time.Millisecond) })
			}

			// RPC calls for unavailable blobs
			if tc.checkAvailability {
				mockDA.MockRPC.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("retrieval error")).Run(func(args mock.Arguments) { time.Sleep(5 * time.Millisecond) })
				mockDA.MockRPC.On("GetByHeight", mock.Anything, mock.Anything).Return(mockDA.Header, nil).Once().Run(func(args mock.Arguments) { time.Sleep(5 * time.Millisecond) })
				mockDA.MockRPC.On("GetProofs", mock.Anything, mock.Anything, mock.Anything).Return(mockDA.BlobProofs, nil).Once().Run(func(args mock.Arguments) { time.Sleep(5 * time.Millisecond) })
				mockDA.MockRPC.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]bool{false}, nil).Once().Run(func(args mock.Arguments) { time.Sleep(5 * time.Millisecond) })
			}

			// Create the StateUpdateValidator
			validator := block.NewSettlementValidator(testutil.NewLogger(t), manager)

			bds, err := getBlockDescriptors(batch)
			require.NoError(t, err)
			// Generate batch with block descriptors
			slBatch := getSLBatch(bds, daResultSubmitBatch.SubmitMetaData, 1, 10, manager.State.GetProposer().SettlementAddress)

			// apply batch

			for i, block := range batch.Blocks {
				blockData := p2p.BlockData{Block: *block, Commit: *batch.Commits[i]}
				msg := pubsub.NewMessage(blockData, map[string][]string{p2p.EventTypeKey: {p2p.EventNewGossipedBlock}})
				manager.OnReceivedBlock(msg)
			}

			// Validate state
			err = validator.ValidateStateUpdate(slBatch)

			// Check the result
			if tc.expectedErrType == nil {
				assert.NoError(t, err)
			} else {
				require.Equal(t, reflect.ValueOf(tc.expectedErrType).Type(), reflect.TypeOf(err))
			}
		})
	}
}

// TestValidateStateUpdate_PartialBatchApplication tests that SettlementValidateLoop correctly skips
// batches where not all blocks have been applied yet, preventing "not found" errors.
// This test exercises the actual race condition scenario where validation is triggered while blocks are being applied.
func TestValidateStateUpdate_PartialBatchApplication(t *testing.T) {
	version.DRS = "0"

	// Setup manager with test dependencies
	manager, proposerKey, cleanup := setupManagerForRaceConditionTest(t)
	defer cleanup()

	// Generate one large batch covering heights 1-20
	largeBatch, err := testutil.GenerateBatch(1, 20, proposerKey, [32]byte{})
	require.NoError(t, err)

	// Split into two batches for testing
	batch1 := &types.Batch{
		Blocks:  largeBatch.Blocks[:10],
		Commits: largeBatch.Commits[:10],
	}

	batch2 := &types.Batch{
		Blocks:  largeBatch.Blocks[10:],
		Commits: largeBatch.Commits[10:],
	}

	// Submit to DA
	daResult1 := manager.GetActiveDAClient().SubmitBatch(batch1)
	require.Equal(t, da.StatusSuccess, daResult1.Code)

	daResult2 := manager.GetActiveDAClient().SubmitBatch(batch2)
	require.Equal(t, da.StatusSuccess, daResult2.Code)

	// Create settlement batches
	bds1, err := getBlockDescriptors(batch1)
	require.NoError(t, err)
	slBatch1 := getSLBatch(bds1, daResult1.SubmitMetaData, 1, 10, manager.State.GetProposer().SettlementAddress)

	bds2, err := getBlockDescriptors(batch2)
	require.NoError(t, err)
	slBatch2 := getSLBatch(bds2, daResult2.SubmitMetaData, 11, 20, manager.State.GetProposer().SettlementAddress)

	// Setup mock settlement client
	setupMockSLClient(t, manager, slBatch1, slBatch2)

	// Apply batch1 completely
	err = manager.ApplyBatchFromSL(slBatch1.Batch)
	require.NoError(t, err)
	assert.Equal(t, uint64(10), manager.State.Height())

	// Update LastSettlementHeight to indicate batch2 exists on settlement
	manager.LastSettlementHeight.Store(20)

	// Start SettlementValidateLoop in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var validationErr error
	validationDone := make(chan struct{})

	go func() {
		validationErr = manager.SettlementValidateLoop(ctx)
		close(validationDone)
	}()

	// Apply batch2 blocks slowly via P2P in a goroutine
	applyDone := make(chan struct{})
	go func() {
		defer close(applyDone)
		for i, block := range batch2.Blocks {
			time.Sleep(20 * time.Millisecond)
			blockData := p2p.BlockData{Block: *block, Commit: *batch2.Commits[i]}
			msg := pubsub.NewMessage(blockData, map[string][]string{p2p.EventTypeKey: {p2p.EventNewGossipedBlock}})
			manager.OnReceivedBlock(msg)
			t.Logf("Applied block %d, current height: %d", block.Header.Height, manager.State.Height())
		}
	}()

	// Trigger validation multiple times while blocks are being applied
	// This simulates the race condition
	for i := 0; i < 10; i++ {
		time.Sleep(15 * time.Millisecond)
		manager.TriggerSettlementValidation()
		t.Logf("Triggered validation %d, current height: %d", i, manager.State.Height())
	}

	// Wait for blocks to finish applying
	<-applyDone

	// Trigger validation one final time after all blocks are applied
	manager.TriggerSettlementValidation()
	t.Logf("Final validation trigger after all blocks applied, height: %d", manager.State.Height())

	// Give validation time to complete
	time.Sleep(100 * time.Millisecond)

	// Cancel context and wait for validation loop to exit
	cancel()
	<-validationDone

	// Verify no error occurred (with the fix, validation should skip partial batches and not error)
	// Without the fix, validationErr would contain "not found" errors
	assert.NoError(t, validationErr, "SettlementValidateLoop should not error with the fix in place")

	// Verify all blocks were applied
	assert.Equal(t, uint64(20), manager.State.Height())

	// Verify validation completed successfully for all batches
	lastValidated := manager.SettlementValidator.GetLastValidatedHeight()
	assert.Equal(t, uint64(20), lastValidated, "All batches should be validated after blocks are applied")

	t.Log("Test completed successfully - race condition properly handled")
}

func getBlockDescriptors(batch *types.Batch) ([]rollapp.BlockDescriptor, error) {
	// Create block descriptors
	var bds []rollapp.BlockDescriptor
	for _, block := range batch.Blocks {

		bd := rollapp.BlockDescriptor{
			Height:     block.Header.Height,
			StateRoot:  block.Header.AppHash[:],
			Timestamp:  block.Header.GetTimestamp(),
			DrsVersion: 0,
		}
		bds = append(bds, bd)
	}
	return bds, nil
}

func getSLBatch(bds []rollapp.BlockDescriptor, daMetaData *da.DASubmitMetaData, startHeight uint64, endHeight uint64, nextSequencer string) *settlement.ResultRetrieveBatch {
	// create the batch in settlement
	return &settlement.ResultRetrieveBatch{
		Batch: &settlement.Batch{
			BlockDescriptors: bds,
			MetaData:         daMetaData,
			StartHeight:      startHeight,
			EndHeight:        endHeight,
			NumBlocks:        endHeight - startHeight + 1,
			NextSequencer:    nextSequencer,
		},
		ResultBase: settlement.ResultBase{
			StateIndex: 1,
		},
	}
}

// setupManagerForRaceConditionTest sets up a manager with all required dependencies for race condition testing
func setupManagerForRaceConditionTest(t *testing.T) (*block.Manager, crypto.PrivKey, func()) {
	// Setup app with custom EndBlock response
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

	// Create and start proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(t, err)

	// Generate proposer key
	proposerKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	// Create manager
	manager, err := testutil.GetManagerWithProposerKey(testutil.GetManagerConfig(), proposerKey, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)

	// Setup DA client
	manager.DAClients[da.Mock] = testutil.GetMockDALC(log.TestingLogger())

	cleanup := func() {
		proxyApp.Stop()
	}

	return manager, proposerKey, cleanup
}

// setupMockSLClient creates and configures a mock settlement client with the given batches
func setupMockSLClient(t *testing.T, manager *block.Manager, slBatch1, slBatch2 *settlement.ResultRetrieveBatch) {
	mockSLClient := new(mock_settlement.MockClientI)
	manager.SLClient = mockSLClient

	mockSLClient.On("GetBatchAtHeight", mock.MatchedBy(func(h uint64) bool {
		return h >= 1 && h <= 10
	})).Return(slBatch1, nil)

	mockSLClient.On("GetBatchAtHeight", mock.MatchedBy(func(h uint64) bool {
		return h >= 11 && h <= 20
	})).Return(slBatch2, nil)

	mockSLClient.On("GetRollapp").Return(&types.Rollapp{
		RollappID: "test-rollapp",
		Revisions: []types.Revision{{Number: 0, StartHeight: 1}},
	}, nil)
}
