package settlement_test

import (
	"testing"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/dymensionxyz/dymint/da"
	mocks "github.com/dymensionxyz/dymint/mocks/settlement"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	tsmock "github.com/stretchr/testify/mock"
	ce "github.com/tendermint/tendermint/crypto/encoding"
	pc "github.com/tendermint/tendermint/proto/tendermint/crypto"
)

const batchSize = 5

func TestLifecycle(t *testing.T) {
	client := registry.GetClient(registry.Mock)
	require := require.New(t)

	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	err := client.Init(settlement.Config{}, pubsubServer, log.TestingLogger())
	require.NoError(err)

	err = client.Start()
	require.NoError(err)

	err = client.Stop()
	require.NoError(err)
}

func TestInvalidSubmit(t *testing.T) {
	assert := assert.New(t)
	settlementClient := registry.GetClient(registry.Mock)
	initClient(t, settlementClient)

	// Create cases
	cases := []struct {
		startHeight uint64
		endHeight   uint64
		shouldPanic bool
	}{
		{startHeight: 1, endHeight: batchSize, shouldPanic: false},
		// batch with endHight < startHeight
		{startHeight: batchSize + 2, endHeight: 1, shouldPanic: true},
		// batch with startHeight != previousEndHeight + 1
		{startHeight: batchSize, endHeight: 1 + batchSize + batchSize, shouldPanic: true},
	}
	for _, c := range cases {
		batch := &types.Batch{
			StartHeight: c.startHeight,
			EndHeight:   c.endHeight,
		}
		daResult := &da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				DAHeight: c.endHeight,
			},
		}
		if c.shouldPanic {
			assert.Panics(func() {
				settlementClient.SubmitBatch(batch, da.Mock, daResult)
			})
		} else {
			settlementClient.SubmitBatch(batch, da.Mock, daResult)
		}
	}

}

func TestSubmitAndRetrieve(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	settlementClient := registry.GetClient(registry.Mock)

	initClient(t, settlementClient)

	// Get settlement lastest batch and check if there is an error as we haven't written anything yet.
	_, err := settlementClient.RetrieveBatch()
	require.Error(err)
	assert.Equal(err, settlement.ErrBatchNotFound)

	// Get nonexisting stateIndex from the settlement layer
	_, err = settlementClient.RetrieveBatch(uint64(100))
	require.Error(err)
	assert.Equal(err, settlement.ErrBatchNotFound)

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
			BaseResult: da.BaseResult{
				DAHeight: batch.EndHeight,
			},
		}
		settlementClient.SubmitBatch(batch, da.Mock, daResult)
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
	settlementClient := registry.GetClient(registry.Mock)
	hubClientMock := mocks.NewHubClient(t)
	hubClientMock.On("GetLatestBatch", tsmock.Anything).Return(nil, settlement.ErrBatchNotFound)
	hubClientMock.On("GetSequencers", tsmock.Anything, tsmock.Anything).Return(nil, settlement.ErrNoSequencerForRollapp)
	options := []settlement.Option{
		settlement.WithHubClient(hubClientMock),
	}

	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	err := settlementClient.Init(settlement.Config{}, pubsubServer, log.TestingLogger(), options...)
	assert.Error(t, err, "empty sequencer list should return an error")

}

func TestGetSequencers(t *testing.T) {
	hubClientMock := mocks.NewHubClient(t)
	hubClientMock.On("Start", tsmock.Anything).Return(nil)
	hubClientMock.On("Stop", tsmock.Anything).Return(nil)
	hubClientMock.On("GetLatestBatch", tsmock.Anything).Return(nil, settlement.ErrBatchNotFound)
	// Mock a sequencer response by the sequencerByRollapp query
	totalSequencers := 5
	sequencers, proposer := generateSequencers(totalSequencers)
	hubClientMock.On("GetSequencers", tsmock.Anything, tsmock.Anything).Return(sequencers, nil)
	options := []settlement.Option{
		settlement.WithHubClient(hubClientMock),
	}
	settlementClient := registry.GetClient(registry.Mock)
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

	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	err := settlementlc.Init(settlement.Config{}, pubsubServer, log.TestingLogger(), options...)
	require.NoError(err)

	err = settlementlc.Start()
	require.NoError(err)
}

func generateProtoPubKey(t *testing.T) pc.PublicKey {
	pubKey := tmtypes.NewMockPV().PrivKey.PubKey()
	protoPubKey, err := ce.PubKeyToProto(pubKey)
	require.NoError(t, err)
	return protoPubKey
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
