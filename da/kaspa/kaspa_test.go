package kaspa_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/kaspa"
	"github.com/dymensionxyz/dymint/da/kaspa/client"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"

	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/pubsub"

	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/kaspa/client"
)

func TestKaspaDataAvailabilityClient(t *testing.T) {
	t.Skip("Skipping Kaspa client test")

	// Set up test environment
	mnemoEnv := "KASPA_MNEMONIC"
	mnemonic := "broom home badge wrap unveil smoke birth erupt scan merry deny neglect pull select hold winner crouch hazard wear van spell jewel moment actual"
	err := os.Setenv(mnemoEnv, mnemonic)
	require.NoError(t, err)

	// Create test config. By default, tests use Kaspa testnet.
	config := client.TestConfig
	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new Aptos client
	client := &kaspa.DataAvailabilityLayerClient{}
	err = client.Init(configBytes, nil, nil, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)

	err = client.Start()
	require.NoError(t, err)
	defer client.Stop()

	// Proposer key for generating test batches
	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		batch   *types.Batch
		daError error
	}{
		{
			name:  "small batch: 1KB",
			batch: testutil.GenerateBatchWithBlocks(1, proposerKey),
		},
		{
			name:  "mid-size batch: 75KB",
			batch: testutil.GenerateBatchWithBlocks(100, proposerKey),
		},
		{
			name:  "big-size batch: 320KB",
			batch: testutil.GenerateBatchWithBlocks(500, proposerKey),
		},
		{
			name:    "not available batch",
			batch:   testutil.GenerateBatchWithBlocks(1, proposerKey),
			daError: da.ErrBlobNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.SubmitBatch(tc.batch)

			require.NoError(t, result.Error)
			require.Equal(t, da.StatusSuccess, result.Code)

			if tc.daError != nil {
				result.SubmitMetaData.DAPath = "3329c48e787294f956d7ef8c815202683b7b6d93acebe3a2b267a4838ab4fae4|3297d423b9d44ce38a287acf3b43239f0234517811fd60c1e884cb3fba780a5f"
			}
			// Retrieve batch
			retrieveResult := client.RetrieveBatches(result.SubmitMetaData.DAPath)
			if tc.daError != nil {
				assert.ErrorIs(t, retrieveResult.Error, da.ErrBlobNotFound)
			} else {
				require.NoError(t, retrieveResult.Error)
				require.Equal(t, da.StatusSuccess, retrieveResult.Code)
				require.Len(t, retrieveResult.Batches, 1)
			}

			// Check batch availability
			checkResult := client.CheckBatchAvailability(result.SubmitMetaData.DAPath)
			if tc.daError != nil {
				assert.ErrorIs(t, checkResult.Error, da.ErrBlobNotFound)
				return
			}

			require.NoError(t, checkResult.Error)
			require.Equal(t, da.StatusSuccess, checkResult.Code)
			// Compare submitted and retrieved batches
			submittedData, err := tc.batch.MarshalBinary()
			require.NoError(t, err)

			retrievedData, err := retrieveResult.Batches[0].MarshalBinary()
			require.NoError(t, err)

			require.Equal(t, submittedData, retrievedData, "submitted and retrieved batches should be identical")
		})
	}
}

// TestKaspaSubmitRetrieve validates the SubmitBatch and RetrieveBatches method
func TestKaspaSubmitRetrieve(t *testing.T) {

	// init mock
	mockClient, client := setDAandMock(t)

	// generate batch
	block := testutil.GetRandomBlock(1, 1)
	batch := &types.Batch{
		Blocks: []*types.Block{block},
	}

	// mock txhash
	txHash := []string{"txhash"}

	// generate blob data from batch
	blobData, err := batch.MarshalBinary()
	require.NoError(t, err)

	// calculate blobhash
	h := sha256.New()
	h.Write(blobData)
	blobHash := h.Sum(nil)

	mockClient.On("SubmitBlob", mock.Anything, mock.Anything).Return(txHash, hex.EncodeToString(blobHash), nil)
	mockClient.On("GetBlob", mock.Anything, mock.Anything).Return(blobData, nil)
	mockClient.On("GetBlob", mock.Anything, mock.Anything).Return(blobData, nil)

	// submit blob
	rsubmit := client.SubmitBatch(batch)
	assert.Equal(t, da.StatusSuccess, rsubmit.Code)
	retriever := client.(da.BatchRetriever)

	// check availability
	ravail := client.CheckBatchAvailability(rsubmit.SubmitMetaData.DAPath)
	assert.Equal(t, da.StatusSuccess, ravail.Code)

	// validate retrieval
	rretrieve := retriever.RetrieveBatches(rsubmit.SubmitMetaData.DAPath)
	assert.Equal(t, da.StatusSuccess, rretrieve.Code)
	require.True(t, len(rretrieve.Batches) == 1)
	assert.Equal(t, batch.Blocks[0], rretrieve.Batches[0].Blocks[0])

}

func TestKaspaAvailCheck(t *testing.T) {

	testCases := []struct {
		name   string
		err    error
		status da.StatusCode
	}{
		{
			name:   "available",
			err:    nil,
			status: da.StatusSuccess,
		},
		{

			name:   "not available",
			err:    da.ErrBlobNotFound,
			status: da.StatusError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// init mock
			mockClient, client := setDAandMock(t)
			retriever := client.(da.BatchRetriever)

			// generate batch
			block := testutil.GetRandomBlock(1, 1)
			batch := &types.Batch{
				Blocks: []*types.Block{block},
			}

			// generate blob data from batch
			blobData, err := batch.MarshalBinary()
			require.NoError(t, err)

			// calculate blobhash
			h := sha256.New()
			h.Write(blobData)
			blobHash := h.Sum(nil)

			metadata := kaspa.SubmitMetaData{
				TxHash:   []string{"txhash"},
				BlobHash: hex.EncodeToString(blobHash),
			}

			mockClient.On("GetBlob", mock.Anything, mock.Anything).Return(blobData, tc.err)

			// validate avail check
			rValidAvail := retriever.CheckBatchAvailability(metadata.ToPath())
			assert.Equal(t, tc.status, rValidAvail.Code)
			require.ErrorIs(t, rValidAvail.Error, tc.err)

		})
	}

}

func setDAandMock(t *testing.T) (*mocks.MockKaspaClient, da.DataAvailabilityLayerClient) {
	var err error
	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(t, err)
	defer func() {
		err = pubsubServer.Stop()
		require.NoError(t, err)
	}()

	require := require.New(t)

	// init kaspa DA with mock client
	dalc := registry.GetClient("kaspa")

	config := client.TestConfig
	conf, err := json.Marshal(config)
	require.NoError(err)

	mockKaspaClient := mocks.NewMockKaspaClient(t)
	options := []da.Option{
		kaspa.WithKaspaClient(mockKaspaClient),
	}

	err = dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger(), options...)
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	return mockKaspaClient, dalc
}
