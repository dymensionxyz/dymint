package avail_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/avail"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	availgo "github.com/availproject/avail-go-sdk/sdk"
	"github.com/tendermint/tendermint/libs/pubsub"

	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/avail"
)

func TestDataAvailabilityLayerClient(t *testing.T) {
	t.Skip("Skipping Avail client test")

	// Create test config with mnemonic directly
	config := avail.TestConfig
	config.KeyConfig.Mnemonic = "plug mandate gossip deposit reduce civil lawn extra fantasy grow increase off"
	config.KeyConfig.MnemonicPath = "" // Clear the path since we're using direct mnemonic

	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new Avail client
	client := &avail.DataAvailabilityLayerClient{}
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.SubmitBatch(tc.batch)

			require.NoError(t, result.Error)
			require.Equal(t, da.StatusSuccess, result.Code)

			// Retrieve batch
			retrieveResult := client.RetrieveBatches(result.SubmitMetaData.DAPath)
			if tc.daError != nil {
				assert.ErrorIs(t, retrieveResult.Error, da.ErrBlobNotFound)
			} else {
				require.NoError(t, retrieveResult.Error)
				require.Equal(t, da.StatusSuccess, retrieveResult.Code)
				require.Len(t, retrieveResult.Batches, 1)
			}

			// Compare submitted and retrieved batches
			submittedData, err := tc.batch.MarshalBinary()
			require.NoError(t, err)

			retrievedData, err := retrieveResult.Batches[0].MarshalBinary()
			require.NoError(t, err)

			require.Equal(t, submittedData, retrievedData, "submitted and retrieved batches should be identical")
		})
	}
}

// TestAvail validates the SubmitBatch and RetrieveBatches method
func TestAvail(t *testing.T) {
	mockClient, client := setDAandMock(t)

	block1 := testutil.GetRandomBlock(1, 1)
	batch1 := &types.Batch{
		Blocks: []*types.Block{block1},
	}

	mockClient.On("SubmitData", mock.Anything, mock.Anything).Return("blockhash", nil)
	mockClient.On("GetAccountAddress", mock.Anything).Return("address")

	rsubmit := client.SubmitBatch(batch1)

	assert.Equal(t, da.StatusSuccess, rsubmit.Code)
	assert.NotNil(t, rsubmit.SubmitMetaData)

	data1, err := batch1.MarshalBinary()
	require.NoError(t, err)
	getResult := []availgo.DataSubmission{
		{
			Data: data1,
		},
	}
	mockClient.On("GetBlobsBySigner", mock.Anything, mock.Anything, mock.Anything).Return(getResult, nil).Run(func(args mock.Arguments) { time.Sleep(5 * time.Millisecond) })

	retriever := client.(da.BatchRetriever)

	rretrieve := retriever.RetrieveBatches(rsubmit.SubmitMetaData.DAPath)
	assert.Equal(t, da.StatusSuccess, rretrieve.Code)
	require.True(t, len(rretrieve.Batches) == 1)
	assert.Equal(t, batch1.Blocks[0], rretrieve.Batches[0].Blocks[0])
}

func setDAandMock(t *testing.T) (*mocks.MockAvailClient, da.DataAvailabilityLayerClient) {
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
	dalc := registry.GetClient("avail")

	config := avail.Config{
		KeyConfig: da.KeyConfig{
			Mnemonic: "awel eardsa dfer",
		},
		RpcEndpoint: "http://localhost:26658/rpc",
		AppID:       1,
	}

	conf, err := json.Marshal(config)
	require.NoError(err)

	mockRPCClient := mocks.NewMockAvailClient(t)
	options := []da.Option{
		avail.WithRPCClient(mockRPCClient),
	}

	err = dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger(), options...)
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	return mockRPCClient, dalc
}
