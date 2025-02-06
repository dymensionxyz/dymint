package avail_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/avail"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	availgo "github.com/availproject/avail-go-sdk/sdk"
	"github.com/tendermint/tendermint/libs/pubsub"

	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/avail"
)

// TestSubmitBatch validates the SubmitBatch method
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
		Seed:        "awel eardsa dfer",
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
