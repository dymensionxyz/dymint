package bnb_test

import (
	"encoding/json"
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

// TestBNB validates the SubmitBatch and RetrieveBatches method
func TestBNBSubmitRetrieve(t *testing.T) {
	mockClient, client := setDAandMock(t)

	block1 := testutil.GetRandomBlock(1, 1)
	batch1 := &types.Batch{
		Blocks: []*types.Block{block1},
	}

	txHash := common.BytesToHash([]byte("txhash"))
	commitment := []byte("commitment")
	proof := []byte("proof")

	mockClient.On("SubmitBlob", mock.Anything, mock.Anything).Return(txHash, commitment, proof, nil)

	data1, err := batch1.MarshalBinary()
	require.NoError(t, err)

	mockClient.On("GetBlob", mock.Anything, mock.Anything).Return(data1, nil)

	rsubmit := client.SubmitBatch(batch1)

	retriever := client.(da.BatchRetriever)

	rretrieve := retriever.RetrieveBatches(rsubmit.SubmitMetaData.DAPath)
	assert.Equal(t, da.StatusSuccess, rretrieve.Code)
	require.True(t, len(rretrieve.Batches) == 1)
	assert.Equal(t, batch1.Blocks[0], rretrieve.Batches[0].Blocks[0])
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

	config := bnb.BNBConfig{
		Timeout:    5000000000,
		Endpoint:   "http://localhost:8545/rpc",
		ChainId:    1,
		PrivateKey: "459c954ea1366473e0edfa4cf55b8028fa6405e44959cfcaa1771890dc527275",
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
