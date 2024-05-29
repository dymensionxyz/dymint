package dymension_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	rollapptypes "github.com/dymensionxyz/dymension/v3/x/rollapp/types"
	sequencertypes "github.com/dymensionxyz/dymension/v3/x/sequencer/types"
	"github.com/dymensionxyz/dymint/da"
	rollapptypesmock "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymension/v3/x/rollapp/types"
	sequencertypesmock "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymension/v3/x/sequencer/types"
	dymensionmock "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/settlement/dymension"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/dymension"
	"github.com/dymensionxyz/dymint/testutil"

	sdkcodectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func TestGetSequencers(t *testing.T) {
	var err error
	require := require.New(t)
	cosmosClientMock := dymensionmock.NewMockCosmosClient(t)

	sequencerQueryClientMock := sequencertypesmock.NewMockQueryClient(t)
	count := 5
	sequencersRollappResponse, _ := generateSequencerByRollappResponse(t, count)
	sequencerQueryClientMock.On("SequencersByRollappByStatus", mock.Anything, mock.Anything).Return(sequencersRollappResponse, nil)

	cosmosClientMock.On("GetRollappClient").Return(rollapptypesmock.NewMockQueryClient(t))
	cosmosClientMock.On("GetSequencerClient").Return(sequencerQueryClientMock)

	options := []settlement.Option{
		dymension.WithCosmosClient(cosmosClientMock),
	}

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(err)

	hubClient := dymension.Client{}
	err = hubClient.Init(settlement.Config{}, pubsubServer, log.TestingLogger(), options...)
	require.NoError(err)

	sequencers, err := hubClient.GetSequencers()
	require.NoError(err)
	require.Len(sequencers, count)
}

func TestPostBatchRPCError(t *testing.T) {
	require := require.New(t)
	pubsubServer := pubsub.NewServer()
	err := pubsubServer.Start()
	require.NoError(err)

	// Create a mock cosmos client
	cosmosClientMock := dymensionmock.NewMockCosmosClient(t)
	sequencerQueryClientMock := sequencertypesmock.NewMockQueryClient(t)
	rollappQueryClientMock := rollapptypesmock.NewMockQueryClient(t)
	cosmosClientMock.On("GetRollappClient").Return(rollappQueryClientMock)
	cosmosClientMock.On("GetSequencerClient").Return(sequencerQueryClientMock)
	accountPubkey, err := sdkcodectypes.NewAnyWithValue(secp256k1.GenPrivKey().PubKey())
	require.NoError(err)
	cosmosClientMock.On("GetAccount", mock.Anything).Return(cosmosaccount.Account{Record: &keyring.Record{PubKey: accountPubkey}}, nil)

	options := []settlement.Option{
		dymension.WithCosmosClient(cosmosClientMock),
		dymension.WithBatchAcceptanceTimeout(time.Millisecond * 300),
		dymension.WithBatchAcceptanceAttempts(2),
	}

	// Create a batch which will be submitted
	propserKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(err)
	batch, err := testutil.GenerateBatch(2, 2, propserKey)
	require.NoError(err)

	hubClient := dymension.Client{}
	err = hubClient.Init(settlement.Config{}, pubsubServer, log.TestingLogger(), options...)
	require.NoError(err)

	// prepare mocks
	// submit passes
	// polling return nothing
	// submit returns already exists error
	cosmosClientMock.On("BroadcastTx", mock.Anything, mock.Anything).Return(cosmosclient.Response{TxResponse: &types.TxResponse{Code: 0}}, nil).Once()
	submitBatchError := errors.New("rpc error: code = Unknown desc = rpc error: code = Unknown desc = failed to execute message; message index: 0: expected height (5), but received (4): start-height does not match rollapps state")
	cosmosClientMock.On("BroadcastTx", mock.Anything, mock.Anything).Return(cosmosclient.Response{TxResponse: &types.TxResponse{Code: 1}}, submitBatchError).Once()

	daMetaData := &da.DASubmitMetaData{
		Height: 1,
		Client: da.Mock,
	}
	rollappQueryClientMock.On("StateInfo", mock.Anything, mock.Anything).Return(
		&rollapptypes.QueryGetStateInfoResponse{StateInfo: rollapptypes.StateInfo{
			StartHeight: 1, StateInfoIndex: rollapptypes.StateInfoIndex{Index: 1}, DAPath: daMetaData.ToPath(), NumBlocks: 1,
		}},
		nil).Times(2)

	daMetaData.Height = 2
	rollappQueryClientMock.On("StateInfo", mock.Anything, mock.Anything).Return(
		&rollapptypes.QueryGetStateInfoResponse{StateInfo: rollapptypes.StateInfo{
			StartHeight: 2, StateInfoIndex: rollapptypes.StateInfoIndex{Index: 2}, DAPath: daMetaData.ToPath(), NumBlocks: 1,
		}},
		nil).Once()

	resultSubmitBatch := &da.ResultSubmitBatch{}
	resultSubmitBatch.SubmitMetaData = &da.DASubmitMetaData{}
	errChan := make(chan error, 1) // Create a channel to receive an error from the goroutine
	// Post the batch in a goroutine and capture any error.
	go func() {
		err := hubClient.SubmitBatch(batch, da.Mock, resultSubmitBatch)
		errChan <- err // Send any error to the errChan
	}()

	// Use a select statement to wait for a potential error or a timeout.
	select {
	case err = <-errChan:
	case <-time.After(3 * time.Second):
		err = errors.New("timeout")
	}
	assert.NoError(t, err, "PostBatch should not produce an error")
}

