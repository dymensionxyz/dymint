package celestia_test

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/tendermint/tendermint/libs/log"

	mocks "github.com/dymensionxyz/dymint/mocks/da/celestia"
	"github.com/rollkit/celestia-openrpc/types/state"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	damock "github.com/dymensionxyz/dymint/da/mock"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

const mockDaBlockTime = 100 * time.Millisecond

func TestDALC(t *testing.T) {
	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	defer pubsubServer.Stop()

	require := require.New(t)
	assert := assert.New(t)

	//init mock DA
	mockdlc := damock.DataAvailabilityLayerClient{}
	mockconf := []byte(mockDaBlockTime.String())
	err := mockdlc.Init(mockconf, nil, store.NewDefaultInMemoryKVStore(), log.TestingLogger())
	require.NoError(err)

	err = mockdlc.Start()
	require.NoError(err)

	//init celestia DA with mock RPC client
	dalc := registry.GetClient("celestia")
	config := celestia.Config{
		BaseURL:  "http://localhost:26658",
		Timeout:  30 * time.Second,
		GasLimit: 3000000,
		Fee:      200000000,
	}
	err = config.InitNamespaceID()
	require.NoError(err)
	conf, err := json.Marshal(config)
	require.NoError(err)

	mockRPCClient := mocks.NewCelestiaRPCClient(t)
	options := []da.Option{
		celestia.WithRPCClient(mockRPCClient),
	}

	err = dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger(), options...)
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	// only blocks b1 and b2 will be submitted to DA
	block1 := getRandomBlock(1, 10)
	block2 := getRandomBlock(2, 10)
	batch1 := &types.Batch{
		StartHeight: block1.Header.Height,
		EndHeight:   block1.Header.Height,
		Blocks:      []*types.Block{block1},
	}
	batch2 := &types.Batch{
		StartHeight: block2.Header.Height,
		EndHeight:   block2.Header.Height,
		Blocks:      []*types.Block{block2},
	}

	var mockres1, mockres2 da.ResultSubmitBatch
	mockRPCClient.On("SubmitPayForBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&state.TxResponse{}, nil).Once().Run(func(args mock.Arguments) {
		mockres1 = mockdlc.SubmitBatch(batch1)
	})

	mockRPCClient.On("SubmitPayForBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&state.TxResponse{}, nil).Once().Run(func(args mock.Arguments) {
		mockres2 = mockdlc.SubmitBatch(batch2)
	})

	time.Sleep(2 * mockDaBlockTime)

	t.Log("Submitting batch1")
	_ = dalc.SubmitBatch(batch1)
	h1 := mockres1.DAHeight
	assert.Equal(da.StatusSuccess, mockres1.Code)

	time.Sleep(2 * mockDaBlockTime)

	t.Log("Submitting batch1")
	_ = dalc.SubmitBatch(batch2)
	h2 := mockres2.DAHeight
	assert.Equal(da.StatusSuccess, mockres2.Code)

	var retreiveRes da.ResultRetrieveBatch
	mockRPCClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
		height := args.Get(1).(uint64)
		result := mockdlc.RetrieveBatches(height)

		retreiveRes.Code = result.Code
		retreiveRes.Batches = result.Batches
	})
	// wait a bit more than mockDaBlockTime, so dymint blocks can be "included" in mock block
	time.Sleep(mockDaBlockTime + 20*time.Millisecond)

	// call retrieveBlocks
	retriever := dalc.(da.BatchRetriever)

	_ = retriever.RetrieveBatches(h1)
	assert.Equal(da.StatusSuccess, retreiveRes.Code)
	require.True(len(retreiveRes.Batches) == 1)
	compareBatches(t, batch1, retreiveRes.Batches[0])

	_ = retriever.RetrieveBatches(h2)
	assert.Equal(da.StatusSuccess, retreiveRes.Code)
	require.True(len(retreiveRes.Batches) == 1)
	compareBatches(t, batch2, retreiveRes.Batches[0])

	_ = retriever.RetrieveBatches(2)
	assert.Equal(da.StatusSuccess, retreiveRes.Code)
	require.True(len(retreiveRes.Batches) == 0)
}

//TODO: move to testutils
/* ---------------------------------- UTILS --------------------------------- */
func compareBlocks(t *testing.T, b1, b2 *types.Block) {
	t.Helper()
	assert.Equal(t, b1.Header.Height, b2.Header.Height)
	assert.Equal(t, b1.Header.Hash(), b2.Header.Hash())
	assert.Equal(t, b1.Header.AppHash, b2.Header.AppHash)
}

func compareBatches(t *testing.T, b1, b2 *types.Batch) {
	t.Helper()
	assert.Equal(t, b1.StartHeight, b2.StartHeight)
	assert.Equal(t, b1.EndHeight, b2.EndHeight)
	assert.Equal(t, len(b1.Blocks), len(b2.Blocks))
	for i := range b1.Blocks {
		compareBlocks(t, b1.Blocks[i], b2.Blocks[i])
	}
}

// copy-pasted from store/store_test.go
func getRandomBlock(height uint64, nTxs int) *types.Block {
	block := &types.Block{
		Header: types.Header{
			Height: height,
		},
		Data: types.Data{
			Txs: make(types.Txs, nTxs),
			IntermediateStateRoots: types.IntermediateStateRoots{
				RawRootsList: make([][]byte, nTxs),
			},
		},
	}
	copy(block.Header.AppHash[:], getRandomBytes(32))

	for i := 0; i < nTxs; i++ {
		block.Data.Txs[i] = getRandomTx()
		block.Data.IntermediateStateRoots.RawRootsList[i] = getRandomBytes(32)
	}

	if nTxs == 0 {
		block.Data.Txs = nil
		block.Data.IntermediateStateRoots.RawRootsList = nil
	}

	return block
}

func getRandomTx() types.Tx {
	size := rand.Int()%100 + 100
	return types.Tx(getRandomBytes(size))
}

func getRandomBytes(n int) []byte {
	data := make([]byte, n)
	_, _ = rand.Read(data)
	return data
}
