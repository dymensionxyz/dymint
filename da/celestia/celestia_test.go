package celestia_test

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/celestiaorg/nmt"
	"github.com/rollkit/celestia-openrpc/types/blob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/tendermint/tendermint/libs/log"

	mocks "github.com/dymensionxyz/dymint/mocks/da/celestia"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	damock "github.com/dymensionxyz/dymint/da/mock"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

const mockDaBlockTime = 100 * time.Millisecond

func TestRetrievalRealNode(t *testing.T) {
	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	defer pubsubServer.Stop()
	dalc := registry.GetClient("celestia")
	config := celestia.Config{
		BaseURL:        "http://192.168.1.5:26658",
		Timeout:        30 * time.Second,
		GasPrices:      1.0,
		GasAdjustment:  1.3,
		NamespaceIDStr: "e06c57a64b049d6463ef",
		AuthToken:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJwdWJsaWMiLCJyZWFkIiwid3JpdGUiLCJhZG1pbiJdfQ.Lo9NywGbESS4dSSIbLsZuuq5bTYL0LnLjlrNgct6aXg",
	}
	require := require.New(t)

	conf, err := json.Marshal(config)
	require.NoError(err)
	err = dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger())
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	// only blocks b1 and b2 will be submitted to DA
	block1 := getRandomBlock(1, 10)
	block2 := getRandomBlock(2, 20)
	block3 := getRandomBlock(3, 20)
	block4 := getRandomBlock(3, 10)

	batch1 := &types.Batch{
		StartHeight: block1.Header.Height,
		EndHeight:   block3.Header.Height,
		Blocks:      []*types.Block{block1, block2, block3, block4},
	}

	t.Log("Submitting batch")
	resp := dalc.SubmitBatch(batch1)
	t.Log("Height:", resp.DAHeight)
	for _, commitment := range resp.Commitments {
		t.Log("Commitment:", hex.EncodeToString(commitment))

	}
	for _, index := range resp.Indexes {
		t.Log("Index:", index)

	}
	for _, length := range resp.Length {
		t.Log("Shares:", length)

	}

	//commitmentString := "3f568f651fe72fa2131bd86c09bb23763e0a3cb45211b035bfa688711c76ce78"
	//commitment, _ := hex.DecodeString(commitmentString)
	//resultRetrieveBatch := dalc.(da.BatchRetrieverByCommitment).RetrieveBatchesByCommitment(resp.DAHeight, [][]byte{commitment})
	resultRetrieveBatch := dalc.(da.BatchRetrieverByCommitment).RetrieveBatchesByCommitment(resp.DAHeight, resp.Commitments)
	if resultRetrieveBatch.Code == da.StatusError {
		t.Error("Failed to retrieve batch")
	} else {
		t.Log("Result:", resultRetrieveBatch.BaseResult.Message)
	}

}

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

	nIDSize := 1
	tree := exampleNMT(nIDSize, true, 1, 2, 3, 4)
	// build a proof for an NID that is within the namespace range of the tree
	nID := []byte{1}
	proof, err := tree.ProveNamespace(nID)
	blobProof := blob.Proof([]*nmt.Proof{&proof})

	var mockres1, mockres2 da.ResultSubmitBatch
	mockRPCClient.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint64(1234), nil).Once().Run(func(args mock.Arguments) {
		mockres1 = mockdlc.SubmitBatch(batch1)
	})
	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&blobProof, nil).Once().Run(func(args mock.Arguments) {
		mockres1 = mockdlc.SubmitBatch(batch1)
	})
	mockRPCClient.On("Included", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Once().Run(func(args mock.Arguments) {
		mockres1 = mockdlc.SubmitBatch(batch1)
	})

	mockRPCClient.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint64(1234), nil).Once().Run(func(args mock.Arguments) {
		mockres2 = mockdlc.SubmitBatch(batch2)
	})
	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&blobProof, nil).Once().Run(func(args mock.Arguments) {
		mockres2 = mockdlc.SubmitBatch(batch2)
	})
	mockRPCClient.On("Included", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Once().Run(func(args mock.Arguments) {
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

func TestLightNode(t *testing.T) {
	t.Skip()
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
