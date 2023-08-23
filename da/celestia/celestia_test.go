package celestia_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/celestiaorg/go-cnc"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	mocks "github.com/dymensionxyz/dymint/mocks/da/celestia"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/libs/pubsub"
	rpcmock "github.com/tendermint/tendermint/rpc/client/mocks"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

const (
	submitPFBFuncName = "SubmitPFB"
	TxFuncName        = "Tx"
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
		name                      string
		submitPFBReturn           []interface{}
		sumbitPFDRun              func(args mock.Arguments)
		TxFnReturn                []interface{}
		TxFnRun                   func(args mock.Arguments)
		isSubmitBatchAsync        bool
		expectedSubmitPFBMinCalls int
		expectedInclusionHeight   int
		expectedHealthEvent       *da.EventDataDAHealthStatus
	}{
		{
			name:                      "TestSubmitPFBResponseNil",
			submitPFBReturn:           []interface{}{nil, nil},
			sumbitPFDRun:              func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			isSubmitBatchAsync:        true,
			expectedSubmitPFBMinCalls: 2,
			expectedHealthEvent:       &da.EventDataDAHealthStatus{Healthy: false},
		},
		{
			name:                      "TestSubmitPFBResponseCodeSuccess",
			submitPFBReturn:           []interface{}{&cnc.TxResponse{Code: 0, Height: int64(143)}, nil},
			sumbitPFDRun:              func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			isSubmitBatchAsync:        false,
			expectedSubmitPFBMinCalls: 1,
			expectedInclusionHeight:   143,
			expectedHealthEvent:       &da.EventDataDAHealthStatus{Healthy: true},
		},
		{
			name:                      "TestSubmitPFBResponseCodeFailure",
			submitPFBReturn:           []interface{}{&cnc.TxResponse{Code: 1}, nil},
			sumbitPFDRun:              func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			isSubmitBatchAsync:        true,
			expectedSubmitPFBMinCalls: 2,
			expectedHealthEvent:       &da.EventDataDAHealthStatus{Healthy: false},
		},
		{
			name:                      "TestSubmitPFBDelayedInclusion",
			submitPFBReturn:           []interface{}{&cnc.TxResponse{TxHash: "1234"}, errors.New("timeout")},
			sumbitPFDRun:              func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			TxFnReturn:                []interface{}{&coretypes.ResultTx{Hash: bytes.HexBytes("1234"), Height: int64(145)}, nil},
			TxFnRun:                   func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			isSubmitBatchAsync:        false,
			expectedSubmitPFBMinCalls: 1,
			expectedInclusionHeight:   145,
			expectedHealthEvent:       &da.EventDataDAHealthStatus{Healthy: true},
		},
		{
			name:                      "TestSubmitPFBDelayedInclusionTxNotFound",
			submitPFBReturn:           []interface{}{&cnc.TxResponse{TxHash: "1234"}, errors.New("timeout")},
			sumbitPFDRun:              func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			TxFnReturn:                []interface{}{nil, errors.New("notFound")},
			TxFnRun:                   func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			isSubmitBatchAsync:        true,
			expectedSubmitPFBMinCalls: 2,
			expectedHealthEvent:       &da.EventDataDAHealthStatus{Healthy: false},
		},
	}
	for _, tc := range cases {
		// Create mock clients
		rpcmockClient := &rpcmock.Client{}
		mockCNCClient := mocks.NewCNCClientI(t)
		// Configure DALC options
		options := []da.Option{
			celestia.WithTxPollingRetryDelay(1 * time.Second),
			celestia.WithTxPollingAttempts(1),
			celestia.WithSubmitRetryDelay(30 * time.Millisecond),
			celestia.WithCNCClient(mockCNCClient),
			celestia.WithRPCClient(rpcmockClient),
		}
		// Subscribe to the health status event
		pubsubServer := pubsub.NewServer()
		pubsubServer.Start()
		HealthSubscription, err := pubsubServer.Subscribe(context.Background(), "testSubmitBatch", da.EventQueryDAHealthStatus)
		assert.NoError(err)
		// Start the DALC
		dalc := celestia.DataAvailabilityLayerClient{}
		err = dalc.Init(configBytes, pubsubServer, nil, log.TestingLogger(), options...)
		require.NoError(err)
		err = dalc.Start()
		require.NoError(err)
		// Set the mock functions
		mockCNCClient.On(submitPFBFuncName, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.submitPFBReturn...).Run(tc.sumbitPFDRun)
		rpcmockClient.On(TxFuncName, mock.Anything, mock.Anything, mock.Anything).Return(tc.TxFnReturn...).Run(tc.TxFnRun)
		if tc.isSubmitBatchAsync {
			go dalc.SubmitBatch(batch)
			time.Sleep(100 * time.Millisecond)
		} else {
			res := dalc.SubmitBatch(batch)
			assert.Equal(res.DAHeight, uint64(tc.expectedInclusionHeight))
		}
		done := make(chan bool)
		go func() {
			select {
			case event := <-HealthSubscription.Out():
				healthStatusEvent := event.Data().(*da.EventDataDAHealthStatus)
				assert.Equal(tc.expectedHealthEvent.Healthy, healthStatusEvent.Healthy)
				done <- true
				break
			case <-time.After(100 * time.Millisecond):
				t.Error("expected health status event but didn't get one")
				done <- true
				break
			}
		}()
		<-done
		err = dalc.Stop()
		require.NoError(err)
		// Wait for the goroutines to finish before accessing the mock calls
		time.Sleep(3 * time.Second)
		t.Log("Verifying mock calls")
		assert.GreaterOrEqual(testutil.CountMockCalls(mockCNCClient.Calls, submitPFBFuncName), tc.expectedSubmitPFBMinCalls)
	}
}
