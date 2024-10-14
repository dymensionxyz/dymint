package block_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/nmt"
	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/store"
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

	celestia "github.com/dymensionxyz/dymint/da/celestia"
	damocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/celestia/types"
)

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
		{
			name:               "Failed validation drs version",
			p2pBlocks:          true,
			stateUpdateFraud:   "drs",
			doubleSignedBlocks: nil,
			expectedErrType:    types.ErrStateUpdateDRSVersionFraud{},
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

			// add drs version to state
			manager.State.AddDRSVersion(0, version.Commit)

			// Create block descriptors
			var bds []rollapp.BlockDescriptor
			for _, block := range batch.Blocks {
				bd := rollapp.BlockDescriptor{
					Height:     block.Header.Height,
					StateRoot:  block.Header.AppHash[:],
					Timestamp:  block.Header.GetTimestamp(),
					DrsVersion: version.Commit,
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

			// Create the StateUpdateValidator
			validator := block.NewStateUpdateValidator(testutil.NewLogger(t), manager)

			// set fraud data
			switch tc.stateUpdateFraud {
			case "drs":
				slBatch.BlockDescriptors[0].DrsVersion = "b306cc32d3ef1782879fdef5e6ab60f270a16817"
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
				manager.ProcessNextDABatch(slBatch.MetaData.DA)
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

func TestStateUpdateValidator_ValidateDAFraud(t *testing.T) {

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

	// Create manager
	manager, err := testutil.GetManagerWithProposerKey(chainId, testutil.GetManagerConfig(), proposerKey, nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Create DA
	// init celestia DA with mock RPC client
	manager.DAClient = registry.GetClient("celestia")

	config := celestia.Config{
		BaseURL:        "http://localhost:26658",
		Timeout:        30 * time.Second,
		GasPrices:      celestia.DefaultGasPrices,
		NamespaceIDStr: "0000000000000000ffff",
	}
	err = config.InitNamespaceID()
	require.NoError(t, err)
	conf, err := json.Marshal(config)
	require.NoError(t, err)

	mockRPCClient := damocks.NewMockCelestiaRPCClient(t)
	options := []da.Option{
		celestia.WithRPCClient(mockRPCClient),
		celestia.WithRPCAttempts(1),
		celestia.WithRPCRetryDelay(time.Second * 2),
	}
	/*roots := [][]byte{[]byte("apple"), []byte("watermelon"), []byte("kiwi")}
	dah := &header.DataAvailabilityHeader{
		RowRoots:    roots,
		ColumnRoots: roots,
	}
	header := &header.ExtendedHeader{
		DAH: dah,
	}*/

	nID := config.NamespaceID.Bytes()
	nIDSize := 1
	tree := exampleNMT(nIDSize, true, 1, 2, 3, 4)
	// build a proof for an NID that is within the namespace range of the tree
	proof, _ := tree.ProveNamespace(nID)
	blobProof := blob.Proof([]*nmt.Proof{&proof})

	mockRPCClient.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint64(1234), nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&blobProof, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("Included", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })

	mockRPCClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
	})

	err = manager.DAClient.Init(conf, nil, store.NewDefaultInMemoryKVStore(), log.TestingLogger(), options...)
	require.NoError(t, err)

	err = manager.DAClient.Start()
	require.NoError(t, err)

	manager.Retriever = manager.DAClient.(da.BatchRetriever)

	// Generate batch
	batch, err := testutil.GenerateBatch(1, 10, proposerKey, chainId, [32]byte{})
	assert.NoError(t, err)

	// Submit batch to DA
	daResultSubmitBatch := manager.DAClient.SubmitBatch(batch)
	assert.Equal(t, daResultSubmitBatch.Code, da.StatusSuccess)

	// add drs version to state
	manager.State.AddDRSVersion(0, version.Commit)

	// Create block descriptors
	var bds []rollapp.BlockDescriptor
	for _, block := range batch.Blocks {
		bd := rollapp.BlockDescriptor{
			Height:     block.Header.Height,
			StateRoot:  block.Header.AppHash[:],
			Timestamp:  block.Header.GetTimestamp(),
			DrsVersion: version.Commit,
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

	// Create the StateUpdateValidator
	validator := block.NewStateUpdateValidator(testutil.NewLogger(t), manager)

	err = validator.ValidateStateUpdate(slBatch)
	require.NoError(t, err)

}

// exampleNMT creates a new NamespacedMerkleTree with the given namespace ID size and leaf namespace IDs. Each byte in the leavesNIDs parameter corresponds to one leaf's namespace ID. If nidSize is greater than 1, the function repeats each NID in leavesNIDs nidSize times before prepending it to the leaf data.
func exampleNMT(nidSize int, ignoreMaxNamespace bool, leavesNIDs ...byte) *nmt.NamespacedMerkleTree {
	tree := nmt.New(sha256.New(), nmt.NamespaceIDSize(nidSize), nmt.IgnoreMaxNamespace(ignoreMaxNamespace))
	for i, nid := range leavesNIDs {
		namespace := bytes.Repeat([]byte{nid}, nidSize)
		d := append(namespace, []byte(fmt.Sprintf("leaf_%d", i))...)
		if err := tree.Push(d); err != nil {
			panic(fmt.Sprintf("unexpected error: %v", err))
		}
	}
	return tree
}
