package client_test

import (
	stdbytes "bytes"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	tmcfg "github.com/tendermint/tendermint/config"
	tmjson "github.com/tendermint/tendermint/libs/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/encoding"
	"github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/mempool"
	tmmocks "github.com/dymensionxyz/dymint/mocks/github.com/tendermint/tendermint/abci/types"
	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/rpc/client"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	"github.com/dymensionxyz/dymint/version"
)

var expectedInfo = abci.ResponseInfo{
	Version:         "v0.0.1",
	AppVersion:      1,
	LastBlockHeight: 0,
}

var mockTxProcessingTime = 10 * time.Millisecond

func TestConnectionGetters(t *testing.T) {
	assert := assert.New(t)

	_, rpc := getRPC(t)
	assert.NotNil(rpc.Consensus())
	assert.NotNil(rpc.Mempool())
	assert.NotNil(rpc.Snapshot())
	assert.NotNil(rpc.Query())
}

func TestInfo(t *testing.T) {
	assert := assert.New(t)

	mockApp, rpc := getRPC(t)
	mockApp.On("Info", mock.Anything).Return(expectedInfo)

	info, err := rpc.ABCIInfo(context.Background())
	assert.NoError(err)
	assert.Equal(expectedInfo, info.Response)
}

func TestCheckTx(t *testing.T) {
	assert := assert.New(t)

	expectedTx := []byte("tx data")

	mockApp, rpc := getRPC(t)
	mockApp.On("CheckTx", abci.RequestCheckTx{Tx: expectedTx}).Once().Return(abci.ResponseCheckTx{})

	res, err := rpc.CheckTx(context.Background(), expectedTx)
	assert.NoError(err)
	assert.NotNil(res)
	mockApp.AssertExpectations(t)
}

func TestGenesisChunked(t *testing.T) {
	assert := assert.New(t)

	genDoc := testutil.GenerateGenesis(1)

	mockApp := &tmmocks.MockApplication{}
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)
	mockApp.On("Info", mock.Anything).Return(expectedInfo)
	privKey, _, _ := crypto.GenerateEd25519Key(crand.Reader)
	signingKey, _, _ := crypto.GenerateEd25519Key(crand.Reader)

	config := config.NodeConfig{
		RootDir: "",
		DBPath:  "",
		P2PConfig: config.P2PConfig{
			ListenAddress:                config.DefaultListenAddress,
			BootstrapNodes:               "",
			GossipSubCacheSize:           50,
			BootstrapRetryTime:           30 * time.Second,
			DiscoveryEnabled:             true,
			BlockSyncRequestIntervalTime: 30 * time.Second,
		},
		RPC: config.RPCConfig{},
		BlockManagerConfig: config.BlockManagerConfig{
			BlockTime:                  100 * time.Millisecond,
			BatchSubmitTime:            60 * time.Second,
			BatchSubmitBytes:           1000,
			MaxSkewTime:                24 * time.Hour,
			SequencerSetUpdateInterval: config.DefaultSequencerSetUpdateInterval,
		},
		DAConfig:         "",
		SettlementLayer:  "mock",
		SettlementConfig: settlement.Config{},
	}
	n, err := node.NewNode(
		context.Background(),
		config,
		privKey,
		signingKey,
		proxy.NewLocalClientCreator(mockApp),
		genDoc,
		"",
		log.TestingLogger(),
		mempool.NopMetrics(),
	)
	require.NoError(t, err)

	rpc := client.NewClient(n)

	var expectedID uint = 2
	gc, err := rpc.GenesisChunked(context.Background(), expectedID)
	assert.Error(err)
	assert.Nil(gc)

	err = n.Start()
	require.NoError(t, err)

	expectedID = 0
	gc2, err := rpc.GenesisChunked(context.Background(), expectedID)
	gotID := gc2.ChunkNumber
	assert.NoError(err)
	assert.NotNil(gc2)
	assert.Equal(int(expectedID), gotID)

	gc3, err := rpc.GenesisChunked(context.Background(), 5)
	assert.Error(err)
	assert.Nil(gc3)
}

func TestBroadcastTxAsync(t *testing.T) {
	assert := assert.New(t)

	expectedTx := []byte("tx data")

	mockApp, rpc, node := getRPCAndNode(t)
	mockApp.On("CheckTx", abci.RequestCheckTx{Tx: expectedTx}).Return(abci.ResponseCheckTx{})
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)

	err := node.Start()
	require.NoError(t, err)

	res, err := rpc.BroadcastTxAsync(context.Background(), expectedTx)
	assert.NoError(err)
	assert.NotNil(res)
	assert.Empty(res.Code)
	assert.Empty(res.Data)
	assert.Empty(res.Log)
	assert.Empty(res.Codespace)
	assert.NotEmpty(res.Hash)
	mockApp.AssertExpectations(t)

	err = node.Stop()
	require.NoError(t, err)
}

