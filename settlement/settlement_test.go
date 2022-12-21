package settlement_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log/test"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/mock"
	"github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
)

const batchSize = 5

func TesClientsLifeCycle(t *testing.T) {

	for _, settlement := range registry.RegisteredClients() {
		t.Run(string(settlement), func(t *testing.T) {
			doTestLifecycle(t, registry.GetClient(settlement))
		})
	}
}

func doTestLifecycle(t *testing.T, settlementClient settlement.LayerClient) {
	require := require.New(t)

	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	err := settlementClient.Init([]byte{}, pubsubServer, test.NewLogger(t))
	require.NoError(err)

	err = settlementClient.Start()
	require.NoError(err)

	err = settlementClient.Stop()
	require.NoError(err)
}

func TestSubmitAndRetrieve(t *testing.T) {
	for _, settlement := range registry.RegisteredClients() {
		t.Run(string(settlement), func(t *testing.T) {
			//TODO(omritoptix): Currently not testing dymension SL as part of this tests.
			if settlement == registry.ClientMock {
				doTestSubmitAndRetrieve(t, registry.GetClient(settlement))
				doTestInvalidSubmit(t, registry.GetClient(settlement))
			}
		})
	}
}

func getConfForClient(settlementlc settlement.LayerClient) []byte {
	var config interface{}
	conf := []byte{}
	if _, ok := settlementlc.(*mock.SettlementLayerClient); ok {
		config = mock.Config{
			AutoUpdateBatches: false,
			BatchSize:         batchSize,
		}
	}
	conf, _ = json.Marshal(config)

	return conf
}

func initClient(t *testing.T, settlementlc settlement.LayerClient) {
	require := require.New(t)
	conf := getConfForClient(settlementlc)

	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()
	err := settlementlc.Init(conf, pubsubServer, test.NewLogger(t))
	require.NoError(err)

	err = settlementlc.Start()
	require.NoError(err)
}

func doTestInvalidSubmit(t *testing.T, settlementlc settlement.LayerClient) {
	assert := assert.New(t)
	initClient(t, settlementlc)

	// Create cases
	cases := []struct {
		startHeight uint64
		endHeight   uint64
		status      settlement.StatusCode
	}{
		{startHeight: 1, endHeight: batchSize, status: settlement.StatusSuccess},
		// batch with endHight < startHeight
		{startHeight: batchSize + 2, endHeight: 1, status: settlement.StatusError},
		// batch with startHeight != previousEndHeight + 1
		{startHeight: batchSize, endHeight: 1 + batchSize + batchSize, status: settlement.StatusError},
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
		resultSubmitBatch := settlementlc.SubmitBatch(batch, da.Mock, daResult)
		assert.Equal(resultSubmitBatch.Code, c.status)
	}

}

func doTestSubmitAndRetrieve(t *testing.T, settlementlc settlement.LayerClient) {
	require := require.New(t)
	assert := assert.New(t)

	initClient(t, settlementlc)

	// Get settlement lastest batch and check there is an error as we haven't written anything yet.
	_, err := settlementlc.RetrieveBatch()
	require.Error(err)
	assert.Equal(err, settlement.ErrBatchNotFound)

	// Get nonexisting stateIndex from the settlement layer
	_, err = settlementlc.RetrieveBatch(uint64(100))
	require.Error(err)
	assert.Equal(err, settlement.ErrBatchNotFound)

	// Create and submit multiple batches
	numBatches := 4
	var batch *types.Batch
	// iterate batches
	for i := 0; i < numBatches; i++ {
		startHeight := uint64(i)*batchSize + 1
		// Create the batch
		batch = testutil.GenerateBatch(startHeight, uint64(startHeight+batchSize-1))
		// Submit the batch
		daResult := &da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				DAHeight: batch.EndHeight,
			},
		}
		resultSubmitBatch := settlementlc.SubmitBatch(batch, da.Mock, daResult)
		assert.Equal(resultSubmitBatch.Code, settlement.StatusSuccess)
	}

	// Retrieve the latest batch and make sure it matches latest batch submitted
	lastestBatch, err := settlementlc.RetrieveBatch()
	require.NoError(err)
	assert.Equal(batch.EndHeight, lastestBatch.EndHeight)

	// Retrieve one batch before last
	batchResult, err := settlementlc.RetrieveBatch(lastestBatch.StateIndex - 1)
	require.NoError(err)
	middleOfBatchHeight := uint64(numBatches-1)*(batchSize) - (batchSize / 2)
	assert.LessOrEqual(batchResult.StartHeight, middleOfBatchHeight)
	assert.GreaterOrEqual(batchResult.EndHeight, middleOfBatchHeight)

	result, err := settlementlc.GetLatestFinalizedStateHeight("")
	require.NoError(err)
	assert.Equal(result, int64(numBatches*batchSize))
}
