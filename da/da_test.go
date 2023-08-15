package da_test

import (
	"encoding/json"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/celestiaorg/go-cnc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"
	"google.golang.org/grpc"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	cmock "github.com/dymensionxyz/dymint/da/celestia/mock"
	grpcda "github.com/dymensionxyz/dymint/da/grpc"
	"github.com/dymensionxyz/dymint/da/grpc/mockserv"
	"github.com/dymensionxyz/dymint/da/mock"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

const mockDaBlockTime = 100 * time.Millisecond

func TestLifecycle(t *testing.T) {
	srv := startMockGRPCServ(t)
	defer srv.GracefulStop()

	for _, dalc := range registry.RegisteredClients() {
		//TODO(omritoptix): Possibly add support for avail here.
		if dalc == "avail" {
			continue
		}
		t.Run(dalc, func(t *testing.T) {
			doTestLifecycle(t, dalc)
		})
	}
}

func doTestLifecycle(t *testing.T, daType string) {
	var err error
	require := require.New(t)
	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()

	dacfg := []byte{}
	dalc := registry.GetClient(daType)

	if daType == "celestia" {
		dacfg, err = json.Marshal(celestia.CelestiaDefaultConfig)
		require.NoError(err)
	}

	err = dalc.Init(dacfg, pubsubServer, nil, log.TestingLogger())
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	err = dalc.Stop()
	require.NoError(err)
}

func TestDALC(t *testing.T) {
	grpcServer := startMockGRPCServ(t)
	defer grpcServer.GracefulStop()

	httpServer := startMockCelestiaNodeServer(t)
	defer httpServer.Stop()

	for _, dalc := range registry.RegisteredClients() {
		//TODO(omritoptix): Possibly add support for avail here.
		if dalc == "avail" {
			continue
		}
		t.Run(dalc, func(t *testing.T) {
			doTestDALC(t, registry.GetClient(dalc))
		})
	}
}

func doTestDALC(t *testing.T, dalc da.DataAvailabilityLayerClient) {
	require := require.New(t)
	assert := assert.New(t)

	// mock DALC will advance block height every 100ms
	conf := []byte{}
	if _, ok := dalc.(*mock.DataAvailabilityLayerClient); ok {
		conf = []byte(mockDaBlockTime.String())
	}
	if _, ok := dalc.(*celestia.DataAvailabilityLayerClient); ok {
		config := celestia.Config{
			BaseURL:     "http://localhost:26658",
			Timeout:     30 * time.Second,
			GasLimit:    3000000,
			Fee:         200000000,
			NamespaceID: cnc.Namespace{Version: 0, ID: []byte{0, 0, 0, 0, 0, 0, 255, 255}},
		}
		conf, _ = json.Marshal(config)
	}
	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	err := dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger())
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	// wait a bit more than mockDaBlockTime, so mock can "produce" some blocks
	time.Sleep(mockDaBlockTime + 20*time.Millisecond)

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

	resp := dalc.SubmitBatch(batch1)
	h1 := resp.DAHeight
	assert.Equal(da.StatusSuccess, resp.Code)

	resp = dalc.SubmitBatch(batch2)
	h2 := resp.DAHeight
	assert.Equal(da.StatusSuccess, resp.Code)

	// wait a bit more than mockDaBlockTime, so dymint blocks can be "included" in mock block
	time.Sleep(mockDaBlockTime + 20*time.Millisecond)

	check := dalc.CheckBatchAvailability(h1)
	// print the check result
	t.Logf("CheckBatchAvailability result: %+v", check)
	assert.Equal(da.StatusSuccess, check.Code)
	assert.True(check.DataAvailable)

	check = dalc.CheckBatchAvailability(h2)
	assert.Equal(da.StatusSuccess, check.Code)
	assert.True(check.DataAvailable)

	// this height should not be used by DALC
	check = dalc.CheckBatchAvailability(h1 - 1)
	assert.Equal(da.StatusSuccess, check.Code)
	assert.False(check.DataAvailable)
}

func TestRetrieve(t *testing.T) {
	grpcServer := startMockGRPCServ(t)
	defer grpcServer.GracefulStop()

	httpServer := startMockCelestiaNodeServer(t)
	defer httpServer.Stop()

	for _, client := range registry.RegisteredClients() {
		//TODO(omritoptix): Possibly add support for avail here.
		if client == "avail" {
			continue
		}
		t.Run(client, func(t *testing.T) {
			dalc := registry.GetClient(client)
			_, ok := dalc.(da.BatchRetriever)
			if ok {
				doTestRetrieve(t, dalc)
			}
		})
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

func doTestRetrieve(t *testing.T, dalc da.DataAvailabilityLayerClient) {
	require := require.New(t)
	assert := assert.New(t)

	// mock DALC will advance block height every 100ms
	conf := []byte{}
	if _, ok := dalc.(*mock.DataAvailabilityLayerClient); ok {
		conf = []byte(mockDaBlockTime.String())
	}
	if _, ok := dalc.(*celestia.DataAvailabilityLayerClient); ok {
		config := celestia.Config{
			BaseURL:     "http://localhost:26658",
			Timeout:     30 * time.Second,
			GasLimit:    3000000,
			Fee:         2000000,
			NamespaceID: cnc.Namespace{Version: 0, ID: []byte{0, 0, 0, 0, 0, 0, 255, 255}},
		}
		conf, _ = json.Marshal(config)
	}
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
