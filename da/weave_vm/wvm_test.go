package weave_vm_test

import (
	"context"
	cryptoRand "crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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
	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

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

func setupTestDALC(t *testing.T, mockWVM *MockWeaveVM) (*weave_vm.DataAvailabilityLayerClient, *pubsub.Server) {
	t.Helper()

	cfg := getTestConfig()
	configBytes, err := json.Marshal(cfg)
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

	t.Cleanup(func() {
		require.NoError(t, pubsubServer.Stop())
		mockWVM.AssertExpectations(t)
	})

	return dalc, pubsubServer
}

// compareBatches is a helper to compare expected vs. actual batches.
func compareBatches(t *testing.T, expected, actual *types.Batch) {
	assert.Equal(t, expected.StartHeight(), actual.StartHeight())
	assert.Equal(t, expected.EndHeight(), actual.EndHeight())
	assert.Equal(t, len(expected.Blocks), len(actual.Blocks))
	for i := range expected.Blocks {
		compareBlocks(t, expected.Blocks[i], actual.Blocks[i])
	}
}

// compareBlocks is a helper to compare expected vs. actual blocks.
func compareBlocks(t *testing.T, expected, actual *types.Block) {
	assert.Equal(t, expected.Header.Height, actual.Header.Height)
	assert.Equal(t, expected.Header.Hash(), actual.Header.Hash())
	assert.Equal(t, expected.Header.AppHash, actual.Header.AppHash)
}

// TestInit checks different config initialization scenarios with sub-tests.
func TestInit(t *testing.T) {
	t.Run("ValidConfigWithPrivateKey", func(t *testing.T) {
		config := weaveVMtypes.Config{
			ChainID:       1,
			PrivateKeyHex: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Endpoint:      "http://localhost:8545",
		}
		configBytes, err := json.Marshal(config)
		require.NoError(t, err)

		dalc := &weave_vm.DataAvailabilityLayerClient{}
		err = dalc.Init(configBytes, pubsub.NewServer(), store.NewDefaultInMemoryKVStore(), log.TestingLogger())
		require.NoError(t, err)
	})

	t.Run("ValidConfigWithWeb3Signer", func(t *testing.T) {
		config := weaveVMtypes.Config{
			ChainID:            1,
			Web3SignerEndpoint: "http://localhost:8545",
			Endpoint:           "http://localhost:8545",
		}
		configBytes, err := json.Marshal(config)
		require.NoError(t, err)

		dalc := &weave_vm.DataAvailabilityLayerClient{}
		err = dalc.Init(configBytes, pubsub.NewServer(), store.NewDefaultInMemoryKVStore(), log.TestingLogger())
		require.NoError(t, err)
	})

	t.Run("InvalidConfigNoAuth", func(t *testing.T) {
		config := weaveVMtypes.Config{
			ChainID:  1,
			Endpoint: "http://localhost:8545",
		}
		configBytes, err := json.Marshal(config)
		require.NoError(t, err)

		dalc := &weave_vm.DataAvailabilityLayerClient{}
		err = dalc.Init(configBytes, pubsub.NewServer(), store.NewDefaultInMemoryKVStore(), log.TestingLogger())
		require.Error(t, err)
	})

	t.Run("InvalidConfigNoChainID", func(t *testing.T) {
		config := weaveVMtypes.Config{
			PrivateKeyHex: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Endpoint:      "http://localhost:8545",
		}
		configBytes, err := json.Marshal(config)
		require.NoError(t, err)

		dalc := &weave_vm.DataAvailabilityLayerClient{}
		err = dalc.Init(configBytes, pubsub.NewServer(), store.NewDefaultInMemoryKVStore(), log.TestingLogger())
		require.Error(t, err)
	})
}

func getTestConfig() weaveVMtypes.Config {
	attempts := 1 // Single attempt for tests to simplify validation
	return weaveVMtypes.Config{
		ChainID:       1,
		PrivateKeyHex: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		Endpoint:      "http://localhost:8545",
		Timeout:       1 * time.Second,
		RetryDelay:    100 * time.Millisecond,
		RetryAttempts: &attempts,
		// Disable backoff for predictable behavior in tests
		Backoff: uretry.BackoffConfig{
			InitialDelay: 1 * time.Millisecond,
			MaxDelay:     1 * time.Millisecond,
			GrowthFactor: 1.0,
		},
	}
}

func TestSubmitBatch(t *testing.T) {
	mockWVM := new(MockWeaveVM)
	dalc, _ := setupTestDALC(t, mockWVM)

	testCases := []struct {
		name        string
		setupMocks  func()
		createBatch func() *types.Batch
		expectCode  da.StatusCode
		expectError string
	}{
		{
			name: "Submit Batch",
			setupMocks: func() {
				// Setup successful submission
				mockWVM.On("SendTransaction", mock.Anything, weaveVMtypes.ArchivePoolAddress, mock.Anything).
					Return(testTxHash, nil).Once()

				// Setup successful receipt
				mockWVM.On("GetTransactionReceipt", mock.Anything, testTxHash).
					Return(&ethtypes.Receipt{
						Status:      1,
						BlockHash:   common.HexToHash(testBlockHash),
						BlockNumber: big.NewInt(123),
						TxHash:      common.HexToHash(testTxHash),
					}, nil).Once()
			},
			createBatch: func() *types.Batch {
				block := getRandomBlock(1, 10)
				return &types.Batch{Blocks: []*types.Block{block}}
			},
			expectCode: da.StatusSuccess,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mock expectations
			mockWVM.ExpectedCalls = nil
			mockWVM.Calls = nil

			// Setup test case mocks
			tc.setupMocks()

			// Execute test
			batch := tc.createBatch()
			res := dalc.SubmitBatch(batch)

			// Verify results
			assert.Equal(t, tc.expectCode, res.Code)
			if tc.expectError != "" {
				assert.Contains(t, res.Error.Error(), tc.expectError)
			} else {
				submitMD := res.SubmitMetaData
				assert.NotNil(t, submitMD)
				assert.Equal(t, da.WeaveVM, submitMD.Client)
				assert.Equal(t, uint64(123), submitMD.Height)
			}

			// Verify all expected mock calls were made
			mockWVM.AssertExpectations(t)
		})
	}
}

