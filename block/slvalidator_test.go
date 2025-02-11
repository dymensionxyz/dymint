package block_test

import (
	"crypto/rand"
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"github.com/tendermint/tendermint/crypto/ed25519"
	tmtypes "github.com/tendermint/tendermint/types"

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

	fakeProposerKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	nextProposerKey := ed25519.GenPrivKey()
	nextSequencerKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
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
			p2pBlocks:          false,
			stateUpdateFraud:   "nextsequencer",
			doubleSignedBlocks: nil,
			expectedErrType:    &types.ErrInvalidNextSequencersHashFraud{},
			last:               true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Create manager
			manager, err := testutil.GetManagerWithProposerKey(testutil.GetManagerConfig(), proposerKey, nil, 1, 1, 0, proxyApp, nil)
			require.NoError(t, err)
			require.NotNil(t, manager)

			if tc.last {
				proposerPubKey := nextProposerKey.PubKey()
				pubKeybytes := proposerPubKey.Bytes()
				if err != nil {
					panic(err)
				}

				// Set next sequencer
				raw, _ := nextSequencerKey.GetPublic().Raw()
				pubkey := ed25519.PubKey(raw)
				manager.State.SetProposer(types.NewSequencer(pubkey, hex.EncodeToString(pubKeybytes), "", nil))

				// set proposer
				raw, _ = proposerKey.GetPublic().Raw()
				pubkey = ed25519.PubKey(raw)
				manager.State.Proposer.Store(types.NewSequencer(pubkey, "", "", nil))
			}

			// Create DA
			manager.DAClients[da.Mock] = testutil.GetMockDALC(log.TestingLogger())

			// Generate batch
			var batch *types.Batch
			if tc.last {
				batch, err = testutil.GenerateLastBatch(1, 10, proposerKey, fakeProposerKey, [32]byte{})
				assert.NoError(t, err)
			} else {
				batch, err = testutil.GenerateBatch(1, 10, proposerKey, [32]byte{})
				assert.NoError(t, err)
			}

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

			for _, bd := range bds {
				manager.Store.SaveDRSVersion(bd.Height, bd.DrsVersion, nil)
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
				seq := types.NewSequencerFromValidator(*tmtypes.NewValidator(nextProposerKey.PubKey(), 1))
				slBatch.NextSequencer = seq.SettlementAddress
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
				mockDA.MockRPC.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) { time.Sleep(5 * time.Millisecond) })
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

			for _, bd := range bds {
				manager.Store.SaveDRSVersion(bd.Height, bd.DrsVersion, nil)
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
