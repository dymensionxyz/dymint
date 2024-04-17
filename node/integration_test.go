package node

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/mocks"
)

func TestAggregatorMode(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app := &mocks.Application{}
	app.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{})
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	app.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	app.On("DeliverTx", mock.Anything).Return(abci.ResponseDeliverTx{})
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{})
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
	app.On("Info", mock.Anything).Return(abci.ResponseInfo{LastBlockHeight: 0, LastBlockAppHash: []byte{0}})

	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	signingKey, pubkey, _ := crypto.GenerateEd25519Key(rand.Reader)
	pubkeyBytes, _ := pubkey.Raw()
	proposerKey := hex.EncodeToString(pubkeyBytes)

	anotherKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)

	blockManagerConfig := config.BlockManagerConfig{
		BlockBatchSize:          1,
		BlockTime:               1 * time.Second,
		BatchSubmitMaxTime:      60 * time.Second,
		BlockBatchMaxSizeBytes:  1000,
		NamespaceID:             "0102030405060708",
		GossipedBlocksCacheSize: 50,
	}

	rollappID := "rollapp_1234-1"

	nodeConfig := config.NodeConfig{
		RootDir:            "",
		DBPath:             "",
		P2P:                config.P2PConfig{},
		RPC:                config.RPCConfig{},
		Aggregator:         true,
		BlockManagerConfig: blockManagerConfig,
		DALayer:            "mock",
		DAConfig:           "",
		SettlementLayer:    "mock",
		SettlementConfig:   settlement.Config{ProposerPubKey: proposerKey, RollappID: rollappID},
	}
	node, err := NewNode(
		context.Background(),
		nodeConfig,
		key,
		signingKey,
		proxy.NewLocalClientCreator(app),
		&types.GenesisDoc{ChainID: rollappID},
		log.TestingLogger(),
		mempool.NopMetrics(),
	)
	require.NoError(err)
	require.NotNil(node)

	assert.False(node.IsRunning())

	err = node.Start()
	require.NoError(err)
	defer func() {
		err := node.Stop()
		assert.NoError(err)
	}()
	assert.True(node.IsRunning())

	pid, err := peer.IDFromPrivateKey(anotherKey)
	require.NoError(err)
	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			// TODO(omritoptix): Not sure exactly what does it test as we do nothing with this channel
			default:
				node.incomingTxCh <- &p2p.GossipMessage{Data: []byte(time.Now().String()), From: pid}
				time.Sleep(time.Duration(mrand.Uint32()%20) * time.Millisecond)
			}
		}
	}()
	time.Sleep(3 * time.Second)
	cancel()
}

// TestTxGossipingAndAggregation setups a network of nodes, with single aggregator and multiple producers.
// Nodes should gossip transactions and aggregator node should produce blocks.
// func TestTxGossipingAndAggregation(t *testing.T) {
// 	assert := assert.New(t)
// 	require := require.New(t)

// 	var wg sync.WaitGroup
// 	clientNodes := 4
// 	nodes, apps := createNodes(clientNodes+1, &wg, t)

// 	wg.Add((clientNodes + 1) * clientNodes)
// 	for _, n := range nodes {
// 		require.NoError(n.Start())
// 	}

// 	time.Sleep(1 * time.Second)

// 	for i := 1; i < len(nodes); i++ {
// 		data := strconv.Itoa(i) + time.Now().String()
// 		require.NoError(nodes[i].P2P.GossipTx(context.TODO(), []byte(data)))
// 	}

// 	timeout := time.NewTimer(time.Second * 30)
// 	doneChan := make(chan struct{})
// 	go func() {
// 		defer close(doneChan)
// 		wg.Wait()
// 	}()
// 	select {
// 	case <-doneChan:
// 	case <-timeout.C:
// 		t.Fatal("failing after timeout")
// 	}

// 	for _, n := range nodes {
// 		require.NoError(n.Stop())
// 	}
// 	aggApp := apps[0]
// 	apps = apps[1:]

// 	aggApp.AssertNumberOfCalls(t, "DeliverTx", clientNodes)
// 	aggApp.AssertExpectations(t)

// 	for i, app := range apps {
// 		app.AssertNumberOfCalls(t, "DeliverTx", clientNodes)
// 		app.AssertExpectations(t)

// 		// assert that we have most of the blocks from aggregator
// 		beginCnt := 0
// 		endCnt := 0
// 		commitCnt := 0
// 		for _, call := range app.Calls {
// 			switch call.Method {
// 			case "BeginBlock":
// 				beginCnt++
// 			case "EndBlock":
// 				endCnt++
// 			case "Commit":
// 				commitCnt++
// 			}
// 		}
// 		aggregatorHeight := nodes[0].Store.Height()
// 		adjustedHeight := int(aggregatorHeight - 3) // 3 is completely arbitrary
// 		assert.GreaterOrEqual(beginCnt, adjustedHeight)
// 		assert.GreaterOrEqual(endCnt, adjustedHeight)
// 		assert.GreaterOrEqual(commitCnt, adjustedHeight)

