package block_test

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
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

func TestStateUpdateValidator_ValidateP2PBlocks(t *testing.T) {
	validator := &block.StateUpdateValidator{}

	proposerKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	batch, err := testutil.GenerateBatch(1, 10, proposerKey, "test", [32]byte{})
	require.NoError(t, err)

	doubleSignedBatch, err := testutil.GenerateBatch(1, 10, proposerKey, "test", [32]byte{})
	require.NoError(t, err)

	mixedBatch := make([]*types.Block, 10)
	copy(mixedBatch, batch.Blocks)
	mixedBatch[2] = doubleSignedBatch.Blocks[2]

	tests := []struct {
		name      string
		daBlocks  []*types.Block
		p2pBlocks []*types.Block
		wantErr   bool
	}{
		{
			name:      "Empty blocks",
			daBlocks:  []*types.Block{},
			p2pBlocks: []*types.Block{},
			wantErr:   false,
		},
		{
			name:      "Matching blocks",
			daBlocks:  batch.Blocks,
			p2pBlocks: batch.Blocks,
			wantErr:   false,
		},
		{
			name:      "double signing",
			daBlocks:  batch.Blocks,
			p2pBlocks: doubleSignedBatch.Blocks,
			wantErr:   true,
		},
		{
			name:      "mixed blocks",
			daBlocks:  batch.Blocks,
			p2pBlocks: mixedBatch,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateP2PBlocks(tt.daBlocks, tt.p2pBlocks)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStateUpdateValidator_ValidateDaBlocks(t *testing.T) {

	validator := &block.StateUpdateValidator{}

	proposerKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	batch, err := testutil.GenerateBatch(1, 2, proposerKey, "test", [32]byte{})
	require.NoError(t, err)

	tests := []struct {
		name          string
		slBatch       *settlement.ResultRetrieveBatch
		daBlocks      []*types.Block
		expectedError error
	}{
		{
			name: "Happy path - all validations pass",
			slBatch: &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: []settlement.BlockDescriptor{
						{Height: 1, StateRoot: batch.Blocks[0].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
						{Height: 2, StateRoot: batch.Blocks[1].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
					},
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			},
			daBlocks:      batch.Blocks,
			expectedError: nil,
		},
		{
			name: "Error - number of blocks mismatch",
			slBatch: &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: []settlement.BlockDescriptor{
						{Height: 1, StateRoot: batch.Blocks[0].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
						{Height: 2, StateRoot: batch.Blocks[1].Header.AppHash[:], Timestamp: batch.Blocks[1].Header.GetTimestamp()},
					},
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			},
			daBlocks:      []*types.Block{batch.Blocks[0]},
			expectedError: fmt.Errorf("num blocks mismatch between state update and DA batch. State index: 1 State update blocks: 2 DA batch blocks: 1"),
		},
		{
			name: "Error - height mismatch",
			slBatch: &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: []settlement.BlockDescriptor{
						{Height: 101, StateRoot: batch.Blocks[0].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
						{Height: 102, StateRoot: batch.Blocks[1].Header.AppHash[:], Timestamp: batch.Blocks[1].Header.GetTimestamp()},
					},
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			},
			daBlocks:      batch.Blocks,
			expectedError: fmt.Errorf("height mismatch between state update and DA batch. State index: 1 SL height: 101 DA height: 1"),
		},
		{
			name: "Error - state root mismatch",
			slBatch: &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: []settlement.BlockDescriptor{
						{Height: 1, StateRoot: batch.Blocks[0].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
						{Height: 2, StateRoot: []byte{1, 2, 3, 4}, Timestamp: batch.Blocks[1].Header.GetTimestamp()},
					},
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			},
			daBlocks:      batch.Blocks,
			expectedError: fmt.Errorf("state root mismatch between state update and DA batch. State index: 1: Height: 2 State root SL: %d State root DA: %d", []byte{1, 2, 3, 4}, batch.Blocks[0].Header.AppHash[:]),
		},
		{
			name: "Error - timestamp mismatch",
			slBatch: &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: []settlement.BlockDescriptor{
						{Height: 1, StateRoot: batch.Blocks[0].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
						{Height: 2, StateRoot: batch.Blocks[1].Header.AppHash[:], Timestamp: batch.Blocks[1].Header.GetTimestamp().Add(1 * time.Second)},
					},
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			},
			daBlocks: batch.Blocks,
			expectedError: fmt.Errorf("timestamp mismatch between state update and DA batch. State index: 1: Height: 2 Timestamp SL: %s Timestamp DA: %s",
				batch.Blocks[1].Header.GetTimestamp().UTC().Add(1*time.Second), batch.Blocks[1].Header.GetTimestamp().UTC()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDaBlocks(tt.slBatch, tt.daBlocks)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStateUpdateValidator_ValidateStateUpdate(t *testing.T) {

	// Init app
	app := testutil.GetAppMock(testutil.EndBlock)
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:      "mock",
			Version: version.Commit,
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
	chainId := "test"
	proposerKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	doubleSigned, err := testutil.GenerateBlocks(1, 10, proposerKey, chainId, [32]byte{})
	require.NoError(t, err)

	// Test cases
	testCases := []struct {
		name               string
		p2pBlocks          bool
		doubleSignedBlocks []*types.Block
		stateUpdateFraud   string
		expectedError      error
	}{
		{
			name:               "Successful validation applied from DA",
			p2pBlocks:          false,
			doubleSignedBlocks: nil,
			stateUpdateFraud:   "",
			expectedError:      nil,
		},
		{
			name:               "Successful validation applied from P2P",
			p2pBlocks:          true,
			doubleSignedBlocks: nil,
			stateUpdateFraud:   "",
			expectedError:      nil,
		},
		{
			name:               "Failed validation blocks not matching",
			p2pBlocks:          true,
			stateUpdateFraud:   "",
			doubleSignedBlocks: doubleSigned,
			expectedError:      fmt.Errorf("p2p block different from DA block. p2p height: 1, DA height: 1"),
		},
		{
			name:               "Failed validation wrong state roots",
			p2pBlocks:          true,
			stateUpdateFraud:   "stateroot",
			doubleSignedBlocks: doubleSigned,
			expectedError:      fmt.Errorf("state root mismatch between state update and DA batch. State index: 1: Height: 1 State root SL: [] State root DA: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]"),
		},
		{
			name:               "Failed validation wrong timestamps",
			p2pBlocks:          true,
			stateUpdateFraud:   "timestamp",
			doubleSignedBlocks: doubleSigned,
			expectedError:      fmt.Errorf("timestamp mismatch between state update and DA batch. State index: 1: Height: 1 Timestamp SL: 1970-01-01 00:00:01.000004567 +0000 UTC Timestamp DA: 1970-01-01 00:00:00.000004567 +0000 UTC"),
		},
		{
			name:               "Failed validation wrong height",
			p2pBlocks:          true,
			stateUpdateFraud:   "height",
			doubleSignedBlocks: doubleSigned,
			expectedError:      fmt.Errorf("height mismatch between state update and DA batch. State index: 1 SL height: 2 DA height: 1"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Create manager
			manager, err := testutil.GetManagerWithProposerKey(chainId, testutil.GetManagerConfig(), proposerKey, nil, 1, 1, 0, proxyApp, nil)
			require.NoError(t, err)
			require.NotNil(t, manager)

			// Create DA
			manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
			manager.Retriever = manager.DAClient.(da.BatchRetriever)

			// Generate batch
			batch, err := testutil.GenerateBatch(1, 10, proposerKey, chainId, [32]byte{})
			assert.NoError(t, err)

			// Submit batch to DA
			daResultSubmitBatch := manager.DAClient.SubmitBatch(batch)
			assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)

			// Create block descriptors
			var bds []settlement.BlockDescriptor
			for _, block := range batch.Blocks {
				bd := settlement.BlockDescriptor{
					Height:    block.Header.Height,
					StateRoot: block.Header.AppHash[:],
					Timestamp: block.Header.GetTimestamp(),
				}
				bds = append(bds, bd)
			}

			// create the batch in settlement
			slBatch := &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: bds,
					MetaData: &settlement.BatchMetaData{
						DA: daResultSubmitBatch.SubmitMetaData,
					},
					StartHeight: 1,
					EndHeight:   10,
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			}
			switch tc.stateUpdateFraud {
			case "stateroot":
				slBatch.BlockDescriptors[0].StateRoot = []byte{}
			case "timestamp":
				slBatch.BlockDescriptors[0].Timestamp = slBatch.BlockDescriptors[0].Timestamp.Add(time.Second)
			case "height":
				slBatch.BlockDescriptors[0].Height = 2
			}
			// Create the StateUpdateValidator
			validator := block.NewStateUpdateValidator(testutil.NewLogger(t), manager)

			if tc.doubleSignedBlocks != nil {
				batch.Blocks = tc.doubleSignedBlocks
				batch.Commits, err = testutil.GenerateCommits(batch.Blocks, proposerKey)
				require.NoError(t, err)
			}

			if tc.p2pBlocks {
				for i, block := range batch.Blocks {
					blockData := p2p.BlockData{Block: *block, Commit: *batch.Commits[i]}
					msg := pubsub.NewMessage(blockData, map[string][]string{p2p.EventTypeKey: {p2p.EventNewGossipedBlock}})
					manager.OnReceivedBlock(msg)
				}
			} else {
				manager.ProcessNextDABatch(slBatch.MetaData.DA)
			}
			// Call the function
			err = validator.ValidateStateUpdate(slBatch)

			// Check the result
			if tc.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError.Error())
			}

		})
	}

}
