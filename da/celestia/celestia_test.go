package celestia_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/celestiaorg/go-cnc"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	"github.com/dymensionxyz/dymint/log/test"
	mocks "github.com/dymensionxyz/dymint/mocks/da/celestia"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/libs/pubsub"
	rpcmock "github.com/tendermint/tendermint/rpc/client/mocks"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

const (
	submitPFDFuncName = "SubmitPFD"
	TxFuncName        = "Tx"
)

func TestSubmitBatch(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	configBytes, err := json.Marshal(celestia.Config{})
	require.NoError(err)
	batch := &types.Batch{
		StartHeight: 0,
		EndHeight:   1,
	}
	cases := []struct {
		name                      string
		submitPFDReturn           []interface{}
		sumbitPFDRun              func(args mock.Arguments)
		TxFnReturn                []interface{}
		isSubmitBatchAsync        bool
		expectedSubmitPFDMinCalls int
		expectedInclusionHeight   int
	}{
		{
			name:                      "TestSubmitPFDResponseNil",
			submitPFDReturn:           []interface{}{nil, nil},
			sumbitPFDRun:              func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			isSubmitBatchAsync:        true,
			expectedSubmitPFDMinCalls: 2,
		},
		{
			name:                      "TestSubmitPFDResponseCodeSuccess",
			submitPFDReturn:           []interface{}{&cnc.TxResponse{Code: 0, Height: int64(143)}, nil},
			sumbitPFDRun:              func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			isSubmitBatchAsync:        false,
			expectedSubmitPFDMinCalls: 1,
			expectedInclusionHeight:   143,
		},
		{
			name:                      "TestSubmitPFDResponseCodeFailure",
			submitPFDReturn:           []interface{}{&cnc.TxResponse{Code: 1}, nil},
			sumbitPFDRun:              func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			isSubmitBatchAsync:        true,
			expectedSubmitPFDMinCalls: 2,
		},
		{
			name:                      "TestSubmitPFDResponseTxHashAndError",
			submitPFDReturn:           []interface{}{&cnc.TxResponse{TxHash: "1234"}, errors.New("timeout")},
			sumbitPFDRun:              func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			TxFnReturn:                []interface{}{&coretypes.ResultTx{Hash: bytes.HexBytes("1234"), Height: int64(145)}, nil},
			isSubmitBatchAsync:        false,
			expectedSubmitPFDMinCalls: 1,
			expectedInclusionHeight:   145,
		},
		{
			name:                      "TestSubmitPFDResponseTxHashAndErrorAndTxNotFound",
			submitPFDReturn:           []interface{}{&cnc.TxResponse{TxHash: "1234"}, errors.New("timeout")},
			sumbitPFDRun:              func(args mock.Arguments) { time.Sleep(10 * time.Millisecond) },
			TxFnReturn:                []interface{}{nil, errors.New("notFound")},
			isSubmitBatchAsync:        true,
			expectedSubmitPFDMinCalls: 2,
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
			celestia.WithCNCClient(mockCNCClient),
			celestia.WithRPCClient(rpcmockClient),
		}
		// Start the DALC
		dalc := celestia.DataAvailabilityLayerClient{}
		err = dalc.Init(configBytes, pubsub.NewServer(), nil, test.NewLogger(t), options...)
		require.NoError(err)
		err = dalc.Start()
		require.NoError(err)
		// Set the mock functions
		mockCNCClient.On(submitPFDFuncName, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.submitPFDReturn...).Run(tc.sumbitPFDRun)
		rpcmockClient.On(TxFuncName, mock.Anything, mock.Anything, mock.Anything).Return(tc.TxFnReturn...)
		if tc.isSubmitBatchAsync {
			go dalc.SubmitBatch(batch)
			time.Sleep(50 * time.Millisecond)
			// Iterate over the calls to SubmitPFD and sum the number of times it was called
		} else {
			res := dalc.SubmitBatch(batch)
			assert.Equal(res.DAHeight, uint64(tc.expectedInclusionHeight))
		}
		err = dalc.Stop()
		require.NoError(err)
		// Wait for the goroutines to finish before accessing the mock calls
		time.Sleep(50 * time.Millisecond)
		t.Log("Verifying mock calls")
		assert.GreaterOrEqual(testutil.CountMockCalls(mockCNCClient.Calls, submitPFDFuncName), tc.expectedSubmitPFDMinCalls)
	}
}