func TestRetrieveBatches(t *testing.T) {
	mockWVM := new(MockWeaveVM)
	dalc, _ := setupTestDALC(t, mockWVM)

	batch := &types.Batch{
		Blocks: []*types.Block{getRandomBlock(1, 10)},
	}
	batchData, _ := batch.MarshalBinary()

	testCases := []struct {
		name            string
		setupMocks      func()
		submitMeta      *da.DASubmitMetaData
		expectCode      da.StatusCode
		expectError     string
		validateBatches func(t *testing.T, batches []*types.Batch)
	}{
		{
			name: "Successful Retrieval",
			setupMocks: func() {
				// Mock GetTransactionByHash to return valid data
				tx := ethtypes.NewTx(&ethtypes.LegacyTx{Data: batchData})
				mockWVM.On("GetTransactionByHash", mock.Anything, testTxHash).
					Return(tx, false, nil).Once()
			},
			submitMeta: &da.DASubmitMetaData{
				Client:     da.WeaveVM,
				Height:     123,
				WvmTxHash:  testTxHash,
				Commitment: crypto.Keccak256(batchData),
			},
			expectCode: da.StatusSuccess,
			validateBatches: func(t *testing.T, batches []*types.Batch) {
				require.Len(t, batches, 1)
				block := batches[0].Blocks[0]
				assert.Equal(t, uint64(1), block.Header.Height)
			},
		},
		{
			name: "Non-Existent Transaction",
			setupMocks: func() {
				// Mock GetTransactionByHash to return an error
				mockWVM.On("GetTransactionByHash", mock.Anything, testTxHash).
					Return(nil, false, errors.New("retrieval failed")).Once()
			},
			submitMeta: &da.DASubmitMetaData{
				Client:     da.WeaveVM,
				Height:     123,
				WvmTxHash:  testTxHash,
				Commitment: crypto.Keccak256(batchData),
			},
			expectCode:  da.StatusError,
			expectError: da.ErrRetrieval.Error(),
		},
		{
			name: "Malformed Batch Data",
			setupMocks: func() {
				// Mock GetTransactionByHash to return corrupted data
				tx := ethtypes.NewTx(&ethtypes.LegacyTx{Data: []byte("corrupted data")})
				mockWVM.On("GetTransactionByHash", mock.Anything, testTxHash).
					Return(tx, false, nil).Once()
			},
			submitMeta: &da.DASubmitMetaData{
				Client:     da.WeaveVM,
				Height:     123,
				WvmTxHash:  testTxHash,
				Commitment: crypto.Keccak256(batchData),
			},
			expectCode:  da.StatusError,
			expectError: da.ErrProofNotMatching.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockWVM.ExpectedCalls = nil
			mockWVM.Calls = nil

			tc.setupMocks()

			res := dalc.RetrieveBatches(tc.submitMeta)

			assert.Equal(t, tc.expectCode, res.Code)
			if tc.expectError != "" {
				assert.Contains(t, res.Error.Error(), tc.expectError)
			} else if tc.validateBatches != nil {
				tc.validateBatches(t, res.Batches)
			}

			mockWVM.AssertExpectations(t)
		})
	}
}

// Utility for creating batches directly from blocks
func ToBatch(b *types.Block) *types.Batch {
	return &types.Batch{Blocks: []*types.Block{b}}
}

// TestRetryBehavior verifies we retry if SendTransaction fails the first time.
func TestRetryBehavior(t *testing.T) {
	mockWVM := new(MockWeaveVM)
	dalc, _ := setupTestDALC(t, mockWVM)

	batch := testutil.MustGenerateBatchAndKey(0, 1)

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
	require.Equal(t, da.StatusSuccess, result.Code)
	require.NoError(t, result.Error)
}

// Generates a random block with a given height and number of transactions
func getRandomBlock(height uint64, nTxs int) *types.Block {
	block := &types.Block{
		Header: types.Header{
			Height:                height,
			ConsensusMessagesHash: types.ConsMessagesHash(nil),
		},
		Data: types.Data{
			Txs: make(types.Txs, nTxs),
		},
	}
	copy(block.Header.AppHash[:], getRandomBytes(32))

	for i := 0; i < nTxs; i++ {
		block.Data.Txs[i] = getRandomTx()
	}

	if nTxs == 0 {
		block.Data.Txs = nil
	}

	return block
}

// Generates a random transaction
func getRandomTx() types.Tx {
	size := rand.Int()%100 + 100
	return types.Tx(getRandomBytes(size))
}

// Generates a random byte slice of the given length
func getRandomBytes(n int) []byte {
	data := make([]byte, n)
	_, _ = cryptoRand.Read(data)
	return data
}