func TestBroadcastTxSync(t *testing.T) {
	assert := assert.New(t)

	expectedTx := []byte("tx data")
	expectedResponse := abci.ResponseCheckTx{
		Code:      1,
		Data:      []byte("data"),
		Log:       "log",
		Info:      "info",
		GasWanted: 0,
		GasUsed:   0,
		Events:    nil,
		Codespace: "space",
	}

	mockApp, rpc, node := getRPCAndNode(t)
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)

	err := node.Start()
	require.NoError(t, err)

	mockApp.On("CheckTx", abci.RequestCheckTx{Tx: expectedTx}).Return(expectedResponse)

	res, err := rpc.BroadcastTxSync(context.Background(), expectedTx)
	assert.NoError(err)
	assert.NotNil(res)
	assert.Equal(expectedResponse.Code, res.Code)
	assert.Equal(bytes.HexBytes(expectedResponse.Data), res.Data)
	assert.Equal(expectedResponse.Log, res.Log)
	assert.Equal(expectedResponse.Codespace, res.Codespace)
	assert.NotEmpty(res.Hash)
	mockApp.AssertExpectations(t)

	err = node.Stop()
	require.NoError(t, err)
}

func TestBroadcastTxCommit(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	expectedTx := []byte("tx data")
	expectedCheckResp := abci.ResponseCheckTx{
		Code:      abci.CodeTypeOK,
		Data:      []byte("data"),
		Log:       "log",
		Info:      "info",
		GasWanted: 0,
		GasUsed:   0,
		Events:    nil,
		Codespace: "space",
	}
	expectedDeliverResp := abci.ResponseDeliverTx{
		Code:      0,
		Data:      []byte("foo"),
		Log:       "bar",
		Info:      "baz",
		GasWanted: 100,
		GasUsed:   10,
		Events:    nil,
		Codespace: "space",
	}

	mockApp, rpc, node := getRPCAndNode(t)
	mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	mockApp.BeginBlock(abci.RequestBeginBlock{})
	mockApp.On("CheckTx", abci.RequestCheckTx{Tx: expectedTx}).Return(expectedCheckResp)
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)
	// in order to broadcast, the node must be started
	err := node.Start()
	require.NoError(err)

	go func() {
		time.Sleep(mockTxProcessingTime)
		err := node.EventBus().PublishEventTx(tmtypes.EventDataTx{TxResult: abci.TxResult{
			Height: 1,
			Index:  0,
			Tx:     expectedTx,
			Result: expectedDeliverResp,
		}})
		require.NoError(err)
	}()

	res, err := rpc.BroadcastTxCommit(context.Background(), expectedTx)
	assert.NoError(err)
	require.NotNil(res)
	assert.Equal(expectedCheckResp, res.CheckTx)
	assert.Equal(expectedDeliverResp, res.DeliverTx)
	mockApp.AssertExpectations(t)

	err = node.Stop()
	require.NoError(err)
}

func TestGetBlock(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	mockApp, rpc, node := getRPCAndNode(t)
	mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	mockApp.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	mockApp.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{})
	mockApp.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)

	err := node.Start()
	require.NoError(err)

	block := getRandomBlock(1, 10)
	_, err = node.Store.SaveBlock(block, &types.Commit{}, nil)
	node.BlockManager.State.SetHeight(block.Header.Height)
	require.NoError(err)

	blockResp, err := rpc.Block(context.Background(), nil)
	require.NoError(err)
	require.NotNil(blockResp)
	assert.NotNil(blockResp.Block)

	err = node.Stop()
	require.NoError(err)
}

