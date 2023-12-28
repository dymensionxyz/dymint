package celestia_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	mocks "github.com/dymensionxyz/dymint/mocks/da/celestia"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/rollkit/celestia-openrpc/types/state"

	"github.com/tendermint/tendermint/libs/pubsub"
)

const (
	submitPFBFuncName = "SubmitPayForBlob"
)

func TestSubmitBatch(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	configBytes, err := json.Marshal(celestia.CelestiaDefaultConfig)
	require.NoError(err)
	batch := &types.Batch{
		StartHeight: 0,
		EndHeight:   1,
	}
	cases := []struct {
		name                    string
		submitPFBReturn         []interface{}
		sumbitPFDRun            func(args mock.Arguments)
		expectedInclusionHeight int
		expectedHealthEvent     *da.EventDataDAHealthStatus
	}{
		{
			name:                    "TestSubmitPFBResponseCodeSuccess",
			submitPFBReturn:         []interface{}{&state.TxResponse{Code: 0, Height: int64(143)}, nil},
			sumbitPFDRun:            func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			expectedInclusionHeight: 143,
			expectedHealthEvent:     &da.EventDataDAHealthStatus{Healthy: true},
		},
		{
			name:                "TestSubmitPFBErrored",
			submitPFBReturn:     []interface{}{&state.TxResponse{TxHash: "1234"}, errors.New("timeout")},
			sumbitPFDRun:        func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			expectedHealthEvent: &da.EventDataDAHealthStatus{Healthy: false},
		},
	}
	for _, tc := range cases {
		// Create mock clients
		mockRPCClient := mocks.NewCelestiaRPCClient(t)
		// Configure DALC options
		options := []da.Option{
			celestia.WithSubmitRetryDelay(10 * time.Millisecond),
			celestia.WithRPCClient(mockRPCClient),
		}
		// Subscribe to the health status event
		pubsubServer := pubsub.NewServer()
		pubsubServer.Start()
		HealthSubscription, err := pubsubServer.Subscribe(context.Background(), "testSubmitBatch", da.EventQueryDAHealthStatus)
		assert.NoError(err, tc.name)
		// Start the DALC
		dalc := celestia.DataAvailabilityLayerClient{}
		err = dalc.Init(configBytes, pubsubServer, nil, log.TestingLogger(), options...)
		require.NoError(err, tc.name)
		err = dalc.Start()
		require.NoError(err, tc.name)

		mockRPCClient.On(submitPFBFuncName, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.submitPFBReturn...).Run(tc.sumbitPFDRun)

		done := make(chan bool)
		go func() {
			res := dalc.SubmitBatch(batch)
			assert.Equal(res.DAHeight, uint64(tc.expectedInclusionHeight), tc.name)
			time.Sleep(100 * time.Millisecond)
			done <- true
		}()

		select {
		case event := <-HealthSubscription.Out():
			healthStatusEvent := event.Data().(*da.EventDataDAHealthStatus)
			t.Log("got health status event", healthStatusEvent.Healthy)
			assert.Equal(tc.expectedHealthEvent.Healthy, healthStatusEvent.Healthy, tc.name)
		case <-time.After(1 * time.Second):
			t.Error("timeout. expected health status event but didn't get one")
		case <-done:
			t.Error("submit done. expected health status event but didn't get one")
		}
		err = dalc.Stop()
		require.NoError(err, tc.name)
		// Wait for the goroutines to finish before accessing the mock calls
		time.Sleep(3 * time.Second)
		assert.GreaterOrEqual(testutil.CountMockCalls(mockRPCClient.Calls, submitPFBFuncName), 1, tc.name)
	}
}