// 		// assert that all blocks known to node are same as produced by aggregator
// 		for h := uint64(1); h <= nodes[i].Store.Height(); h++ {
// 			nodeBlock, err := nodes[i].Store.LoadBlock(h)
// 			require.NoError(err)
// 			aggBlock, err := nodes[0].Store.LoadBlock(h)
// 			require.NoError(err)
// 			assert.Equal(aggBlock, nodeBlock)
// 		}
// 	}
// }

// func createNodes(num int, wg *sync.WaitGroup, t *testing.T) ([]*Node, []*mocks.Application) {
// 	t.Helper()

// 	// create keys first, as they are required for P2P connections
// 	keys := make([]crypto.PrivKey, num)
// 	for i := 0; i < num; i++ {
// 		keys[i], _, _ = crypto.GenerateEd25519Key(rand.Reader)
// 	}

// 	nodes := make([]*Node, num)
// 	apps := make([]*mocks.Application, num)
// 	dalc := &mockda.DataAvailabilityLayerClient{}
// 	_ = dalc.Init(nil, store.NewDefaultInMemoryKVStore(), log.TestingLogger())
// 	_ = dalc.Start()
// 	// Create a pubsub server
// 	pubsub := pubsub.NewServer()
// 	pubsub.Start()
// 	// Create a settlement layer client
// 	settlementlc := slregistry.GetClient(slregistry.ClientMock)
// 	_ = settlementlc.Init(nil, pubsub, log.TestingLogger())
// 	_ = settlementlc.Start()

// 	nodes[0], apps[0] = createNode(0, true, dalc, pubsub, settlementlc, keys, wg, t)
// 	for i := 1; i < num; i++ {
// 		nodes[i], apps[i] = createNode(i, false, dalc, pubsub, settlementlc, keys, wg, t)
// 	}

// 	return nodes, apps
// }

// func createNode(n int, aggregator bool, dalc da.DataAvailabilityLayerClient, pubsub *pubsub.Server,
// 	settlementlc settlement.LayerClient, keys []crypto.PrivKey, wg *sync.WaitGroup, t *testing.T) (*Node, *mocks.Application) {
// 	t.Helper()
// 	require := require.New(t)
// 	// nodes will listen on consecutive ports on local interface
// 	// random connections to other nodes will be added
// 	startPort := 10000
// 	p2pConfig := config.P2PConfig{
// 		ListenAddress: "/ip4/127.0.0.1/tcp/" + strconv.Itoa(startPort+n),
// 	}
// 	bmConfig := config.BlockManagerConfig{
// 		BlockTime:         1 * time.Second,
// 		NamespaceID:       [8]byte{8, 7, 6, 5, 4, 3, 2, 1},
// 		BatchSyncInterval: time.Second,
// 	}
// 	for i := 0; i < len(keys); i++ {
// 		if i == n {
// 			continue
// 		}
// 		r := i
// 		id, err := peer.IDFromPrivateKey(keys[r])
// 		require.NoError(err)
// 		p2pConfig.Seeds += "/ip4/127.0.0.1/tcp/" + strconv.Itoa(startPort+r) + "/p2p/" + id.Pretty() + ","
// 	}
// 	p2pConfig.Seeds = strings.TrimSuffix(p2pConfig.Seeds, ",")

// 	app := &mocks.Application{}
// 	app.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{})
// 	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
// 	app.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
// 	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{})
// 	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
// 	app.On("DeliverTx", mock.Anything).Return(abci.ResponseDeliverTx{}).Run(func(args mock.Arguments) {
// 		wg.Done()
// 	})

// 	signingKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
// 	node, err := NewNode(
// 		context.Background(),
// 		config.NodeConfig{
// 			P2P:                p2pConfig,
// 			DALayer:            "mock",
// 			SettlementLayer:    "mock",
// 			Aggregator:         aggregator,
// 			BlockManagerConfig: bmConfig,
// 		},
// 		keys[n],
// 		signingKey,
// 		proxy.NewLocalClientCreator(app),
// 		&types.GenesisDoc{ChainID: "test"},
// 		log.TestingLogger().With("node", n))
// 	require.NoError(err)
// 	require.NotNil(node)

// 	// use same, common DALC so nodes can share data
// 	node.dalc = dalc
// 	node.blockManager.SetDALC(dalc)

// 	return node, app
// }