func TestValidatedHeight(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	mockApp, rpc, node := getRPCAndNode(t)

	mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	mockApp.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)

	err := node.Start()
	require.NoError(err)

	ptr := func(i int64) *int64 { return &i }

	tests := []struct {
		name            string
		validatedHeight uint64
		nodeHeight      uint64
		submittedHeight uint64
		queryHeight     *int64
		result          client.BlockValidationStatus
	}{
		{
			name:            "SL validated height",
			validatedHeight: 10,
			nodeHeight:      15,
			queryHeight:     ptr(10),
			submittedHeight: 10,
			result:          client.SLValidated,
		},
		{
			name:            "P2P validated height",
			validatedHeight: 10,
			nodeHeight:      15,
			queryHeight:     ptr(13),
			submittedHeight: 10,
			result:          client.P2PValidated,
		},
		{
			name:            "Non validated height",
			validatedHeight: 10,
			nodeHeight:      15,
			queryHeight:     ptr(20),
			submittedHeight: 10,
			result:          client.NotValidated,
		},
		{
			name:            "Invalid height",
			validatedHeight: 1,
			nodeHeight:      5,
			queryHeight:     ptr(-1),
			submittedHeight: 10,
			result:          -1,
		},
		{
			name:            "Nil height",
			validatedHeight: 1,
			nodeHeight:      5,
			queryHeight:     nil,
			submittedHeight: 10,
			result:          -1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			node.BlockManager.SettlementValidator.UpdateLastValidatedHeight(test.validatedHeight)
			node.BlockManager.LastSettlementHeight.Store(test.submittedHeight)

			node.BlockManager.State.SetHeight(test.nodeHeight)

			validationResponse, err := rpc.BlockValidated(test.queryHeight)
			require.NoError(err)
			require.NotNil(validationResponse)
			assert.Equal(test.result, validationResponse.Result)
		})
	}

	err = node.Stop()
	require.NoError(err)
}

func TestGetCommit(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	mockApp, rpc, node := getRPCAndNode(t)

	mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	mockApp.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)

	blocks := []*types.Block{getRandomBlock(1, 5), getRandomBlock(2, 6), getRandomBlock(3, 8), getRandomBlock(4, 10)}

	err := node.Start()
	require.NoError(err)

	for _, b := range blocks {
		_, err = node.Store.SaveBlock(b, &types.Commit{Height: b.Header.Height}, nil)
		node.BlockManager.State.SetHeight(b.Header.Height)
		require.NoError(err)
	}
	t.Run("Fetch all commits", func(t *testing.T) {
		for _, b := range blocks {
			h := int64(b.Header.Height)
			commit, err := rpc.Commit(context.Background(), &h)
			require.NoError(err)
			require.NotNil(commit)
			assert.Equal(b.Header.Height, uint64(commit.Height))
		}
	})

	t.Run("Fetch commit for nil height", func(t *testing.T) {
		commit, err := rpc.Commit(context.Background(), nil)
		require.NoError(err)
		require.NotNil(commit)
		assert.Equal(blocks[3].Header.Height, uint64(commit.Height))
	})

	err = node.Stop()
	require.NoError(err)
}

// Ensures the results of the commit and validators queries are consistent wrt. val set hash
func TestValidatorSetHashConsistency(t *testing.T) {
	require := require.New(t)
	mockApp, rpc, node := getRPCAndNode(t)

	mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	mockApp.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)

	v := tmtypes.NewValidator(ed25519.GenPrivKey().PubKey(), 1)
	s := types.NewSequencerFromValidator(*v)
	node.BlockManager.State.SetProposer(s)

	b := getRandomBlock(1, 10)
	copy(b.Header.SequencerHash[:], node.BlockManager.State.GetProposerHash())

	err := node.Start()
	require.NoError(err)

	_, err = node.Store.SaveBlock(b, &types.Commit{Height: b.Header.Height}, nil)
	node.BlockManager.State.SetHeight(b.Header.Height)
	require.NoError(err)

	batch := node.Store.NewBatch()
	batch, err = node.Store.SaveProposer(b.Header.Height, *node.BlockManager.State.GetProposer(), batch)
	require.NoError(err)
	err = batch.Commit()
	require.NoError(err)

	h := int64(b.Header.Height)
	commit, err := rpc.Commit(context.Background(), &h)
	require.NoError(err)
	require.NotNil(commit)

	vals, err := rpc.Validators(context.Background(), &h, nil, nil)
	require.NoError(err)

	valsRes, err := tmtypes.ValidatorSetFromExistingValidators(vals.Validators)
	require.NoError(err)
	hash := valsRes.Hash()
	ok := stdbytes.Equal(commit.ValidatorsHash, hash)
	require.True(ok)
}

