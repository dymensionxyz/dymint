package da_test

import (
	"bytes"
	cryptoRand "crypto/rand"
	"encoding/json"
	"math/rand" //#gosec
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	"github.com/dymensionxyz/dymint/da/local"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

const mockDaBlockTime = 100 * time.Millisecond

// TODO: move to mock DA test
func TestLifecycle(t *testing.T) {
	doTestLifecycle(t, "mock")
}

func doTestLifecycle(t *testing.T, daType string) {
	var err error
	require := require.New(t)
	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(err)

	dacfg := []byte{}
	dalc := registry.GetClient(daType)

	err = dalc.Init(dacfg, pubsubServer, nil, log.TestingLogger())
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	err = dalc.Stop()
	require.NoError(err)
}

func TestDALC(t *testing.T) {
	doTestDALC(t, registry.GetClient("mock"))
}

func doTestDALC(t *testing.T, mockDalc da.DataAvailabilityLayerClient) {
	require := require.New(t)
	assert := assert.New(t)
	var err error

	// mock DALC will advance block height every 100ms
	if _, ok := mockDalc.(*local.DataAvailabilityLayerClient); !ok {
		t.Fatal("mock DALC is not of type *local.DataAvailabilityLayerClient")
	}
	conf := []byte(mockDaBlockTime.String())
	dalc := mockDalc.(*local.DataAvailabilityLayerClient)

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(err)
	err = dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger())
	require.NoError(err)

	err = dalc.Start()
	require.NoError(err)

	// wait a bit more than mockDaBlockTime, so mock can "produce" some blocks
	time.Sleep(mockDaBlockTime + 20*time.Millisecond)

	// only blocks b1 and b2 will be submitted to DA
	block1 := getRandomBlock(1, 10)
	block2 := getRandomBlock(2, 10)
	batch1 := &types.Batch{
		Blocks: []*types.Block{block1},
	}
	batch2 := &types.Batch{
		Blocks: []*types.Block{block2},
	}

	resp := dalc.SubmitBatch(batch1)
	h1 := resp.SubmitMetaData
	assert.Equal(da.StatusSuccess, resp.Code)

	resp = dalc.SubmitBatch(batch2)
	h2 := resp.SubmitMetaData
	assert.Equal(da.StatusSuccess, resp.Code)

	// wait a bit more than mockDaBlockTime, so dymint blocks can be "included" in mock block
	time.Sleep(mockDaBlockTime + 20*time.Millisecond)

	check := dalc.CheckBatchAvailability(h1.DAPath)
	// print the check result
	t.Logf("CheckBatchAvailability result: %+v", check)
	assert.Equal(da.StatusSuccess, check.Code)

	check = dalc.CheckBatchAvailability(h2.DAPath)
	assert.Equal(da.StatusSuccess, check.Code)

	path := &local.SubmitMetaData{}
	daMetaData, err := path.FromPath(h1.DAPath)
	require.NoError(err)
	daMetaData.Height = daMetaData.Height - 1
	// this height should not be used by DALC
	check = dalc.CheckBatchAvailability(h1.DAPath)
	assert.Equal(da.StatusSuccess, check.Code)
}

func TestRetrieve(t *testing.T) {
	dalc := registry.GetClient("mock")
	_, ok := dalc.(da.BatchRetriever)
	if !ok {
		t.Fatal("mock DALC is not of type da.BatchRetriever")
	}
	doTestRetrieve(t, dalc)
}

