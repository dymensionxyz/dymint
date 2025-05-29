package eth_test

import (
	"encoding/json"
	"testing"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/eth"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/pubsub"

	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/eth"
)

// TestEthSubmitRetrieve validates the SubmitBatch and RetrieveBatches method using mock client
func TestEthSubmitRetrieve(t *testing.T) {

	// init mock
	mockClient, client := setDAandMock(t)

	// generate batch
	block := testutil.GetRandomBlock(1, 1)
	batch := &types.Batch{
		Blocks: []*types.Block{block},
	}

	// mock commitment and proof
	commitment := []byte("commitment")
	proof := []byte("proof")
	slot := "1"
	mockClient.On("SubmitBlob", mock.Anything, mock.Anything).Return(commitment, proof, slot, nil)
	mockClient.On("ValidateInclusion", mock.Anything, mock.Anything, mock.Anything).Return(nil)

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

// TestAvailCheck tests CheckAvailability function using mock client
func TestAvailCheck(t *testing.T) {

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

			metadata := eth.SubmitMetaData{
				Proof:      []byte("proof"),
				Commitment: []byte("commitment"),
				Slot:       "1",
			}

			mockClient.On("ValidateInclusion", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)

			// validate avail check
			rValidAvail := retriever.CheckBatchAvailability(metadata.ToPath())
			assert.Equal(t, tc.status, rValidAvail.Code)
			require.ErrorIs(t, rValidAvail.Error, tc.err)

		})
	}

}
func setDAandMock(t *testing.T) (*mocks.MockEthClient, da.DataAvailabilityLayerClient) {
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
	dalc := registry.GetClient("eth")

	config := eth.TestConfig

	conf, err := json.Marshal(config)
	require.NoError(err)

	mockRPCClient := mocks.NewMockEthClient(t)
	options := []da.Option{
		eth.WithRPCClient(mockRPCClient),
	}

	err = dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger(), options...)
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	return mockRPCClient, dalc
}