func TestBlockSearch(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	mockApp, rpc, node := getRPCAndNode(t)

	mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	mockApp.On("Commit", mock.Anything).Return(abci.ResponseCommit{})

	heights := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, h := range heights {
		block := getRandomBlock(uint64(h), 5)
		_, err := node.Store.SaveBlock(block, &types.Commit{
			Height:     uint64(h),
			HeaderHash: block.Header.Hash(),
		}, nil)
		require.NoError(err)
	}
	indexBlocks(t, node, heights)

	tests := []struct {
		query      string
		page       int
		perPage    int
		totalCount int
		orderBy    string
	}{
		{
			query:      "block.height >= 1 AND end_event.foo <= 5",
			page:       1,
			perPage:    5,
			totalCount: 5,
			orderBy:    "asc",
		},
		{
			query:      "block.height >= 2 AND end_event.foo <= 10",
			page:       1,
			perPage:    3,
			totalCount: 9,
			orderBy:    "desc",
		},
		{
			query:      "begin_event.proposer = 'FCAA001' AND end_event.foo <= 5",
			page:       1,
			perPage:    5,
			totalCount: 5,
			orderBy:    "asc",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.query, func(t *testing.T) {
			result, err := rpc.BlockSearch(context.Background(), test.query, &test.page, &test.perPage, test.orderBy)
			require.NoError(err)
			assert.Equal(test.totalCount, result.TotalCount)
			assert.Len(result.Blocks, test.perPage)
		})

	}
}

func TestGetBlockByHash(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	mockApp, rpc, node := getRPCAndNode(t)
	mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	mockApp.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	mockApp.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{})
	mockApp.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
	mockApp.On("Info", mock.Anything).Return(abci.ResponseInfo{LastBlockHeight: 0, LastBlockAppHash: []byte{0}})
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)

	err := node.Start()
	require.NoError(err)

	block := getRandomBlock(1, 10)
	_, err = node.Store.SaveBlock(block, &types.Commit{}, nil)
	require.NoError(err)
	abciBlock, err := types.ToABCIBlock(block)
	require.NoError(err)

	height := int64(block.Header.Height)
	retrievedBlock, err := rpc.Block(context.Background(), &height)
	require.NoError(err)
	require.NotNil(retrievedBlock)
	assert.Equal(abciBlock, retrievedBlock.Block)
	assert.Equal(abciBlock.Hash(), retrievedBlock.Block.Header.Hash())

	blockHash := block.Header.Hash()
	blockResp, err := rpc.BlockByHash(context.Background(), blockHash[:])
	require.NoError(err)
	require.NotNil(blockResp)

	assert.NotNil(blockResp.Block)

	err = node.Stop()
	require.NoError(err)
}

func TestTx(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	mockApp, rpc, node := getRPCAndNodeSequencer(t)

	require.NotNil(rpc)
	mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	mockApp.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:         "mock",
			DrsVersion: 0,
		},
		ConsensusParamUpdates: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas:   100,
				MaxBytes: 100,
			},
		},
	})
	mockApp.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
	mockApp.On("DeliverTx", mock.Anything).Return(abci.ResponseDeliverTx{})
	mockApp.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	mockApp.On("Info", mock.Anything).Return(abci.ResponseInfo{LastBlockHeight: 0, LastBlockAppHash: []byte{0}})
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)

	err := node.Start()
	require.NoError(err)

	tx1 := tmtypes.Tx("tx1")
	res, err := rpc.BroadcastTxSync(context.Background(), tx1)
	assert.NoError(err)
	assert.NotNil(res)

	time.Sleep(1 * time.Second)

	resTx, errTx := rpc.Tx(context.Background(), res.Hash, true)
	assert.NotNil(resTx)
	assert.NoError(errTx)
	assert.EqualValues(tx1, resTx.Tx)
	assert.EqualValues(res.Hash, resTx.Hash)

	tx2 := tmtypes.Tx("tx2")
	resTx, errTx = rpc.Tx(context.Background(), tx2.Hash(), true)
	assert.Nil(resTx)
	assert.Error(errTx)
}

func TestUnconfirmedTxs(t *testing.T) {
	tx1 := tmtypes.Tx("tx1")
	tx2 := tmtypes.Tx("another tx")

	cases := []struct {
		name               string
		txs                []tmtypes.Tx
		expectedCount      int
		expectedTotal      int
		expectedTotalBytes int
	}{
		{"no txs", nil, 0, 0, 0},
		{"one tx", []tmtypes.Tx{tx1}, 1, 1, len(tx1)},
		{"two txs", []tmtypes.Tx{tx1, tx2}, 2, 2, len(tx1) + len(tx2)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mockApp, rpc, node := getRPCAndNode(t)
			mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
			mockApp.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
			gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
			mockApp.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)

			err := node.Start()
			require.NoError(err)

			for _, tx := range c.txs {
				res, err := rpc.BroadcastTxAsync(context.Background(), tx)
				assert.NoError(err)
				assert.NotNil(res)
			}

			numRes, err := rpc.NumUnconfirmedTxs(context.Background())
			assert.NoError(err)
			assert.NotNil(numRes)
			assert.EqualValues(c.expectedCount, numRes.Count)
			assert.EqualValues(c.expectedTotal, numRes.Total)
			assert.EqualValues(c.expectedTotalBytes, numRes.TotalBytes)

			limit := -1
			txRes, err := rpc.UnconfirmedTxs(context.Background(), &limit)
			assert.NoError(err)
			assert.NotNil(txRes)
			assert.EqualValues(c.expectedCount, txRes.Count)
			assert.EqualValues(c.expectedTotal, txRes.Total)
			assert.EqualValues(c.expectedTotalBytes, txRes.TotalBytes)
			assert.Len(txRes.Txs, c.expectedCount)
		})
	}
}

