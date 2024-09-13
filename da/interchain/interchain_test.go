package interchain_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/dymensionxyz/dymint/da/interchain"
	"github.com/dymensionxyz/dymint/types"
)

// TODO: add interchain DA chain mock
func TestDALayerClient(t *testing.T) {
	t.Skip() // Test is not finished yet

	client := new(interchain.DALayerClient)
	config := interchain.DefaultDAConfig()
	rawConfig, err := json.Marshal(config)
	require.NoError(t, err)
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	err = client.Init(rawConfig, nil, nil, logger)
	require.NoError(t, err)

	batch := types.Batch{
		StartHeight: 1,
		EndHeight:   3,
		Blocks:      []*types.Block{{Header: types.Header{Height: 1}}},
		Commits:     []*types.Commit{{Height: 1}},
	}

	submitResult := client.SubmitBatchV2(batch)
	require.NoError(t, submitResult.Error)

	retrieveResult := client.RetrieveBatchesV2(submitResult)
	require.NoError(t, retrieveResult.Error)

	require.Equal(t, batch, retrieveResult.Batch)
}
