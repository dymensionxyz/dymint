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
func TestDALayerClient_Init(t *testing.T) {
	t.Skip() // Test is not finished yet

	client := new(interchain.DALayerClient)
	config := interchain.DefaultDAConfig()
	rawConfig, err := json.Marshal(config)
	require.NoError(t, err)
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	err = client.Init(rawConfig, nil, nil, logger)
	require.NoError(t, err)

	result := client.SubmitBatchV2(&types.Batch{
		StartHeight: 1,
		EndHeight:   3,
		Blocks:      []*types.Block{{Header: types.Header{Height: 1}}},
		Commits:     []*types.Commit{{Height: 1}},
	})
	require.NoError(t, result.Error)
	t.Logf("result: %#v", result)
}
