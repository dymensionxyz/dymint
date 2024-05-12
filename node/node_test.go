package node_test

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/config"
	tmmocks "github.com/dymensionxyz/dymint/mocks/github.com/tendermint/tendermint/abci/types"
)

// simply check that node is starting and stopping without panicking
func TestStartup(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// TODO(omritoptix): Test with and without aggregator mode.
	node, err := testutil.CreateNode(false, nil)
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
	app.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{})
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	app.On("Info", mock.Anything).Return(abci.ResponseInfo{})
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	signingKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	anotherKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	rollappID := "rollapp_1234-1"

	nodeConfig := config.NodeConfig{
		RootDir: "",
		DBPath:  "",
		P2P:     config.P2PConfig{},
		RPC:     config.RPCConfig{},
		BlockManagerConfig: config.BlockManagerConfig{
			BlockTime:               100 * time.Millisecond,
			BlockBatchSize:          2,
			BatchSubmitMaxTime:      60 * time.Second,
			BlockBatchMaxSizeBytes:  1000,
			GossipedBlocksCacheSize: 50,
		},
		DALayer:         "mock",
		DAConfig:        "",
		SettlementLayer: "mock",
		SettlementConfig: settlement.Config{
			RollappID: rollappID,
		},
	}
	node, err := node.NewNode(
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
}

func TestHealthStatusEventHandler(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	node, err := testutil.CreateNode(false, nil)
	require.NoError(err)
	require.NotNil(node)

	err = node.Start()
	assert.NoError(err)
	// wait for node to start
	time.Sleep(1 * time.Second)

	slError := errors.New("settlement")
	daError := errors.New("da")

	cases := []struct {
		name                           string
		baseLayerHealthStatusEvent     map[string][]string
		baseLayerHealthStatusEventData interface{}
		expectHealthStatusEventEmitted bool
		expectedError                  error
	}{
		{
			name:                           "settlement layer is healthy and da layer is healthy",
			baseLayerHealthStatusEvent:     settlement.EventHealthStatusList,
			baseLayerHealthStatusEventData: &settlement.EventDataHealth{Error: slError},
			expectHealthStatusEventEmitted: true,
			expectedError:                  slError,
		},
		{
			name:                           "now da also becomes unhealthy",
			baseLayerHealthStatusEvent:     da.EventHealthStatusList,
			baseLayerHealthStatusEventData: &da.EventDataHealth{Error: daError},
			expectHealthStatusEventEmitted: true,
			expectedError:                  daError,
		},
		{
			name:                           "now the settlement layer becomes healthy",
			baseLayerHealthStatusEvent:     settlement.EventHealthStatusList,
			baseLayerHealthStatusEventData: &settlement.EventDataHealth{},
			expectHealthStatusEventEmitted: false,
			expectedError:                  nil,
		},
		{
			name:                           "now the da layer becomes healthy, so we expect the health status to be healthy and the event to be emitted",
			baseLayerHealthStatusEvent:     da.EventHealthStatusList,
			baseLayerHealthStatusEventData: &da.EventDataHealth{},
			expectHealthStatusEventEmitted: true,
			expectedError:                  nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			done := make(chan bool, 1)
			ready := make(chan bool, 1)
			go func() {
				HealthSubscription, err := node.PubsubServer.Subscribe(context.Background(), c.name, events.QueryHealthStatus)
				ready <- true
				assert.NoError(err)
				select {
				case event := <-HealthSubscription.Out():
					if !c.expectHealthStatusEventEmitted {
						t.Error("didn't expect health status event but got one")
					}
					healthStatusEvent := event.Data().(*events.DataHealthStatus)
					assert.ErrorIs(healthStatusEvent.Error, c.expectedError)
					done <- true
					break
				case <-time.After(1 * time.Second):
					if c.expectHealthStatusEventEmitted {
						t.Error("expected health status event but didn't get one")
					}
					done <- true
					break
				}
			}()
			<-ready
			// Emit an event.
			node.PubsubServer.PublishWithEvents(context.Background(), c.baseLayerHealthStatusEventData, c.baseLayerHealthStatusEvent)
			<-done
		})
	}
	err = node.Stop()
	assert.NoError(err)
}
