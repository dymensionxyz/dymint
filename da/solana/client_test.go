package solana_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/da/solana"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/pubsub"

	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/solana"
)

// TestSolanaSubmitRetrieve validates the SubmitBatch and RetrieveBatches method
func TestSolanaSubmitRetrieve(t *testing.T) {

	// init mock
	mockClient, client := setDAandMock(t)

	// generate batch
	block := testutil.GetRandomBlock(1, 1)
	batch := &types.Batch{
		Blocks: []*types.Block{block},
	}

	blob, err := batch.MarshalBinary()
	require.NoError(t, err)

	// mock txhash
	txHash := []string{"txHash"}

	// blobhash
	h := sha256.New()
	h.Write(blob)
	blobHash := h.Sum(nil)
	blobHashString := hex.EncodeToString(blobHash)

	mockClient.On("SubmitBlob", mock.Anything, mock.Anything).Return(txHash, blobHashString, nil)

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

			// generate batch
			block := testutil.GetRandomBlock(1, 1)
			batch := &types.Batch{
				Blocks: []*types.Block{block},
			}

			blob, err := batch.MarshalBinary()
			require.NoError(t, err)

			// blobhash
			h := sha256.New()
			h.Write(blob)
			blobHash := h.Sum(nil)
			blobHashString := hex.EncodeToString(blobHash)

			metadata := solana.SubmitMetaData{
				TxHash:   []string{"txHash"},
				BlobHash: blobHashString,
			}

			if tc.err == nil {
				mockClient.On("GetBlob", mock.Anything, mock.Anything).Return(blob, nil)
			} else {
				mockClient.On("GetBlob", mock.Anything, mock.Anything).Return(nil, da.ErrBlobNotFound)
			}
			// validate avail check
			rValidAvail := retriever.CheckBatchAvailability(metadata.ToPath())
			assert.Equal(t, tc.status, rValidAvail.Code)
			require.ErrorIs(t, rValidAvail.Error, tc.err)

		})
	}

}

func setDAandMock(t *testing.T) (*mocks.MockSolanaClient, da.DataAvailabilityLayerClient) {
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
	dalc := registry.GetClient("solana")

	config := solana.TestConfig

	conf, err := json.Marshal(config)
	require.NoError(err)

	mockRPCClient := mocks.NewMockSolanaClient(t)
	options := []da.Option{
		solana.WithSolanaClient(mockRPCClient),
	}

	err = dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger(), options...)
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	return mockRPCClient, dalc
}
