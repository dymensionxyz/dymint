package kaspa_test

import (
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
	//t.Skip("Skipping Kaspa client test")

	// Set up test environment
	mnemoEnv := "KASPA_MNEMONIC"
	//mnemonic := "broom home badge wrap unveil smoke birth erupt scan merry deny neglect pull select hold winner crouch hazard wear van spell jewel moment actual"
	mnemonic := "seed sun dice artwork mango length sudden trial shove wolf dove during aerobic embark copy border unveil convince cost civil there wrong echo front"
	err := os.Setenv(mnemoEnv, mnemonic)
	require.NoError(t, err)

	// Create test config. By default, tests use Kaspa testnet.
	config := client.TestConfig
	client.TestConfig.FromAddress = "kaspatest:qp75u7cuphjwyq9j6ghe2v0j3gtvxlppyurq279h4ckpdc7umdh6vrusw9c7d"
	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new Aptos client
	client := &kaspa.DataAvailabilityLayerClient{}
	err = client.Init(configBytes, nil, nil, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)

	err = client.Start()
	require.NoError(t, err)
	defer client.Stop()

	// Ensure that the client has enough APT tokens to submit batches
	/*balance, err := client.GetSignerBalance()
	require.NoError(t, err)
	if balance.Amount.LT(math.NewInt(minAPTBalance)) {
		t.Log("Insufficient APT balance for testing: ", balance)
		t.Fail()
	}*/

	// Proposer key for generating test batches
	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		batch   *types.Batch
		wantErr bool
	}{
		{
			name:  "small batch: 1KB",
			batch: testutil.GenerateBatchWithBlocks(1, proposerKey),
		},
		{
			name:  "mid-size batch 1: 20KB",
			batch: testutil.GenerateBatchWithBlocks(40, proposerKey),
		},
		{
			name:  "mid-size batch 2: 48KB",
			batch: testutil.GenerateBatchWithBlocks(80, proposerKey),
		},
		{
			name:  "mid-size batch 2: almost 64KB",
			batch: testutil.GenerateBatchWithBlocks(107, proposerKey),
		},
		// Error: EXCEEDED_MAX_TRANSACTION_SIZE
		{
			name:  "big-size batch 2: 72KB",
			batch: testutil.GenerateBatchWithBlocks(120, proposerKey),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.SubmitBatch(tc.batch)
			if tc.wantErr {
				require.Error(t, result.Error)
				return
			}
			require.NoError(t, result.Error)
			require.Equal(t, da.StatusSuccess, result.Code)

			// Check batch availability
			checkResult := client.CheckBatchAvailability(result.SubmitMetaData.ToPath())
			require.NoError(t, checkResult.Error)
			require.Equal(t, da.StatusSuccess, checkResult.Code)

			// Retrieve batch
			retrieveResult := client.RetrieveBatches(result.SubmitMetaData.ToPath())
			require.NoError(t, retrieveResult.Error)
			require.Equal(t, da.StatusSuccess, retrieveResult.Code)
			require.Len(t, retrieveResult.Batches, 1)

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

	// mock txhash, commitment and proof
	txHash := "txhash"

	mockClient.On("SubmitBlob", mock.Anything, mock.Anything).Return(txHash, nil)

	// generate blob data from batch
	blobData, err := batch.MarshalBinary()
	require.NoError(t, err)

	mockClient.On("GetBlob", mock.Anything, mock.Anything).Return(blobData, nil)

	// submit blob
	rsubmit := client.SubmitBatch(batch)

	retriever := client.(da.BatchRetriever)

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

			metadata := kaspa.SubmitMetaData{
				TxHash: "txhash",
			}

			mockClient.On("GetBlob", mock.Anything, mock.Anything).Return(nil, tc.err)

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

	// init avail DA with mock RPC client
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
