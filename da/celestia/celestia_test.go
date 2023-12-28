package celestia_test

import (
	"encoding/json"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/pubsub"
	"google.golang.org/grpc"

	"github.com/tendermint/tendermint/libs/log"

	mocks "github.com/dymensionxyz/dymint/mocks/da/celestia"
	"github.com/rollkit/celestia-openrpc/types/state"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	cmock "github.com/dymensionxyz/dymint/da/celestia/mock"
	grpcda "github.com/dymensionxyz/dymint/da/grpc"
	"github.com/dymensionxyz/dymint/da/grpc/mockserv"
	damock "github.com/dymensionxyz/dymint/da/mock"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

const mockDaBlockTime = 100 * time.Millisecond

func TestDALC(t *testing.T) {
	httpServer := startMockCelestiaNodeServer(t)
	defer httpServer.Stop()

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

	// // wait a bit more than mockDaBlockTime, so mock DA can "produce" some blocks
	// time.Sleep(mockDaBlockTime + 20*time.Millisecond)

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
		t.Log("IM HERE")
		//check if batch1 exists in mock DA
		if batch1 == nil {
			t.Fatal("batch1 is nil")
		}
		mockres1 = mockdlc.SubmitBatch(batch1)
	})

	mockRPCClient.On("SubmitPayForBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&state.TxResponse{}, nil).Once().Run(func(args mock.Arguments) {
		t.Log("IM HERE2")
		//check if batch1 exists in mock DA
		if batch2 == nil {
			t.Fatal("batch2 is nil")
		}
		mockres2 = mockdlc.SubmitBatch(batch2)
	})

	time.Sleep(mockDaBlockTime + 20*time.Millisecond)

	t.Log("Submitting batch1")
	_ = dalc.SubmitBatch(batch1)
	h1 := mockres1.DAHeight
	assert.Equal(da.StatusSuccess, mockres1.Code)

	t.Log("Submitting batch1")
	_ = dalc.SubmitBatch(batch2)
	h2 := mockres2.DAHeight
	assert.Equal(da.StatusSuccess, mockres2.Code)

	var retreiveRes da.ResultRetrieveBatch
	mockRPCClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
		height := args.Get(1).(uint64)
		t.Log("GetAll called with height", height)
		result := mockdlc.RetrieveBatches(height)

		retreiveRes.Code = result.Code
		retreiveRes.Batches = result.Batches
	})
	// wait a bit more than mockDaBlockTime, so dymint blocks can be "included" in mock block
	time.Sleep(mockDaBlockTime + 20*time.Millisecond)

	// call retrieveBlocks
	retriever := dalc.(da.BatchRetriever)

	_ = retriever.RetrieveBatches(h1)
	// print the check result
	// t.Logf("CheckBatchAvailability result: %+v", res)
	assert.Equal(da.StatusSuccess, retreiveRes.Code)
	//compare the retrieved batch with the original batch
	require.True(len(retreiveRes.Batches) == 1)
	// assert.True(reflect.DeepEqual(batch1, retreiveRes.Batches[0]))
	compareBatches(t, batch1, retreiveRes.Batches[0])

	_ = retriever.RetrieveBatches(h2)
	// print the check result
	assert.Equal(da.StatusSuccess, retreiveRes.Code)
	//compare the retrieved batch with the original batch
	require.True(len(retreiveRes.Batches) == 1)
	// assert.True(reflect.DeepEqual(batch1, retreiveRes.Batches[0]))
	compareBatches(t, batch2, retreiveRes.Batches[0])

	_ = retriever.RetrieveBatches(2)
	// print the check result
	assert.Equal(da.StatusSuccess, retreiveRes.Code)
	//compare the retrieved batch with the original batch
	require.True(len(retreiveRes.Batches) == 0)
}

/*
func TestRetrieve(t *testing.T) {
	// grpcServer := startMockGRPCServ(t)
	// defer grpcServer.GracefulStop()

	httpServer := startMockCelestiaNodeServer(t)
	defer httpServer.Stop()

	dalc := registry.GetClient("celestia")
	require := require.New(t)
	assert := assert.New(t)

	// mock DALC will advance block height every 100ms
	conf := []byte{}
	if _, ok := dalc.(*damock.DataAvailabilityLayerClient); ok {
		conf = []byte(mockDaBlockTime.String())
	}
	if _, ok := dalc.(*celestia.DataAvailabilityLayerClient); ok {
		config := celestia.Config{
			BaseURL:  "http://localhost:26658",
			Timeout:  30 * time.Second,
			GasLimit: 3000000,
			Fee:      2000000,
		}
		err := config.InitNamespaceID()
		require.NoError(err)
		conf, _ = json.Marshal(config)
	}

	//TODO: ADD OPTIONS WITH MOCK RPC CLIENT
	// options := []da.Option{
	// 	celestia.WithRPCClient(mockRPCClient),
	// }

	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	err := dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger())
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	// wait a bit more than mockDaBlockTime, so mock can "produce" some blocks
	time.Sleep(mockDaBlockTime + 20*time.Millisecond)

	retriever := dalc.(da.BatchRetriever)
	countAtHeight := make(map[uint64]int)
	batches := make(map[*types.Batch]uint64)

	for i := uint64(0); i < 100; i++ {
		b := getRandomBlock(i, rand.Int()%20)
		batch := &types.Batch{
			StartHeight: i,
			EndHeight:   i,
			Blocks:      []*types.Block{b},
			Commits: []*types.Commit{{
				Height:     b.Header.Height,
				HeaderHash: b.Header.Hash(),
			},
			},
		}
		resp := dalc.SubmitBatch(batch)
		assert.Equal(da.StatusSuccess, resp.Code, resp.Message)
		time.Sleep(time.Duration(rand.Int63() % mockDaBlockTime.Milliseconds()))

		countAtHeight[resp.DAHeight]++
		batches[batch] = resp.DAHeight
	}

	// wait a bit more than mockDaBlockTime, so mock can "produce" last blocks
	time.Sleep(mockDaBlockTime + 20*time.Millisecond)

	for h, cnt := range countAtHeight {
		t.Log("Retrieving block, DA Height", h)
		ret := retriever.RetrieveBatches(h)
		assert.Equal(da.StatusSuccess, ret.Code, ret.Message)
		require.NotEmpty(ret.Batches, h)
		assert.Len(ret.Batches, cnt, h)
	}

	for b, h := range batches {
		ret := retriever.RetrieveBatches(h)
		assert.Equal(da.StatusSuccess, ret.Code, h)
		require.NotEmpty(ret.Batches, h)
		assert.Contains(ret.Batches, b, h)
	}

}
*/

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

func startMockGRPCServ(t *testing.T) *grpc.Server {
	t.Helper()
	conf := grpcda.DefaultConfig
	srv := mockserv.GetServer(store.NewDefaultInMemoryKVStore(), conf, []byte(mockDaBlockTime.String()))
	lis, err := net.Listen("tcp", conf.Host+":"+strconv.Itoa(conf.Port))
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_ = srv.Serve(lis)
	}()
	return srv
}

func startMockCelestiaNodeServer(t *testing.T) *cmock.Server {
	t.Helper()
	httpSrv := cmock.NewServer(mockDaBlockTime, log.TestingLogger())
	l, err := net.Listen("tcp4", ":26658")
	if err != nil {
		t.Fatal("failed to create listener for mock celestia-node RPC server", "error", err)
	}
	err = httpSrv.Start(l)
	if err != nil {
		t.Fatal("can't start mock celestia-node RPC server")
	}
	return httpSrv
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

	// TODO(tzdybal): see https://github.com/dymensionxyz/dymint/issues/143
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