// TestPostBatch should test the following:
// 1. Batch fails to submit emits unhealthy event
// 2. Batch is submitted successfully but not accepted which emits unhealthy event
// 3. Batch is submitted successfully, hub event not emitted, but checking for inclusion succeeds
// 4. Batch is submitted successfully and accepted by catching hub event
func TestPostBatch(t *testing.T) {
	var err error
	require := require.New(t)

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(err)

	// Create a mock cosmos client
	cosmosClientMock := dymensionmock.NewMockCosmosClient(t)
	sequencerQueryClientMock := sequencertypesmock.NewMockQueryClient(t)
	rollappQueryClientMock := rollapptypesmock.NewMockQueryClient(t)
	cosmosClientMock.On("GetRollappClient").Return(rollappQueryClientMock)
	cosmosClientMock.On("GetSequencerClient").Return(sequencerQueryClientMock)
	submitBatchError := errors.New("failed to submit batch")
	accountPubkey, err := sdkcodectypes.NewAnyWithValue(secp256k1.GenPrivKey().PubKey())
	require.NoError(err)
	cosmosClientMock.On("GetAccount", mock.Anything).Return(cosmosaccount.Account{Record: &keyring.Record{PubKey: accountPubkey}}, nil)
	cosmosClientMock.On("StopEventListener").Return(nil)
	cosmosClientMock.On("StartEventListener").Return(nil)
	cosmosClientMock.On("EventListenerQuit").Return(make(<-chan struct{}))
	batchAcceptedCh := make(chan coretypes.ResultEvent, 1)
	cosmosClientMock.On("SubscribeToEvents", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((<-chan coretypes.ResultEvent)(batchAcceptedCh), nil)

	options := []settlement.Option{
		dymension.WithCosmosClient(cosmosClientMock),
		dymension.WithBatchAcceptanceTimeout(time.Millisecond * 300),
		dymension.WithRetryAttempts(2),
		dymension.WithRetryMinDelay(time.Millisecond * 200),
		dymension.WithRetryMaxDelay(time.Second * 2),
	}

	// Create a batch which will be submitted
	propserKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(err)
	batch, err := testutil.GenerateBatch(1, 1, propserKey)
	require.NoError(err)

	cases := []struct {
		name                    string
		isBatchSubmitSuccess    bool
		isBatchAcceptedHubEvent bool
		shouldMockBatchIncluded bool
		isBatchIncludedSuccess  bool
	}{
		{
			name:                    "SubmitBatchFailure",
			isBatchSubmitSuccess:    false,
			isBatchAcceptedHubEvent: false,
			shouldMockBatchIncluded: true,
			isBatchIncludedSuccess:  false,
		},
		{
			name:                    "SubmitBatchSuccessNoBatchAcceptedHubEventNotIncluded",
			isBatchSubmitSuccess:    true,
			isBatchAcceptedHubEvent: false,
			shouldMockBatchIncluded: true,
			isBatchIncludedSuccess:  false,
		},
		{
			name:                    "SubmitBatchSuccessNotAcceptedYesIncluded",
			isBatchSubmitSuccess:    true,
			isBatchAcceptedHubEvent: false,
			shouldMockBatchIncluded: true,
			isBatchIncludedSuccess:  true,
		},
		{
			name:                    "SubmitBatchSuccessAndAccepted",
			isBatchSubmitSuccess:    true,
			isBatchAcceptedHubEvent: true,
			shouldMockBatchIncluded: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Reset the mock functions
			testutil.UnsetMockFn(cosmosClientMock.On("BroadcastTx"))
			testutil.UnsetMockFn(rollappQueryClientMock.On("StateInfo"))
			// Set the mock logic based on the test case
			if !c.isBatchSubmitSuccess {
				cosmosClientMock.On("BroadcastTx", mock.Anything, mock.Anything).Return(cosmosclient.Response{TxResponse: &types.TxResponse{Code: 1}}, submitBatchError)
			} else {
				cosmosClientMock.On("BroadcastTx", mock.Anything, mock.Anything).Return(cosmosclient.Response{TxResponse: &types.TxResponse{Code: 0}}, nil)
			}
			if c.shouldMockBatchIncluded {
				if c.isBatchIncludedSuccess {
					daMetaData := &da.DASubmitMetaData{
						Height: 1,
						Client: da.Mock,
					}
					rollappQueryClientMock.On("StateInfo", mock.Anything, mock.Anything).Return(
						&rollapptypes.QueryGetStateInfoResponse{StateInfo: rollapptypes.StateInfo{
							StartHeight: batch.StartHeight, StateInfoIndex: rollapptypes.StateInfoIndex{Index: 1}, DAPath: daMetaData.ToPath(), NumBlocks: 1,
						}},
						nil)
				} else {
					rollappQueryClientMock.On("StateInfo", mock.Anything, mock.Anything).Return(nil, status.New(codes.NotFound, "not found").Err())
				}
			}
			hubClient := dymension.Client{}
			err := hubClient.Init(settlement.Config{}, pubsubServer, log.TestingLogger(), options...)
			require.NoError(err)
			err = hubClient.Start()
			require.NoError(err)

			resultSubmitBatch := &da.ResultSubmitBatch{}
			resultSubmitBatch.SubmitMetaData = &da.DASubmitMetaData{}
			errChan := make(chan error, 1) // Create a channel to receive an error from the goroutine
			// Post the batch in a goroutine and capture any error.
			go func() {
				err := hubClient.SubmitBatch(batch, da.Mock, resultSubmitBatch)
				errChan <- err // Send any error to the errChan
			}()

			// Wait for the batch to be submitted and submit an event notifying that the batch was accepted
			if c.isBatchAcceptedHubEvent {
				time.Sleep(100 * time.Millisecond)
				batchAcceptedCh <- coretypes.ResultEvent{
					Query: fmt.Sprintf("state_update.rollapp_id='%s' AND state_update.status='PENDING'", ""),
					Events: map[string][]string{
						"state_update.num_blocks":       {"1"},
						"state_update.start_height":     {"1"},
						"state_update.state_info_index": {"1"},
					},
				}
			}
			// Use a select statement to wait for a potential error or a timeout.
			select {
			case err := <-errChan:
				// Check for error from PostBatch.
				assert.NoError(t, err, "PostBatch should not produce an error")
			case <-time.After(200 * time.Millisecond):
			}

			// Stop the hub client and wait for it to stop
			err = hubClient.Stop()
			require.NoError(err)
			time.Sleep(1 * time.Second)
		})
	}
}