func doTestRetrieve(t *testing.T, dalc da.DataAvailabilityLayerClient) {
	require := require.New(t)
	assert := assert.New(t)
	var err error

	// mock DALC will advance block height every 100ms
	conf := []byte{}
	if _, ok := dalc.(*local.DataAvailabilityLayerClient); ok {
		conf = []byte(mockDaBlockTime.String())
	}
	if _, ok := dalc.(*celestia.DataAvailabilityLayerClient); ok {
		config := celestia.Config{
			BaseConfig: da.BaseConfig{
				Timeout: 30 * time.Second,
			},
			BaseURL:   "http://localhost:26658",
			GasPrices: celestia.DefaultGasPrices,
		}
		err := config.InitNamespaceID()
		require.NoError(err)
		conf, _ = json.Marshal(config)
	}

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(err)
	err = dalc.Init(conf, pubsubServer, store.NewDefaultInMemoryKVStore(), log.TestingLogger())
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
			Blocks: []*types.Block{b},
			Commits: []*types.Commit{
				{
					Height:     b.Header.Height,
					HeaderHash: b.Header.Hash(),
				},
			},
		}
		resp := dalc.SubmitBatch(batch)
		assert.Equal(da.StatusSuccess, resp.Code, resp.Message)
		time.Sleep(time.Duration(rand.Int63() % mockDaBlockTime.Milliseconds()))

		path := &local.SubmitMetaData{}
		daMetaData, err := path.FromPath(resp.SubmitMetaData.DAPath)
		require.NoError(err)

		countAtHeight[daMetaData.Height]++
		batches[batch] = daMetaData.Height
	}

	// wait a bit more than mockDaBlockTime, so mock can "produce" last blocks
	time.Sleep(mockDaBlockTime + 20*time.Millisecond)

	for h, cnt := range countAtHeight {
		daMetaData := &local.SubmitMetaData{
			Height: h,
		}
		t.Log("Retrieving block, DA Height", h)
		ret := retriever.RetrieveBatches(daMetaData.ToPath())
		assert.Equal(da.StatusSuccess, ret.Code, ret.Message)
		require.NotEmpty(ret.Batches, h)
		assert.Len(ret.Batches, cnt, h)
	}

	for b, h := range batches {
		daMetaData := &local.SubmitMetaData{
			Height: h,
		}
		ret := retriever.RetrieveBatches(daMetaData.ToPath())
		assert.Equal(da.StatusSuccess, ret.Code, h)
		require.NotEmpty(ret.Batches, h)
		assert.Contains(ret.Batches, b, h)
	}
}

// TODO: move to testutils

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

	// TODO(tzdybal): see https://github.com/dymensionxyz/dymint/issues/143
	if nTxs == 0 {
		block.Data.Txs = nil
	}

	return block
}

func getRandomTx() types.Tx {
	NuM := rand.Int()
	size := NuM%100 + 100
	return types.Tx(getRandomBytes(size))
}

func getRandomBytes(n int) []byte {
	data := make([]byte, n)
	_, _ = cryptoRand.Read(data)
	return data
}

func FuzzDASubmitMetaData(f *testing.F) {
	f.Fuzz(func(t *testing.T, client string, height uint64, index, length int, commitment, namespace, root []byte) {
		if client == "" || strings.Contains(client, da.PathSeparator) || len(commitment) == 0 || len(namespace) == 0 || len(root) == 0 {
			t.Skip()
		}
		submitMetadata := celestia.SubmitMetaData{
			Height:     height,
			Index:      index,
			Length:     length,
			Commitment: commitment,
			Namespace:  namespace,
			Root:       root,
		}
		daMetaData := da.DASubmitMetaData{
			Client: da.Client(client),
			DAPath: submitMetadata.ToPath(),
		}

		path := daMetaData.ToPath()

		got, err := daMetaData.FromPath(path)
		require.NoError(t, err)

		require.Equal(t, daMetaData.Client, got.Client)

		gotPath, err := submitMetadata.FromPath(daMetaData.DAPath)
		require.NoError(t, err)

		require.Equal(t, submitMetadata.Height, gotPath.Height)
		require.Equal(t, submitMetadata.Index, gotPath.Index)
		require.Equal(t, submitMetadata.Length, gotPath.Length)
		require.True(t, bytes.Equal(submitMetadata.Commitment, gotPath.Commitment))
		require.True(t, bytes.Equal(submitMetadata.Namespace, gotPath.Namespace))
		require.True(t, bytes.Equal(submitMetadata.Root, gotPath.Root))
	})
}
