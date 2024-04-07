package node_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/stretchr/testify/assert"

	"github.com/libp2p/go-libp2p/core/crypto"
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

	blockManagerConfig := config.BlockManagerConfig{
		BlockBatchSize:          1,
		BlockTime:               1 * time.Second,
		BatchSubmitMaxTime:      60 * time.Second,
		BlockBatchMaxSizeBytes:  1000,
		NamespaceID:             "0102030405060708",
		GossipedBlocksCacheSize: 50,
	}

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
		SettlementConfig:   settlement.Config{ProposerPubKey: proposerKey},
	}
	node, err := node.NewNode(context.Background(), nodeConfig, key, signingKey, proxy.NewLocalClientCreator(app), &types.GenesisDoc{ChainID: "test"}, log.TestingLogger(), mempool.NopMetrics())
	require.NoError(err)
	require.NotNil(node)

	assert.False(node.IsRunning())

	err = node.Start()
	require.NoError(err)
	assert.True(node.IsRunning())
	err = node.Stop()
	assert.NoError(err)
}
