package celestia_test

import (
	"bytes"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/celestia-openrpc/types/header"
	"github.com/celestiaorg/nmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/tendermint/tendermint/libs/log"

	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/celestia/client"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

const mockDaBlockTime = 100 * time.Millisecond

func TestDALC(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	mockRPCClient, dalc, nID, header := setDAandMock(t)
	// only blocks b1 and b2 will be submitted to DA
	block1 := getRandomBlock(1, 10)
	block2 := getRandomBlock(2, 10)
	batch1 := &types.Batch{
		Blocks: []*types.Block{block1},
	}
	batch2 := &types.Batch{
		Blocks: []*types.Block{block2},
	}

	nIDSize := 1
	tree := exampleNMT(nIDSize, true, 1, 2, 3, 4)
	// build a proof for an NID that is within the namespace range of the tree
	proof, _ := tree.ProveNamespace(nID)
	blobProof := blob.Proof([]*nmt.Proof{&proof})

	mockRPCClient.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint64(1234), nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&blobProof, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("Included", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetByHeight", mock.Anything, mock.Anything).Return(header, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })

	mockRPCClient.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint64(1234), nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&blobProof, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("Included", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetByHeight", mock.Anything, mock.Anything).Return(header, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })

	time.Sleep(2 * mockDaBlockTime)

	t.Log("Submitting batch1")
	res1 := dalc.SubmitBatch(batch1)
	h1 := res1.SubmitMetaData
	assert.Equal(da.StatusSuccess, res1.Code)

	time.Sleep(2 * mockDaBlockTime)

	t.Log("Submitting batch2")
	res2 := dalc.SubmitBatch(batch2)
	assert.Equal(da.StatusSuccess, res2.Code)

	data1, _ := batch1.MarshalBinary()
	blob1, _ := blob.NewBlobV0(nID, data1)

	mockRPCClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(blob1, nil).Run(func(args mock.Arguments) {
	})

	// call retrieveBlocks
	retriever := dalc.(da.BatchRetriever)

	retreiveRes := retriever.RetrieveBatches(h1)
	assert.Equal(da.StatusSuccess, retreiveRes.Code)
	require.True(len(retreiveRes.Batches) == 1)
	compareBatches(t, batch1, retreiveRes.Batches[0])
}

func TestRetrievalNotFound(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	mockRPCClient, dalc, nID, headers := setDAandMock(t)
	// only blocks b1 and b2 will be submitted to DA
	block1 := getRandomBlock(1, 10)
	batch1 := &types.Batch{
		Blocks: []*types.Block{block1},
	}

	nIDSize := 1
	tree := exampleNMT(nIDSize, true, 1, 2, 3, 4)
	// build a proof for an NID that is within the namespace range of the tree
	proof, _ := tree.ProveNamespace(nID)
	blobProof := blob.Proof([]*nmt.Proof{&proof})

	mockRPCClient.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint64(1234), nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&blobProof, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("Included", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetByHeight", mock.Anything, mock.Anything).Return(headers, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })

	time.Sleep(2 * mockDaBlockTime)

	// data1, _ := batch1.MarshalBinary()
	// blob1, _ := blob.NewBlobV0(nID, data1)

	t.Log("Submitting batch1")
	res1 := dalc.SubmitBatch(batch1)
	h1 := res1.SubmitMetaData
	assert.Equal(da.StatusSuccess, res1.Code)

	mockRPCClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
	})

	retriever := dalc.(da.BatchRetriever)

	retreiveRes := retriever.RetrieveBatches(h1)

	assert.ErrorIs(retreiveRes.Error, da.ErrBlobNotFound)
	require.True(len(retreiveRes.Batches) == 0)
}

func TestAvalabilityOK(t *testing.T) {
	assert := assert.New(t)
	// require := require.New(t)

	mockRPCClient, dalc, nID, headers := setDAandMock(t)
	// only blocks b1 and b2 will be submitted to DA
	block1 := getRandomBlock(1, 10)
	batch1 := &types.Batch{
		Blocks: []*types.Block{block1},
	}

	nIDSize := 1
	tree := exampleNMT(nIDSize, true, 1, 2, 3, 4)
	// build a proof for an NID that is within the namespace range of the tree
	proof, _ := tree.ProveNamespace(nID)
	blobProof := blob.Proof([]*nmt.Proof{&proof})

	mockRPCClient.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint64(1234), nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&blobProof, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("Included", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetByHeight", mock.Anything, mock.Anything).Return(headers, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })

	time.Sleep(2 * mockDaBlockTime)

	// data1, _ := batch1.MarshalBinary()
	// blob1, _ := blob.NewBlobV0(nID, data1)

	t.Log("Submitting batch1")
	res1 := dalc.SubmitBatch(batch1)
	h1 := res1.SubmitMetaData
	assert.Equal(da.StatusSuccess, res1.Code)

	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&blobProof, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("Included", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetByHeight", mock.Anything, mock.Anything).Return(headers, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })

	retriever := dalc.(da.BatchRetriever)

	availRes := retriever.CheckBatchAvailability(h1)
	assert.Equal(da.StatusSuccess, availRes.Code)
}

func TestAvalabilityWrongProof(t *testing.T) {
	assert := assert.New(t)
	// require := require.New(t)

	mockRPCClient, dalc, nID, headers := setDAandMock(t)
	// only blocks b1 and b2 will be submitted to DA
	block1 := getRandomBlock(1, 10)
	batch1 := &types.Batch{
		Blocks: []*types.Block{block1},
	}

	nIDSize := 1
	tree := exampleNMT(nIDSize, true, 1, 2, 3, 4)
	// build a proof for an NID that is within the namespace range of the tree
	proof, _ := tree.ProveNamespace(nID)
	blobProof := blob.Proof([]*nmt.Proof{&proof})

	mockRPCClient.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint64(1234), nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&blobProof, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("Included", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetByHeight", mock.Anything, mock.Anything).Return(headers, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })

	time.Sleep(2 * mockDaBlockTime)

	// data1, _ := batch1.MarshalBinary()
	// blob1, _ := blob.NewBlobV0(nID, data1)

	t.Log("Submitting batch1")
	res1 := dalc.SubmitBatch(batch1)
	h1 := res1.SubmitMetaData
	assert.Equal(da.StatusSuccess, res1.Code)

	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetByHeight", mock.Anything, mock.Anything).Return(headers, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })

	retriever := dalc.(da.BatchRetriever)

	availRes := retriever.CheckBatchAvailability(h1)
	assert.ErrorIs(availRes.Error, da.ErrUnableToGetProof)
}

func TestRetrievalWrongCommitment(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	commitmentString := "3f568f651fe72fa2131bd86c09bb23763e0a3cb45211b035bfa688711c76ce78"
	commitment, _ := hex.DecodeString(commitmentString)

	mockRPCClient, dalc, namespace, headers := setDAandMock(t)

	mockRPCClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
	})

	retriever := dalc.(da.BatchRetriever)

	h1 := &da.DASubmitMetaData{
		Height:     1,
		Commitment: commitment,
		Namespace:  namespace,
	}
	retrieveRes := retriever.RetrieveBatches(h1)
	assert.ErrorIs(retrieveRes.Error, da.ErrBlobNotFound)
	require.True(len(retrieveRes.Batches) == 0)

	mockRPCClient.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })
	mockRPCClient.On("GetByHeight", mock.Anything, mock.Anything).Return(headers, nil).Once().Run(func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) })

	availRes := retriever.CheckBatchAvailability(h1)
	assert.ErrorIs(availRes.Error, da.ErrUnableToGetProof)
}

