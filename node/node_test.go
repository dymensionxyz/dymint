package node

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/mocks"
)

// simply check that node is starting and stopping without panicking
func TestStartup(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// TODO(omritoptix): Test with and without aggregator mode.
	node, err := CreateNode(false, nil)
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

	app := &mocks.Application{}
	app.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{})
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	signingKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	anotherKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)

	nodeConfig := config.NodeConfig{
		RootDir:            "",
		DBPath:             "",
		P2P:                config.P2PConfig{},
		RPC:                config.RPCConfig{},
		Aggregator:         false,
		BlockManagerConfig: config.BlockManagerConfig{BlockTime: 100 * time.Millisecond, BlockBatchSize: 2},
		DALayer:            "mock",
		DAConfig:           "",
		SettlementLayer:    "mock",
		SettlementConfig:   settlement.Config{},
	}
	node, err := NewNode(context.Background(), nodeConfig, key, signingKey, proxy.NewLocalClientCreator(app), &types.GenesisDoc{ChainID: "test"}, log.TestingLogger())
	require.NoError(err)
	require.NotNil(node)

	err = node.Start()
	require.NoError(err)

	pid, err := peer.IDFromPrivateKey(anotherKey)
	require.NoError(err)
	err = node.Mempool.CheckTx([]byte("tx1"), func(r *abci.Response) {}, mempool.TxInfo{
		SenderID: node.mempoolIDs.GetForPeer(pid),
	})
	require.NoError(err)
	err = node.Mempool.CheckTx([]byte("tx2"), func(r *abci.Response) {}, mempool.TxInfo{
		SenderID: node.mempoolIDs.GetForPeer(pid),
	})
	require.NoError(err)
	time.Sleep(100 * time.Millisecond)
	err = node.Mempool.CheckTx([]byte("tx3"), func(r *abci.Response) {}, mempool.TxInfo{
		SenderID: node.mempoolIDs.GetForPeer(pid),
	})
	require.NoError(err)
	err = node.Mempool.CheckTx([]byte("tx4"), func(r *abci.Response) {}, mempool.TxInfo{
		SenderID: node.mempoolIDs.GetForPeer(pid),
	})
	require.NoError(err)

	time.Sleep(1 * time.Second)

	assert.Equal(int64(4*len("tx*")), node.Mempool.SizeBytes())
}

func TestHealthStatusEventHandler(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	node, err := CreateNode(false, nil)
	require.NoError(err)
	require.NotNil(node)

	err = node.Start()
	assert.NoError(err)
	// wait for node to start
	time.Sleep(100 * time.Millisecond)

	slUnealthyError := errors.New("settlement layer is unhealthy")
	daUnealthyError := errors.New("da layer is unhealthy")

	cases := []struct {
		name                           string
		baseLayerHealthStatusEvent     map[string][]string
		baseLayerHealthStatusEventData interface{}
		expectedHealthStatus           bool
		expectHealthStatusEventEmitted bool
		expectedError                  error
	}{
		// settlement layer is healthy, DA layer is healthy
		{
			name:                           "TestSettlementUnhealthyDAHealthy",
			baseLayerHealthStatusEvent:     map[string][]string{settlement.EventTypeKey: {settlement.EventSettlementHealthStatus}},
			baseLayerHealthStatusEventData: &settlement.EventDataSettlementHealthStatus{Healthy: false, Error: slUnealthyError},
			expectedHealthStatus:           false,
			expectHealthStatusEventEmitted: true,
			expectedError:                  slUnealthyError,
		},
		// Now da also becomes unhealthy
		{
			name:                           "TestDAUnhealthySettlementUnhealthy",
			baseLayerHealthStatusEvent:     map[string][]string{da.EventTypeKey: {da.EventDAHealthStatus}},
			baseLayerHealthStatusEventData: &da.EventDataDAHealthStatus{Healthy: false, Error: daUnealthyError},
			expectedHealthStatus:           false,
			expectHealthStatusEventEmitted: true,
			expectedError:                  daUnealthyError,
		},
		// Now the settlement layer becomes healthy
		{
			name:                           "TestSettlementHealthyDAHealthy",
			baseLayerHealthStatusEvent:     map[string][]string{settlement.EventTypeKey: {settlement.EventSettlementHealthStatus}},
			baseLayerHealthStatusEventData: &settlement.EventDataSettlementHealthStatus{Healthy: true, Error: nil},
			expectedHealthStatus:           false,
			expectHealthStatusEventEmitted: false,
			expectedError:                  nil,
		},
		// Now the da layer becomes healthy so we expect the health status to be healthy and the event to be emitted
		{
			name:                           "TestDAHealthySettlementHealthy",
			baseLayerHealthStatusEvent:     map[string][]string{da.EventTypeKey: {da.EventDAHealthStatus}},
			baseLayerHealthStatusEventData: &da.EventDataDAHealthStatus{Healthy: true, Error: nil},
			expectedHealthStatus:           true,
			expectHealthStatusEventEmitted: true,
			expectedError:                  nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			done := make(chan bool, 1)
			go func() {
				HealthSubscription, err := node.pubsubServer.Subscribe(node.ctx, c.name, events.EventQueryHealthStatus)
				assert.NoError(err)
				select {
				case event := <-HealthSubscription.Out():
					if !c.expectHealthStatusEventEmitted {
						t.Error("didn't expect health status event but got one")
					}
					healthStatusEvent := event.Data().(*events.EventDataHealthStatus)
					assert.Equal(c.expectedHealthStatus, healthStatusEvent.Healthy)
					assert.Equal(c.expectedError, healthStatusEvent.Error)
					done <- true
					break
				case <-time.After(500 * time.Millisecond):
					if c.expectHealthStatusEventEmitted {
						t.Error("expected health status event but didn't get one")
					}
					done <- true
					break
				}
			}()
			// Emit an event.
			node.pubsubServer.PublishWithEvents(context.Background(), c.baseLayerHealthStatusEventData, c.baseLayerHealthStatusEvent)
			<-done
		})
	}
	node.Stop()
}
