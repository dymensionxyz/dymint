package dymension_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	mocks "github.com/dymensionxyz/dymint/mocks"
	settlementmocks "github.com/dymensionxyz/dymint/mocks/settlement"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/dymension"
	"github.com/dymensionxyz/dymint/testutil"

	sdkcodectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func TestGetSequencers(t *testing.T) {
	var err error
	require := require.New(t)
	cosmosClientMock := mocks.NewCosmosClient(t)

	sequencerQueryClientMock := settlementmocks.NewSequencerQueryClient(t)
	count := 5
	sequencersRollappResponse, _ := generateSequencerByRollappResponse(t, count)
	sequencerQueryClientMock.On("SequencersByRollappByStatus", mock.Anything, mock.Anything).Return(sequencersRollappResponse, nil)

	cosmosClientMock.On("GetRollappClient").Return(settlementmocks.NewRollAppQueryClient(t))
	cosmosClientMock.On("GetSequencerClient").Return(sequencerQueryClientMock)

	options := []dymension.Option{
		dymension.WithCosmosClient(cosmosClientMock),
	}

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(err)

	hubClient, err := dymension.NewDymensionHubClient(settlement.Config{}, pubsubServer, log.TestingLogger(), options...)
	require.NoError(err)

	sequencers, err := hubClient.GetSequencers("mock-rollapp")
	require.NoError(err)
	require.Len(sequencers, count)
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
	cosmosClientMock := mocks.NewCosmosClient(t)
	sequencerQueryClientMock := settlementmocks.NewSequencerQueryClient(t)
	rollappQueryClientMock := settlementmocks.NewRollAppQueryClient(t)
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
	require.NoError(err)
	cosmosClientMock.On("SubscribeToEvents", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((<-chan coretypes.ResultEvent)(batchAcceptedCh), nil)

	options := []dymension.Option{
		dymension.WithCosmosClient(cosmosClientMock),
		dymension.WithBatchAcceptanceTimeout(time.Millisecond * 300),
		dymension.WithBatchRetryAttempts(2),
		dymension.WithBatchRetryDelay(time.Millisecond * 200),
	}

	// Create a batch which will be submitted
	propserKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(err)
	batch, err := testutil.GenerateBatch(1, 1, propserKey)
	require.NoError(err)
	// Subscribe to the health status event
	HealthSubscription, err := pubsubServer.Subscribe(context.Background(), "testPostBatch", settlement.EventQuerySettlementHealthStatus)
	assert.NoError(t, err)

	cases := []struct {
		name                    string
		isBatchSubmitSuccess    bool
		isBatchAcceptedHubEvent bool
		shouldMockBatchIncluded bool
		isBatchIncludedSuccess  bool
		expectedError           error
	}{
		{
			name:                    "TestSubmitBatchFailure",
			isBatchSubmitSuccess:    false,
			isBatchAcceptedHubEvent: false,
			shouldMockBatchIncluded: true,
			isBatchIncludedSuccess:  false,
			expectedError:           submitBatchError,
		},
		{
			name:                    "TestSubmitBatchSuccessNoBatchAcceptedHubEventNotIncluded",
			isBatchSubmitSuccess:    true,
			isBatchAcceptedHubEvent: false,
			shouldMockBatchIncluded: true,
			isBatchIncludedSuccess:  false,
			expectedError:           settlement.ErrBatchNotAccepted,
		},
		{
			name:                    "TestSubmitBatchSuccessNotAcceptedYesIncluded",
			isBatchSubmitSuccess:    true,
			isBatchAcceptedHubEvent: false,
			shouldMockBatchIncluded: true,
			isBatchIncludedSuccess:  true,
			expectedError:           nil,
		},
		{
			name:                    "TestSubmitBatchSuccessAndAccepted",
			isBatchSubmitSuccess:    true,
			isBatchAcceptedHubEvent: true,
			shouldMockBatchIncluded: false,
			expectedError:           nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Init the wait group and set the number of expected events
			var wg sync.WaitGroup
			eventsCount := 1
			wg.Add(eventsCount)
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
					rollappQueryClientMock.On("StateInfo", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error"))
				}
			}
			hubClient, err := dymension.NewDymensionHubClient(settlement.Config{}, pubsubServer, log.TestingLogger(), options...)
			require.NoError(err)
			err = hubClient.Start()
			require.NoError(err)
			// Handle the various events that are emitted and timeout if we don't get them
			var eventsReceivedCount int64
			go func() {
				select {
				case healthEvent := <-HealthSubscription.Out():
					t.Logf("got health event: %v", healthEvent)
					healthStatusEvent := healthEvent.Data().(*settlement.EventDataHealth)
					assert.ErrorIs(t, healthStatusEvent.Error, c.expectedError)
					atomic.AddInt64(&eventsReceivedCount, 1)
				case <-time.After(10 * time.Second):
					t.Error("Didn't receive health event")
				}
				wg.Done()
			}()

			resultSubmitBatch := &da.ResultSubmitBatch{}
			resultSubmitBatch.SubmitMetaData = &da.DASubmitMetaData{}
			errChan := make(chan error, 1) // Create a channel to receive an error from the goroutine
			// Post the batch in a goroutine and capture any error.
			go func() {
				err := hubClient.PostBatch(batch, da.Mock, resultSubmitBatch)
				errChan <- err // Send any error to the errChan
			}()

			// Use a select statement to wait for a potential error or a timeout.
			select {
			case err := <-errChan:
				// Check for error from PostBatch.
				assert.NoError(t, err, "PostBatch should not produce an error")
			case <-time.After(50 * time.Millisecond):
				// Timeout case to avoid blocking forever if PostBatch doesn't return.
			}
			// Wait for the batch to be submitted and submit an event notifying that the batch was accepted
			time.Sleep(50 * time.Millisecond)
			if c.isBatchAcceptedHubEvent {
				batchAcceptedCh <- coretypes.ResultEvent{
					Query: fmt.Sprintf("state_update.rollapp_id='%s'", ""),
					Events: map[string][]string{
						"state_update.num_blocks":       {"1"},
						"state_update.start_height":     {"1"},
						"state_update.state_info_index": {"1"},
					},
				}
			}
			wg.Wait()
			assert.Equal(t, eventsCount, int(eventsReceivedCount))
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
