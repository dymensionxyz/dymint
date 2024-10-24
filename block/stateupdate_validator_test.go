package block_test

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/da"
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

func TestSettlementValidator_ValidateStateUpdate(t *testing.T) {

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
		expectedErrType    interface{}
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
			expectedErrType:    types.ErrStateUpdateDoubleSigningFraud{},
		},
		{
			name:               "Failed validation wrong state roots",
			p2pBlocks:          true,
			stateUpdateFraud:   "stateroot",
			doubleSignedBlocks: nil,
			expectedErrType:    types.ErrStateUpdateStateRootNotMatchingFraud{},
		},
		{
			name:               "Failed validation batch num blocks",
			p2pBlocks:          true,
			stateUpdateFraud:   "batchnumblocks",
			doubleSignedBlocks: nil,
			expectedErrType:    types.ErrStateUpdateNumBlocksNotMatchingFraud{},
		},
		{
			name:               "Failed validation batch num bds",
			p2pBlocks:          true,
			stateUpdateFraud:   "batchnumbds",
			doubleSignedBlocks: nil,
			expectedErrType:    types.ErrStateUpdateNumBlocksNotMatchingFraud{},
		},
		{
			name:               "Failed validation wrong timestamps",
			p2pBlocks:          true,
			stateUpdateFraud:   "timestamp",
			doubleSignedBlocks: doubleSigned,
			expectedErrType:    types.ErrStateUpdateTimestampNotMatchingFraud{},
		},
		{
			name:               "Failed validation wrong height",
			p2pBlocks:          true,
			stateUpdateFraud:   "height",
			doubleSignedBlocks: doubleSigned,
			expectedErrType:    types.ErrStateUpdateHeightNotMatchingFraud{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Create manager
			manager, err := testutil.GetManagerWithProposerKey(testutil.GetManagerConfig(), proposerKey, nil, 1, 1, 0, proxyApp, nil)
			require.NoError(t, err)
			require.NotNil(t, manager)

			// Create DA
			manager.DAClient = testutil.GetMockDALC(log.TestingLogger())
			manager.Retriever = manager.DAClient.(da.BatchRetriever)

			// Generate batch
			batch, err := testutil.GenerateBatch(1, 10, proposerKey, [32]byte{})
			assert.NoError(t, err)

			// Submit batch to DA
			daResultSubmitBatch := manager.DAClient.SubmitBatch(batch)
			assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)

			// Create block descriptors
			var bds []rollapp.BlockDescriptor
			for _, block := range batch.Blocks {
				bd := rollapp.BlockDescriptor{
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
					NumBlocks:   10,
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			}

			// Create the SettlementValidator
			validator := block.NewSettlementValidator(testutil.NewLogger(t), manager)

			// set fraud data
			switch tc.stateUpdateFraud {
			case "batchnumblocks":
				slBatch.NumBlocks = 11
			case "batchnumbds":
				bds = append(bds, rollapp.BlockDescriptor{})
				slBatch.BlockDescriptors = bds
			case "stateroot":
				slBatch.BlockDescriptors[0].StateRoot = []byte{}
			case "timestamp":
				slBatch.BlockDescriptors[0].Timestamp = slBatch.BlockDescriptors[0].Timestamp.Add(time.Second)
			case "height":
				slBatch.BlockDescriptors[0].Height = 2
			}

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
				manager.ApplyBatchFromSL(slBatch.MetaData.DA)
			}

			// validate the state update
			err = validator.ValidateStateUpdate(slBatch)

			// Check the result
			if tc.expectedErrType == nil {
				assert.NoError(t, err)
			} else {
				assert.True(t, errors.As(err, &tc.expectedErrType),
					"expected error of type %T, got %T", tc.expectedErrType, err)
			}

		})
	}

}
