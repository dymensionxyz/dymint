package dymension_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	sdkcodectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/local"
	dymensionmock "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/settlement/dymension"
	rollapptypesmock "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	sequencertypesmock "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/sequencer"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/dymension"
	"github.com/dymensionxyz/dymint/testutil"
	rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	sequencertypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/sequencer"
	protoutils "github.com/dymensionxyz/dymint/utils/proto"
)

func TestGetSequencers(t *testing.T) {
	var err error
	require := require.New(t)
	cosmosClientMock := dymensionmock.NewMockCosmosClient(t)

	sequencerQueryClientMock := sequencertypesmock.NewMockQueryClient(t)
	count := 5
	sequencersRollappResponse := generateSequencerByRollappResponse(t, count)
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
	err = hubClient.Init(settlement.Config{}, "rollappTest", pubsubServer, log.TestingLogger(), options...)
	require.NoError(err)

	sequencers, err := hubClient.GetBondedSequencers()
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
	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(err)
	batch, err := testutil.GenerateBatch(1, 1, proposerKey, [32]byte{})
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
			// Set the mock logic based on the test case
			if !c.isBatchSubmitSuccess {
				cosmosClientMock.On("BroadcastTx", mock.Anything, mock.Anything).Return(cosmosclient.Response{TxResponse: &types.TxResponse{Code: 1}}, submitBatchError)
			} else {
				cosmosClientMock.On("BroadcastTx", mock.Anything, mock.Anything).Return(cosmosclient.Response{TxResponse: &types.TxResponse{Code: 0}}, nil)
			}
			if c.shouldMockBatchIncluded {
				if c.isBatchIncludedSuccess {
					daMockMetadata := local.SubmitMetaData{
						Height: 1,
					}
					daMetaData := &da.DASubmitMetaData{
						Client: da.Mock,
						DAPath: daMockMetadata.ToPath(),
					}
					rollappQueryClientMock.On("StateInfo", mock.Anything, mock.Anything).Return(
						&rollapptypes.QueryGetStateInfoResponse{StateInfo: rollapptypes.StateInfo{
							StartHeight: batch.StartHeight(), StateInfoIndex: rollapptypes.StateInfoIndex{Index: 1}, DAPath: daMetaData.ToPath(), NumBlocks: 1,
						}},
						nil)
				} else {
					rollappQueryClientMock.On("StateInfo", mock.Anything, mock.Anything).Return(nil, status.New(codes.NotFound, "not found").Err())
				}
			}
			hubClient := dymension.Client{}
			err := hubClient.Init(settlement.Config{}, "rollappTest", pubsubServer, log.TestingLogger(), options...)
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

func generateSequencerByRollappResponse(t *testing.T, count int) *sequencertypes.QueryGetSequencersByRollappByStatusResponse {
	sequencerInfoList := []sequencertypes.Sequencer{}
	for i := 0; i < count; i++ {
		pk, err := sdkcodectypes.NewAnyWithValue(secp256k1.GenPrivKey().PubKey())
		require.NoError(t, err)

		seq := sequencertypes.Sequencer{
			DymintPubKey: protoutils.CosmosToGogo(pk),
			Status:       sequencertypes.Bonded,
		}
		sequencerInfoList = append(sequencerInfoList, seq)
	}
	response := &sequencertypes.QueryGetSequencersByRollappByStatusResponse{
		Sequencers: sequencerInfoList,
	}
	return response
}
