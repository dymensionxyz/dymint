package weave_vm_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/weave_vm"
	weaveVMtypes "github.com/dymensionxyz/dymint/da/weave_vm/types"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
)

// MockWeaveVM is a mock of WeaveVM interface using testify/mock
type MockWeaveVM struct {
	mock.Mock
}

func (m *MockWeaveVM) SendTransaction(ctx context.Context, to string, data []byte) (string, error) {
	args := m.Called(ctx, to, data)
	return args.String(0), args.Error(1)
}

func (m *MockWeaveVM) GetTransactionReceipt(ctx context.Context, txHash string) (*ethtypes.Receipt, error) {
	args := m.Called(ctx, txHash)
	if receipt, ok := args.Get(0).(*ethtypes.Receipt); ok {
		return receipt, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWeaveVM) GetTransactionByHash(ctx context.Context, txHash string) (*ethtypes.Transaction, bool, error) {
	args := m.Called(ctx, txHash)
	tx, _ := args.Get(0).(*ethtypes.Transaction)
	return tx, args.Bool(1), args.Error(2)
}

const (
	testTxHash    = "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	testBlockHash = "0xblockhash"
)

func getTestConfig() weaveVMtypes.Config {
	attempts := 1 // Only try once to avoid retries in tests
	return weaveVMtypes.Config{
		ChainID:       1,
		PrivateKeyHex: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		Endpoint:      "http://localhost:8545",
		Timeout:       1 * time.Second,        // Short timeout for tests
		RetryDelay:    100 * time.Millisecond, // Short retry delay
		RetryAttempts: &attempts,
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name        string
		config      weaveVMtypes.Config
		expectError bool
	}{
		{
			name: "valid config with private key",
			config: weaveVMtypes.Config{
				ChainID:       1,
				PrivateKeyHex: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				Endpoint:      "http://localhost:8545",
			},
			expectError: false,
		},
		{
			name: "valid config with web3signer",
			config: weaveVMtypes.Config{
				ChainID:            1,
				Web3SignerEndpoint: "http://localhost:8545",
				Endpoint:           "http://localhost:8545",
			},
			expectError: false,
		},
		{
			name: "invalid config - no authentication",
			config: weaveVMtypes.Config{
				ChainID:  1,
				Endpoint: "http://localhost:8545",
			},
			expectError: true,
		},
		{
			name: "invalid config - no chain ID",
			config: weaveVMtypes.Config{
				PrivateKeyHex: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				Endpoint:      "http://localhost:8545",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configBytes, err := json.Marshal(tt.config)
			require.NoError(t, err)

			dalc := &weave_vm.DataAvailabilityLayerClient{}
			err = dalc.Init(configBytes, pubsub.NewServer(), store.NewDefaultInMemoryKVStore(), log.TestingLogger())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDALC(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	mockWVM := new(MockWeaveVM)
	dalc, pubsubServer := setupTestDALC(t, mockWVM)
	defer func() {
		err := pubsubServer.Stop()
		require.NoError(err)
	}()

	// Create test data
	block1 := testutil.GetRandomBlock(1, 10)
	batch1 := &types.Batch{
		Blocks: []*types.Block{block1},
	}
	batchData, err := batch1.MarshalBinary()
	require.NoError(err)

	// Mock WeaveVM responses
	blockHash := common.HexToHash(testBlockHash)
	blockNumber := big.NewInt(123)

	mockWVM.On("SendTransaction", mock.Anything, weaveVMtypes.ArchivePoolAddress, mock.Anything).
		Return(testTxHash, nil).Once()

	mockWVM.On("GetTransactionReceipt", mock.Anything, testTxHash).
		Return(&ethtypes.Receipt{
			BlockHash:   blockHash,
			BlockNumber: blockNumber,
			TxHash:      common.HexToHash(testTxHash),
		}, nil).Once()

	// Submit batch
	t.Log("Submitting batch1")
	res1 := dalc.SubmitBatch(batch1)
	h1 := res1.SubmitMetaData
	assert.Equal(da.StatusSuccess, res1.Code)

	// Create test transaction
	tx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Data: batchData,
	})

	mockWVM.On("GetTransactionByHash", mock.Anything, testTxHash).
		Return(tx, false, nil).Once()

	// Retrieve batch
	retrieveRes := dalc.RetrieveBatches(h1)
	assert.Equal(da.StatusSuccess, retrieveRes.Code)
	require.Len(retrieveRes.Batches, 1)
	compareBatches(t, batch1, retrieveRes.Batches[0])

	mockWVM.AssertExpectations(t)
}

func TestRetryBehavior(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	mockWVM := new(MockWeaveVM)
	dalc, pubsubServer := setupTestDALC(t, mockWVM)
	defer func() {
		err := pubsubServer.Stop()
		require.NoError(err)
	}()

	batch := testutil.MustGenerateBatchAndKey(0, 1)

	// Mock temporary failure followed by success
	mockWVM.On("SendTransaction", mock.Anything, weaveVMtypes.ArchivePoolAddress, mock.Anything).
		Return("", errors.New("temporary error")).Once()

	mockWVM.On("SendTransaction", mock.Anything, weaveVMtypes.ArchivePoolAddress, mock.Anything).
		Return(testTxHash, nil).Once()

	mockWVM.On("GetTransactionReceipt", mock.Anything, testTxHash).
		Return(&ethtypes.Receipt{
			BlockHash:   common.HexToHash(testBlockHash),
			BlockNumber: big.NewInt(123),
			TxHash:      common.HexToHash(testTxHash),
		}, nil).Once()

	result := dalc.SubmitBatch(batch)
	assert.Equal(da.StatusSuccess, result.Code)
	assert.NoError(result.Error)

	mockWVM.AssertExpectations(t)
}

func setupTestDALC(t *testing.T, mockWVM weave_vm.WeaveVM) (*weave_vm.DataAvailabilityLayerClient, *pubsub.Server) {
	config := getTestConfig()

	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(t, err)

	dalc := &weave_vm.DataAvailabilityLayerClient{}
	err = dalc.Init(configBytes, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger(),
		weave_vm.WithRPCClient(mockWVM))
	require.NoError(t, err)

	err = dalc.Start()
	require.NoError(t, err)

	return dalc, pubsubServer
}

func compareBatches(t *testing.T, expected, actual *types.Batch) {
	assert := assert.New(t)
	assert.Equal(expected.StartHeight(), actual.StartHeight())
	assert.Equal(expected.EndHeight(), actual.EndHeight())
	assert.Equal(len(expected.Blocks), len(actual.Blocks))
	for i := range expected.Blocks {
		compareBlocks(t, expected.Blocks[i], actual.Blocks[i])
	}
}

func compareBlocks(t *testing.T, expected, actual *types.Block) {
	assert := assert.New(t)
	assert.Equal(expected.Header.Height, actual.Header.Height)
	assert.Equal(expected.Header.Hash(), actual.Header.Hash())
	assert.Equal(expected.Header.AppHash, actual.Header.AppHash)
}