func setDAandMock(t *testing.T) (*mocks.MockDAClient, da.DataAvailabilityLayerClient, []byte, *header.ExtendedHeader) {
	var err error
	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(t, err)
	defer func() {
		err = pubsubServer.Stop()
		require.NoError(t, err)
	}()

	require := require.New(t)

	// init celestia DA with mock RPC client
	dalc := registry.GetClient("celestia")

	config := celestia.Config{
		BaseURL:        "http://localhost:26658",
		Timeout:        30 * time.Second,
		GasPrices:      celestia.DefaultGasPrices,
		NamespaceIDStr: "0000000000000000ffff",
	}
	err = config.InitNamespaceID()
	require.NoError(err)
	conf, err := json.Marshal(config)
	require.NoError(err)

	mockRPCClient := mocks.NewMockDAClient(t)
	options := []da.Option{
		celestia.WithRPCClient(mockRPCClient),
		celestia.WithRPCAttempts(1),
		celestia.WithRPCRetryDelay(time.Second * 2),
	}

	err = dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger(), options...)
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)
	roots := [][]byte{[]byte("apple"), []byte("watermelon"), []byte("kiwi")}
	dah := &header.DataAvailabilityHeader{
		RowRoots:    roots,
		ColumnRoots: roots,
	}
	header := &header.ExtendedHeader{
		DAH: dah,
	}

	return mockRPCClient, dalc, config.NamespaceID.Bytes(), header
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
	assert.Equal(t, b1.StartHeight(), b2.StartHeight())
	assert.Equal(t, b1.EndHeight(), b2.EndHeight())
	assert.Equal(t, len(b1.Blocks), len(b2.Blocks))
	for i := range b1.Blocks {
		compareBlocks(t, b1.Blocks[i], b2.Blocks[i])
	}
}

// copy-pasted from store/store_test.go
func getRandomBlock(height uint64, nTxs int) *types.Block {
	block := &types.Block{
		Header: types.Header{
			Height:                height,
			ConsensusMessagesHash: types.ConsMessagesHash(nil),
		},
		Data: types.Data{
			Txs: make(types.Txs, nTxs),
		},
	}
	copy(block.Header.AppHash[:], getRandomBytes(32))

	for i := 0; i < nTxs; i++ {
		block.Data.Txs[i] = getRandomTx()
	}

	if nTxs == 0 {
		block.Data.Txs = nil
	}

	return block
}

func getRandomTx() types.Tx {
	size := rand.Int()%100 + 100
	return types.Tx(getRandomBytes(size))
}

func getRandomBytes(n int) []byte {
	data := make([]byte, n)
	_, _ = cryptoRand.Read(data)
	return data
}

// exampleNMT creates a new NamespacedMerkleTree with the given namespace ID size and leaf namespace IDs. Each byte in the leavesNIDs parameter corresponds to one leaf's namespace ID. If nidSize is greater than 1, the function repeats each NID in leavesNIDs nidSize times before prepending it to the leaf data.
func exampleNMT(nidSize int, ignoreMaxNamespace bool, leavesNIDs ...byte) *nmt.NamespacedMerkleTree {
	tree := nmt.New(sha256.New(), nmt.NamespaceIDSize(nidSize), nmt.IgnoreMaxNamespace(ignoreMaxNamespace))
	for i, nid := range leavesNIDs {
		namespace := bytes.Repeat([]byte{nid}, nidSize)
		d := append(namespace, []byte(fmt.Sprintf("leaf_%d", i))...)
		if err := tree.Push(d); err != nil {
			panic(fmt.Sprintf("unexpected error: %v", err))
		}
	}
	return tree
}