func TestUnconfirmedTxsLimit(t *testing.T) {
	t.Skip("Test disabled because of known bug")
	// there's a bug in mempool implementation - count should be 1
	// TODO(tzdybal): uncomment after resolving https://github.com/dymensionxyz/dymint/issues/191

	assert := assert.New(t)
	require := require.New(t)

	mockApp, rpc, node := getRPCAndNode(t)
	mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	mockApp.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})

	err := node.Start()
	require.NoError(err)

	tx1 := tmtypes.Tx("tx1")
	tx2 := tmtypes.Tx("another tx")

	res, err := rpc.BroadcastTxAsync(context.Background(), tx1)
	assert.NoError(err)
	assert.NotNil(res)

	res, err = rpc.BroadcastTxAsync(context.Background(), tx2)
	assert.NoError(err)
	assert.NotNil(res)

	limit := 1
	txRes, err := rpc.UnconfirmedTxs(context.Background(), &limit)
	assert.NoError(err)
	assert.NotNil(txRes)
	assert.EqualValues(1, txRes.Count)
	assert.EqualValues(2, txRes.Total)
	assert.EqualValues(len(tx1)+len(tx2), txRes.TotalBytes)
	assert.Len(txRes.Txs, limit)
	assert.Contains(txRes.Txs, tx1)
	assert.NotContains(txRes.Txs, tx2)
}

func TestConsensusState(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	_, rpc := getRPC(t)
	require.NotNil(rpc)

	resp1, err := rpc.ConsensusState(context.Background())
	assert.Nil(resp1)
	assert.ErrorIs(err, client.ErrConsensusStateNotAvailable)

	resp2, err := rpc.DumpConsensusState(context.Background())
	assert.Nil(resp2)
	assert.ErrorIs(err, client.ErrConsensusStateNotAvailable)
}

func TestBlockchainInfo(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	mockApp, rpc, node := getRPCAndNode(t)
	mockApp.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	mockApp.On("Commit", mock.Anything).Return(abci.ResponseCommit{})

	heights := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, h := range heights {
		block := getRandomBlock(uint64(h), 5)
		_, err := node.Store.SaveBlock(block, &types.Commit{
			Height:     uint64(h),
			HeaderHash: block.Header.Hash(),
		}, nil)
		require.NoError(err)
		node.BlockManager.State.SetHeight(block.Header.Height)
	}

	tests := []struct {
		desc string
		min  int64
		max  int64
		exp  []*tmtypes.BlockMeta
		err  bool
	}{
		{
			desc: "min = 1 and max = 5",
			min:  1,
			max:  5,
			exp:  []*tmtypes.BlockMeta{getBlockMeta(node, 1), getBlockMeta(node, 5)},
			err:  false,
		}, {
			desc: "min height is 0",
			min:  0,
			max:  10,
			exp:  []*tmtypes.BlockMeta{getBlockMeta(node, 1), getBlockMeta(node, 10)},
			err:  false,
		}, {
			desc: "max height is out of range",
			min:  0,
			max:  15,
			exp:  []*tmtypes.BlockMeta{getBlockMeta(node, 1), getBlockMeta(node, 10)},
			err:  false,
		}, {
			desc: "negative min height",
			min:  -1,
			max:  11,
			exp:  nil,
			err:  true,
		}, {
			desc: "negative max height",
			min:  1,
			max:  -1,
			exp:  nil,
			err:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result, err := rpc.BlockchainInfo(context.Background(), test.min, test.max)
			if test.err {
				require.Error(err)
			} else {
				require.NoError(err)
				assert.Equal(result.LastHeight, heights[9])
				assert.Contains(result.BlockMetas, test.exp[0])
				assert.Contains(result.BlockMetas, test.exp[1])
			}
		})
	}
}

