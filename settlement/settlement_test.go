package settlement_test

import (
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/gerr"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/dymensionxyz/dymint/da"
	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	tsmock "github.com/stretchr/testify/mock"
)

const batchSize = 5

func TestLifecycle(t *testing.T) {
	var err error
	client := registry.GetClient(registry.Local)
	require := require.New(t)

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(err)
	err = client.Init(settlement.Config{}, pubsubServer, log.TestingLogger())
	require.NoError(err)

	err = client.Start()
	require.NoError(err)

	err = client.Stop()
	require.NoError(err)
}

func TestSubmitAndRetrieve(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	settlementClient := registry.GetClient(registry.Local)

	initClient(t, settlementClient)

	// Get settlement lastest batch and check if there is an error as we haven't written anything yet.
	_, err := settlementClient.RetrieveBatch()
	assert.ErrorIs(err, gerr.ErrNotFound)

	// Get nonexisting stateIndex from the settlement layer
	_, err = settlementClient.RetrieveBatch(uint64(100))
	assert.ErrorIs(err, gerr.ErrNotFound)

	// Create and submit multiple batches
	numBatches := 4
	var batch *types.Batch
	// iterate batches
	for i := 0; i < numBatches; i++ {
		startHeight := uint64(i)*batchSize + 1
		// Create the batch
		propserKey, _, err := crypto.GenerateEd25519Key(nil)
		require.NoError(err)
		batch, err = testutil.GenerateBatch(startHeight, uint64(startHeight+batchSize-1), propserKey)
		require.NoError(err)
		// Submit the batch
		daResult := &da.ResultSubmitBatch{
			BaseResult: da.BaseResult{},
			SubmitMetaData: &da.DASubmitMetaData{
				Height: batch.EndHeight,
			},
		}
		err = settlementClient.SubmitBatch(batch, da.Mock, daResult)
		require.NoError(err)
		// sleep for 500 ms to make sure batch got accepted by the settlement layer
		time.Sleep(500 * time.Millisecond)
	}

	// Retrieve the latest batch and make sure it matches latest batch submitted
	lastestBatch, err := settlementClient.RetrieveBatch()
	require.NoError(err)
	assert.Equal(batch.EndHeight, lastestBatch.EndHeight)

	// Retrieve one batch before last
	batchResult, err := settlementClient.RetrieveBatch(lastestBatch.StateIndex - 1)
	require.NoError(err)
	middleOfBatchHeight := uint64(numBatches-1)*(batchSize) - (batchSize / 2)
	assert.LessOrEqual(batchResult.StartHeight, middleOfBatchHeight)
	assert.GreaterOrEqual(batchResult.EndHeight, middleOfBatchHeight)
}

func TestGetSequencersEmptyList(t *testing.T) {
	var err error
	settlementClient := registry.GetClient(registry.Local)
	hubClientMock := mocks.NewMockHubClient(t)
	hubClientMock.On("GetSequencers", tsmock.Anything, tsmock.Anything).Return(nil, gerr.ErrNotFound)
	options := []settlement.Option{
		settlement.WithHubClient(hubClientMock),
	}

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(t, err)
	err = settlementClient.Init(settlement.Config{}, pubsubServer, log.TestingLogger(), options...)
	assert.Error(t, err, "empty sequencer list should return an error")
}

func TestGetSequencers(t *testing.T) {
	hubClientMock := mocks.NewMockHubClient(t)
	hubClientMock.On("Start", tsmock.Anything).Return(nil)
	hubClientMock.On("Stop", tsmock.Anything).Return(nil)
	// Mock a sequencer response by the sequencerByRollapp query
	totalSequencers := 5
	sequencers, proposer := generateSequencers(totalSequencers)
	hubClientMock.On("GetSequencers", tsmock.Anything, tsmock.Anything).Return(sequencers, nil)
	options := []settlement.Option{
		settlement.WithHubClient(hubClientMock),
	}
	settlementClient := registry.GetClient(registry.Local)
	initClient(t, settlementClient, options...)

	sequencersList := settlementClient.GetSequencersList()

	assert.Len(t, sequencersList, len(sequencers))
	assert.Equal(t, settlementClient.GetProposer().PublicKey, proposer.PublicKey)

	err := settlementClient.Stop()
	// Wait until the settlement layer stops
	<-time.After(1 * time.Second)
	assert.NoError(t, err)

	// Validate  the amount of inactive sequencers
	var inactiveSequencerAmount int
	for _, sequencer := range sequencersList {
		if sequencer.Status == types.Inactive {
			inactiveSequencerAmount += 1
		}
	}
	assert.Equal(t, inactiveSequencerAmount, totalSequencers-1)
}

/* -------------------------------------------------------------------------- */
/*                                    Utils                                   */
/* -------------------------------------------------------------------------- */

func initClient(t *testing.T, settlementlc settlement.LayerI, options ...settlement.Option) {
	require := require.New(t)
	var err error

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(err)
	err = settlementlc.Init(settlement.Config{}, pubsubServer, log.TestingLogger(), options...)
	require.NoError(err)

	err = settlementlc.Start()
	require.NoError(err)
}

func generateSequencers(count int) ([]*types.Sequencer, *types.Sequencer) {
	sequencers := make([]*types.Sequencer, count)
	proposer := &types.Sequencer{
		PublicKey: ed25519.GenPrivKey().PubKey(),
		Status:    types.Proposer,
	}
	sequencers[0] = proposer
	for i := 1; i < count; i++ {
		sequencers[i] = &types.Sequencer{
			PublicKey: ed25519.GenPrivKey().PubKey(),
			Status:    types.Inactive,
		}
	}
	return sequencers, proposer
}
