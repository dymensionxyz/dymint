package dymension

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	sequencertypes "github.com/dymensionxyz/dymension/x/sequencer/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log/test"
	mocks "github.com/dymensionxyz/dymint/mocks"
	settlementmocks "github.com/dymensionxyz/dymint/mocks/settlement"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"

	sdkcodectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func TestGetSequencers(t *testing.T) {
	require := require.New(t)
	cosmosClientMock := mocks.NewCosmosClient(t)

	sequencerQueryClientMock := settlementmocks.NewSequencerQueryClient(t)
	count := 5
	sequencersRollappResponse, _ := generateSequencerByRollappResponse(t, count)
	sequencerQueryClientMock.On("SequencersByRollapp", mock.Anything, mock.Anything).Return(sequencersRollappResponse, nil)

	cosmosClientMock.On("GetRollappClient").Return(settlementmocks.NewRollAppQueryClient(t))
	cosmosClientMock.On("GetSequencerClient").Return(sequencerQueryClientMock)

	options := []Option{
		WithCosmosClient(cosmosClientMock),
	}

	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()

	hubClient, err := newDymensionHubClient(settlement.Config{}, pubsubServer, test.NewLogger(t), options...)
	require.NoError(err)

	sequencers, err := hubClient.GetSequencers("mock-rollapp")
	require.NoError(err)
	require.Len(sequencers, count)
}

// TestPostBatch should test the following:
// 1. Batch fails to submit emits unhealthy event
// 2. Batch is submitted successfully but not accepted which emits unhealthy event
// 3. Batch is submitted successfully and accepted which emits healthy event
func TestPostBatch(t *testing.T) {
	require := require.New(t)

	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()

	// Create a mock cosmos client
	cosmosClientMock := mocks.NewCosmosClient(t)
	sequencerQueryClientMock := settlementmocks.NewSequencerQueryClient(t)
	cosmosClientMock.On("GetRollappClient").Return(settlementmocks.NewRollAppQueryClient(t))
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

	options := []Option{
		WithCosmosClient(cosmosClientMock),
		WithBatchAcceptanceTimeout(time.Millisecond * 300),
		WithBatchRetryAttempts(2),
		WithBatchRetryDelay(time.Millisecond * 200),
	}

	// Create a batch which will be submitted
	propserKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(err)
	batch, err := testutil.GenerateBatch(0, 1, propserKey)
	require.NoError(err)

	// Subscribe to the health status event
	HealthSubscription, err := pubsubServer.Subscribe(context.Background(), "testPostBatch", settlement.EventQuerySettlementHealthStatus)
	assert.NoError(t, err)

	cases := []struct {
		name                 string
		isBatchSubmitSuccess bool
		isBatchAccepted      bool
		expectedHealthStatus bool
		expectedError        error
	}{
		{
			name:                 "TestSubmitBatchFailure",
			isBatchSubmitSuccess: false,
			isBatchAccepted:      false,
			expectedHealthStatus: false,
			expectedError:        submitBatchError,
		},
		{
			name:                 "TestSubmitBatchSuccessNotAccepted",
			isBatchSubmitSuccess: true,
			isBatchAccepted:      false,
			expectedHealthStatus: false,
			expectedError:        settlement.ErrBatchNotAccepted,
		},
		{
			name:                 "TestSubmitBatchSuccessAndAccepted",
			isBatchSubmitSuccess: true,
			isBatchAccepted:      true,
			expectedHealthStatus: true,
			expectedError:        nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			done := make(chan bool)
			testutil.UnsetMockFn(cosmosClientMock.On("BroadcastTx"))
			if !c.isBatchSubmitSuccess {
				cosmosClientMock.On("BroadcastTx", mock.Anything, mock.Anything).Return(cosmosclient.Response{TxResponse: &types.TxResponse{Code: 1}}, submitBatchError)
			} else {
				cosmosClientMock.On("BroadcastTx", mock.Anything, mock.Anything).Return(cosmosclient.Response{TxResponse: &types.TxResponse{Code: 0}}, nil)
			}
			hubClient, err := newDymensionHubClient(settlement.Config{}, pubsubServer, test.NewLogger(t), options...)
			require.NoError(err)
			hubClient.Start()
			go hubClient.PostBatch(batch, da.Mock, &da.ResultSubmitBatch{})
			// Wait for the batch to be submitted and submit an event notifying that the batch was accepted
			time.Sleep(50 * time.Millisecond)
			if c.isBatchAccepted {
				batchAcceptedCh <- coretypes.ResultEvent{
					Query: fmt.Sprintf("state_update.rollapp_id='%s'", ""),
					Events: map[string][]string{
						"state_update.num_blocks":       {"1"},
						"state_update.start_height":     {"0"},
						"state_update.state_info_index": {"1"},
					},
				}
			}
			go func() {
				select {
				case event := <-HealthSubscription.Out():
					healthStatusEvent := event.Data().(settlement.EventDataSettlementHealthStatus)
					assert.Equal(t, c.expectedHealthStatus, healthStatusEvent.Healthy)
					assert.Equal(t, c.expectedError, healthStatusEvent.Error)
					done <- true
					break
				case <-time.After(1 * time.Second):
					t.Error("expected health status event but didn't get one")
					done <- true
					break
				}
			}()
			<-done
			// Stop the hub client and wait for it to stop
			hubClient.Stop()
			time.Sleep(1 * time.Second)
		})
	}
}

/* -------------------------------------------------------------------------- */
/*                                    Utils                                   */
/* -------------------------------------------------------------------------- */

func generateSequencerByRollappResponse(t *testing.T, count int) (*sequencertypes.QueryGetSequencersByRollappResponse, sequencertypes.SequencerInfo) {
	// Generate the proposer sequencer
	proposerPubKeyAny, err := sdkcodectypes.NewAnyWithValue(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	proposer := sequencertypes.SequencerInfo{
		Sequencer: sequencertypes.Sequencer{
			DymintPubKey: proposerPubKeyAny,
		},
		Status: sequencertypes.Proposer,
	}
	squencerInfoList := []sequencertypes.SequencerInfo{
		proposer,
	}
	// Generate the inactive sequencers
	for i := 0; i < count-1; i++ {
		nonProposerPubKeyAny, err := sdkcodectypes.NewAnyWithValue(secp256k1.GenPrivKey().PubKey())
		require.NoError(t, err)
		squencerInfoList = append(squencerInfoList, sequencertypes.SequencerInfo{
			Sequencer: sequencertypes.Sequencer{
				DymintPubKey: nonProposerPubKeyAny,
			},
			Status: sequencertypes.Inactive,
		})
	}
	response := &sequencertypes.QueryGetSequencersByRollappResponse{
		SequencerInfoList: squencerInfoList,
	}
	return response, proposer
}
