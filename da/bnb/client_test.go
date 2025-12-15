package bnb_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/bnb"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/go-ethereum/common"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/pubsub"

	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/bnb"
)

// TestBNBSubmitRetrieve validates the SubmitBatch and RetrieveBatches method using mock client
func TestBNBSubmitRetrieve(t *testing.T) {
	// init mock
	mockClient, client := setDAandMock(t)

	// generate batch
	block := testutil.GetRandomBlock(1, 1)
	batch := &types.Batch{
		Blocks: []*types.Block{block},
	}

	// mock txhash, commitment and proof
	txHash := common.BytesToHash([]byte("txhash"))
	commitment := []byte("commitment")
	proof := []byte("proof")

	mockClient.On("SubmitBlob", mock.Anything, mock.Anything).Return(txHash, commitment, proof, nil)

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

			metadata := bnb.SubmitMetaData{
				TxHash:     "txhash",
				Proof:      []byte("proof"),
				Commitment: []byte("commitment"),
			}

			mockClient.On("ValidateInclusion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)

			// validate avail check
			rValidAvail := retriever.CheckBatchAvailability(metadata.ToPath())
			assert.Equal(t, tc.status, rValidAvail.Code)
			require.ErrorIs(t, rValidAvail.Error, tc.err)
		})
	}
}

func setDAandMock(t *testing.T) (*mocks.MockBNBClient, da.DataAvailabilityLayerClient) {
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
	dalc := registry.GetClient("bnb")

	// Create temp file with test private key in JSON format
	keyFile, err := os.CreateTemp("", "bnb_test_key")
	require.NoError(err)
	defer os.Remove(keyFile.Name())
	_, err = keyFile.WriteString(`{"private_key": "459c954ea1366473e0edfa4cf55b8028fa6405e44959cfcaa1771890dc527275"}`)
	require.NoError(err)
	keyFile.Close()

	config := bnb.Config{
		BaseConfig: da.BaseConfig{
			Timeout: 5000000000,
		},
		KeyConfig: da.KeyConfig{
			KeyPath: keyFile.Name(),
		},
		Endpoint: "http://localhost:8545/rpc",
		ChainId:  1,
	}

	conf, err := json.Marshal(config)
	require.NoError(err)

	mockRPCClient := mocks.NewMockBNBClient(t)
	options := []da.Option{
		bnb.WithRPCClient(mockRPCClient),
	}

	err = dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger(), options...)
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	return mockRPCClient, dalc
}