/* -------------------------------------------------------------------------- */
/*                                    Utils                                   */
/* -------------------------------------------------------------------------- */

func generateSequencerByRollappResponse(t *testing.T, count int) (*sequencertypes.QueryGetSequencersByRollappByStatusResponse, sequencertypes.Sequencer) {
	// Generate the proposer sequencer
	proposerPubKeyAny, err := sdkcodectypes.NewAnyWithValue(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	proposer := sequencertypes.Sequencer{
		DymintPubKey: proposerPubKeyAny,
		Status:       sequencertypes.Bonded,
		Proposer:     true,
	}
	squencerInfoList := []sequencertypes.Sequencer{proposer}
	// Generate the inactive sequencers
	for i := 0; i < count-1; i++ {
		nonProposerPubKeyAny, err := sdkcodectypes.NewAnyWithValue(secp256k1.GenPrivKey().PubKey())
		require.NoError(t, err)

		nonProposer := sequencertypes.Sequencer{
			DymintPubKey: nonProposerPubKeyAny,
			Status:       sequencertypes.Bonded,
		}
		squencerInfoList = append(squencerInfoList, nonProposer)
	}
	response := &sequencertypes.QueryGetSequencersByRollappByStatusResponse{
		Sequencers: squencerInfoList,
	}
	return response, proposer
}