// TestValidatorSetHandling tests that EndBlock updates are ignored and the validator set is fetched from the state
func TestValidatorSetHandling(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	app := &tmmocks.MockApplication{}
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	app.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	app.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	app.On("Info", mock.Anything).Return(abci.ResponseInfo{LastBlockHeight: 0, LastBlockAppHash: []byte{0}})

	key, _, _ := crypto.GenerateEd25519Key(crand.Reader)
	signingKey, proposerPubKey, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(err)

	proposerPubKeyBytes, err := proposerPubKey.Raw()
	require.NoError(err)

	vKeys := make([]tmcrypto.PrivKey, 1)
	genesisValidators := make([]tmtypes.GenesisValidator, len(vKeys))
	for i := 0; i < len(vKeys); i++ {
		vKeys[i] = ed25519.GenPrivKey()
		genesisValidators[i] = tmtypes.GenesisValidator{Address: vKeys[i].PubKey().Address(), PubKey: vKeys[i].PubKey(), Power: int64(i + 100), Name: "one"}
	}

	// dummy pubkey, we don't care about the actual key
	pbValKey, err := encoding.PubKeyToProto(vKeys[0].PubKey())
	require.NoError(err)
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:         "mock",
			DrsVersion: 0,
		},
		ConsensusParamUpdates: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas:   100,
				MaxBytes: 100,
			},
		},
		ValidatorUpdates: []abci.ValidatorUpdate{{PubKey: pbValKey, Power: 100}},
	})

	waitCh := make(chan interface{})

	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{}).Times(5)
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{}).Run(func(args mock.Arguments) {
		waitCh <- nil
	})

	nodeConfig := config.NodeConfig{
		SettlementLayer: "mock",
		P2PConfig: config.P2PConfig{
			ListenAddress:                config.DefaultListenAddress,
			BootstrapNodes:               "",
			GossipSubCacheSize:           50,
			BootstrapRetryTime:           30 * time.Second,
			DiscoveryEnabled:             true,
			BlockSyncRequestIntervalTime: 30 * time.Second,
		},
		BlockManagerConfig: config.BlockManagerConfig{
			BlockTime:                  10 * time.Millisecond,
			BatchSubmitTime:            60 * time.Second,
			BatchSubmitBytes:           1000,
			MaxSkewTime:                24 * time.Hour,
			SequencerSetUpdateInterval: config.DefaultSequencerSetUpdateInterval,
		},
		SettlementConfig: settlement.Config{
			ProposerPubKey: hex.EncodeToString(proposerPubKeyBytes),
		},
	}

	node, err := node.NewNode(
		context.Background(),
		nodeConfig,
		key,
		signingKey,
		proxy.NewLocalClientCreator(app),
		testutil.GenerateGenesis(0),
		"",
		log.TestingLogger(),
		mempool.NopMetrics(),
	)

	require.NoError(err)
	require.NotNil(node)

	rpc := client.NewClient(node)
	require.NotNil(rpc)

	err = node.Start()
	require.NoError(err)

	defer node.Stop()

	<-waitCh                           // triggered on the 6th commit
	time.Sleep(300 * time.Millisecond) // give time for the sequencers commit to db

	// validator set isn't updated through ABCI anymore
	for h := int64(1); h <= 5; h++ {
		vals, err := rpc.Validators(context.Background(), &h, nil, nil)
		assert.NoError(err)
		assert.NotNil(vals)
		assert.EqualValues(len(genesisValidators), vals.Total)
		assert.Len(vals.Validators, len(genesisValidators))
		assert.EqualValues(vals.BlockHeight, h)
	}

	// check for "latest block"
	vals, err := rpc.Validators(context.Background(), nil, nil, nil)
	assert.NoError(err)
	assert.NotNil(vals)
	assert.EqualValues(len(genesisValidators), vals.Total)
	assert.Len(vals.Validators, len(genesisValidators))
	assert.GreaterOrEqual(vals.BlockHeight, int64(5))
}

