package node_test

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	tmjson "github.com/tendermint/tendermint/libs/json"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	"github.com/dymensionxyz/dymint/version"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"

	tmcfg "github.com/tendermint/tendermint/config"

	"github.com/dymensionxyz/dymint/config"
	tmmocks "github.com/dymensionxyz/dymint/mocks/github.com/tendermint/tendermint/abci/types"
)

// simply check that node is starting and stopping without panicking
func TestStartup(t *testing.T) {

	version.DRS = "0"
	assert := assert.New(t)
	require := require.New(t)

	// TODO(omritoptix): Test with and without sequencer mode.
	node, err := testutil.CreateNode(false, nil, testutil.GenerateGenesis(0))
	require.NoError(err)
	require.NotNil(node)

	assert.False(node.IsRunning())

	err = node.Start()
	assert.NoError(err)
	defer func() {
		err := node.Stop()
		assert.NoError(err)
	}()
	assert.True(node.IsRunning())
}

func TestMempoolDirectly(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app := &tmmocks.MockApplication{}
	gbdBz, _ := tmjson.Marshal(rollapp.GenesisBridgeData{})
	app.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz}, nil)
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	app.On("Info", mock.Anything).Return(abci.ResponseInfo{})
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	signingKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	anotherKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)

	nodeConfig := config.NodeConfig{
		RootDir: "",
		DBPath:  "",
		P2PConfig: config.P2PConfig{
			ListenAddress:                config.DefaultListenAddress,
			GossipSubCacheSize:           50,
			BootstrapRetryTime:           30 * time.Second,
			BootstrapNodes:               "",
			BlockSyncRequestIntervalTime: 30 * time.Second,
			DiscoveryEnabled:             true,
		},
		RPC:           config.RPCConfig{},
		MempoolConfig: *tmcfg.DefaultMempoolConfig(),
		BlockManagerConfig: config.BlockManagerConfig{
			BlockTime:                  1 * time.Second,
			BatchSubmitTime:            60 * time.Second,
			BatchSubmitBytes:           100000,
			MaxSkewTime:                24 * 7 * time.Hour,
			SequencerSetUpdateInterval: config.DefaultSequencerSetUpdateInterval,
		},
		DAConfig:         "",
		SettlementLayer:  "mock",
		SettlementConfig: settlement.Config{},
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

	err = node.Start()
	require.NoError(err)

	pid, err := peer.IDFromPrivateKey(anotherKey)
	require.NoError(err)
	err = node.Mempool.CheckTx([]byte("tx1"), func(r *abci.Response) {}, mempool.TxInfo{
		SenderID: node.MempoolIDs.GetForPeer(pid),
	})
	require.NoError(err)
	err = node.Mempool.CheckTx([]byte("tx2"), func(r *abci.Response) {}, mempool.TxInfo{
		SenderID: node.MempoolIDs.GetForPeer(pid),
	})
	require.NoError(err)
	time.Sleep(100 * time.Millisecond)
	err = node.Mempool.CheckTx([]byte("tx3"), func(r *abci.Response) {}, mempool.TxInfo{
		SenderID: node.MempoolIDs.GetForPeer(pid),
	})
	require.NoError(err)
	err = node.Mempool.CheckTx([]byte("tx4"), func(r *abci.Response) {}, mempool.TxInfo{
		SenderID: node.MempoolIDs.GetForPeer(pid),
	})
	require.NoError(err)

	time.Sleep(1 * time.Second)

	assert.Equal(int64(4*len("tx*")), node.Mempool.SizeBytes())

	err = node.Stop()
	require.NoError(err)
}