// copy-pasted from store/store_test.go
func getRandomBlock(height uint64, nTxs int) *types.Block {
	block := &types.Block{
		Header: types.Header{
			Height:                height,
			Version:               types.Version{Block: testutil.BlockVersion},
			ProposerAddress:       getRandomBytes(20),
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

	tmprotoLC, err := tmtypes.CommitFromProto(&tmproto.Commit{})
	if err != nil {
		return nil
	}
	copy(block.Header.LastCommitHash[:], tmprotoLC.Hash())

	return block
}

func getRandomTx() types.Tx {
	size := rand.Int()%100 + 100
	return types.Tx(getRandomBytes(size))
}

func getRandomBytes(n int) []byte {
	data := make([]byte, n)
	_, _ = crand.Read(data)
	return data
}

func getBlockMeta(node *node.Node, n int64) *tmtypes.BlockMeta {
	b, err := node.Store.LoadBlock(uint64(n))
	if err != nil {
		return nil
	}
	bmeta, err := types.ToABCIBlockMeta(b)
	if err != nil {
		return nil
	}

	return bmeta
}

// getRPC returns a mock application and a new RPC client (non-sequencer mode)
func getRPC(t *testing.T) (*tmmocks.MockApplication, *client.Client) {
	app, rpc, _ := getRPCAndNode(t)
	return app, rpc
}

func getRPCAndNode(t *testing.T) (*tmmocks.MockApplication, *client.Client, *node.Node) {
	return getRPCInternal(t, false)
}

func getRPCAndNodeSequencer(t *testing.T) (*tmmocks.MockApplication, *client.Client, *node.Node) {
	return getRPCInternal(t, true)
}

// getRPC returns a mock application and a new RPC client (non-sequencer mode)
func getRPCInternal(t *testing.T, sequencer bool) (*tmmocks.MockApplication, *client.Client, *node.Node) {
	t.Helper()
	version.DRS = "0"
	require := require.New(t)
	app := &tmmocks.MockApplication{}
	app.On("Info", mock.Anything).Return(expectedInfo)
	key, _, _ := crypto.GenerateEd25519Key(crand.Reader)
	localKey, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(err)

	slSeqKey, pubkey, err := crypto.GenerateEd25519Key(crand.Reader)
	pubkeyBytes, _ := pubkey.Raw()
	proposerKey := hex.EncodeToString(pubkeyBytes)
	require.NoError(err)

	if sequencer {
		localKey = slSeqKey
	}

	config := config.NodeConfig{
		RootDir: "",
		DBPath:  "",
		P2PConfig: config.P2PConfig{
			ListenAddress:                config.DefaultListenAddress,
			BootstrapNodes:               "",
			GossipSubCacheSize:           50,
			BootstrapRetryTime:           30 * time.Second,
			DiscoveryEnabled:             true,
			BlockSyncRequestIntervalTime: 30 * time.Second,
		},
		RPC:           config.RPCConfig{},
		MempoolConfig: *tmcfg.DefaultMempoolConfig(),
		BlockManagerConfig: config.BlockManagerConfig{
			BlockTime:                  100 * time.Millisecond,
			BatchSubmitTime:            60 * time.Second,
			BatchSubmitBytes:           1000,
			MaxSkewTime:                24 * time.Hour,
			SequencerSetUpdateInterval: config.DefaultSequencerSetUpdateInterval,
		},
		DAConfig:        "",
		SettlementLayer: "mock",
		SettlementConfig: settlement.Config{
			ProposerPubKey: proposerKey,
		},
	}
	node, err := node.NewNode(
		context.Background(),
		config,
		key,
		localKey, // this is where sequencer mode is set. if same key as in settlement.Config, it's sequencer
		proxy.NewLocalClientCreator(app),
		testutil.GenerateGenesis(0),
		"",
		log.TestingLogger(),
		mempool.NopMetrics(),
	)
	require.NoError(err)
	require.NotNil(node)

	rpc := client.NewClient(node)
	require.NotNil(rpc)

	return app, rpc, node
}

// From state/indexer/block/kv/kv_test
func indexBlocks(t *testing.T, node *node.Node, heights []int64) {
	t.Helper()

	for _, h := range heights {
		require.NoError(t, node.BlockIndexer.Index(tmtypes.EventDataNewBlockHeader{
			Header: tmtypes.Header{Height: h},
			ResultBeginBlock: abci.ResponseBeginBlock{
				Events: []abci.Event{
					{
						Type: "begin_event",
						Attributes: []abci.EventAttribute{
							{
								Key:   []byte("proposer"),
								Value: []byte("FCAA001"),
								Index: true,
							},
						},
					},
				},
			},
			ResultEndBlock: abci.ResponseEndBlock{
				Events: []abci.Event{
					{
						Type: "end_event",
						Attributes: []abci.EventAttribute{
							{
								Key:   []byte("foo"),
								Value: []byte(fmt.Sprintf("%d", h)),
								Index: true,
							},
						},
					},
				},
			},
		}))
	}
}

func TestMempool2Nodes(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app := &tmmocks.MockApplication{}
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	app.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz})
	app.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	app.On("CheckTx", abci.RequestCheckTx{Tx: []byte("bad")}).Return(abci.ResponseCheckTx{Code: 1})
	app.On("CheckTx", abci.RequestCheckTx{Tx: []byte("good")}).Return(abci.ResponseCheckTx{Code: 0})
	app.On("Info", mock.Anything).Return(expectedInfo)

	key1, _, _ := crypto.GenerateEd25519Key(crand.Reader)
	key2, _, _ := crypto.GenerateEd25519Key(crand.Reader)
	signingKey1, _, _ := crypto.GenerateEd25519Key(crand.Reader)
	signingKey2, _, _ := crypto.GenerateEd25519Key(crand.Reader)
	_, proposerPubkey, _ := crypto.GenerateEd25519Key(crand.Reader)
	proposerPK, err := proposerPubkey.Raw()
	require.NoError(err)

	id1, err := peer.IDFromPrivateKey(key1)
	require.NoError(err)

	genesis := testutil.GenerateGenesis(0)
	node1, err := node.NewNode(context.Background(), config.NodeConfig{
		SettlementLayer: "mock",
		SettlementConfig: settlement.Config{
			ProposerPubKey: hex.EncodeToString(proposerPK),
		},
		P2PConfig: config.P2PConfig{
			ListenAddress:                "/ip4/127.0.0.1/tcp/9001",
			BootstrapNodes:               "",
			GossipSubCacheSize:           50,
			BootstrapRetryTime:           30 * time.Second,
			BlockSyncRequestIntervalTime: 30 * time.Second,
			DiscoveryEnabled:             true,
		},
		BlockManagerConfig: config.BlockManagerConfig{
			BlockTime:                  100 * time.Millisecond,
			BatchSubmitTime:            60 * time.Second,
			BatchSubmitBytes:           1000,
			MaxSkewTime:                24 * time.Hour,
			SequencerSetUpdateInterval: config.DefaultSequencerSetUpdateInterval,
		},
		MempoolConfig: *tmcfg.DefaultMempoolConfig(),
	}, key1, signingKey1, proxy.NewLocalClientCreator(app), genesis, "", log.TestingLogger(), mempool.NopMetrics())
	require.NoError(err)
	require.NotNil(node1)

	node2, err := node.NewNode(context.Background(), config.NodeConfig{
		SettlementLayer: "mock",
		SettlementConfig: settlement.Config{
			ProposerPubKey: hex.EncodeToString(proposerPK),
		},
		BlockManagerConfig: config.BlockManagerConfig{
			BlockTime:                  100 * time.Millisecond,
			BatchSubmitTime:            60 * time.Second,
			BatchSubmitBytes:           1000,
			MaxSkewTime:                24 * time.Hour,
			SequencerSetUpdateInterval: config.DefaultSequencerSetUpdateInterval,
		},
		P2PConfig: config.P2PConfig{
			ListenAddress:                "/ip4/127.0.0.1/tcp/9002",
			BootstrapNodes:               "/ip4/127.0.0.1/tcp/9001/p2p/" + id1.String(),
			BootstrapRetryTime:           30 * time.Second,
			GossipSubCacheSize:           50,
			DiscoveryEnabled:             true,
			BlockSyncRequestIntervalTime: 30 * time.Second,
		},
		MempoolConfig: *tmcfg.DefaultMempoolConfig(),
	}, key2, signingKey2, proxy.NewLocalClientCreator(app), genesis, "", log.TestingLogger(), mempool.NopMetrics())
	require.NoError(err)
	require.NotNil(node1)

	err = node1.Start()
	require.NoError(err)

	err = node2.Start()
	require.NoError(err)

	time.Sleep(1 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	local := client.NewClient(node1)
	require.NotNil(local)

	// broadcast the bad Tx, this should not be propagated or added to the local mempool
	resp, err := local.BroadcastTxSync(ctx, []byte("bad"))
	assert.NoError(err)
	assert.NotNil(resp)
	// broadcast the good Tx, this should be propagated and added to the local mempool
	resp, err = local.BroadcastTxSync(ctx, []byte("good"))
	assert.NoError(err)
	assert.NotNil(resp)
	// broadcast the good Tx again in the same block, this should not be propagated and
	// added to the local mempool
	resp, err = local.BroadcastTxSync(ctx, []byte("good"))
	assert.Error(err)
	assert.Nil(resp)

	txAvailable := node2.Mempool.TxsAvailable()
	select {
	case <-txAvailable:
	case <-ctx.Done():
	}

	assert.Equal(node2.Mempool.SizeBytes(), int64(len("good")))
}
