package walrus_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/walrus"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

func TestDataAvailabilityLayerClient(t *testing.T) {
	// t.Skip("Skipping Walrus client tests")

	// Create test config
	config := walrus.TestConfig
	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new Walrus client
	client := &walrus.DataAvailabilityLayerClient{}
	err = client.Init(configBytes, nil, nil, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)

	err = client.Start()
	require.NoError(t, err)
	defer client.Stop()

	// Proposer key for generating test batches
	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		batch   *types.Batch
		wantErr bool
	}{
		{
			name:  "small batch: 1KB",
			batch: testutil.GenerateBatchWithBlocks(1, proposerKey),
		},
		{
			name:  "mid-size batch: 20KB",
			batch: testutil.GenerateBatchWithBlocks(40, proposerKey),
		},
		{
			name:  "big batch: 88KB",
			batch: testutil.GenerateBatchWithBlocks(150, proposerKey),
		},
		{
			name:  "extremely big batch: 8.64MB",
			batch: testutil.GenerateBatchWithBlocks(15000, proposerKey),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.SubmitBatch(tc.batch)
			if tc.wantErr {
				require.Error(t, result.Error)
				return
			}
			require.NoError(t, result.Error)
			require.Equal(t, da.StatusSuccess, result.Code)

			// Check batch availability
			checkResult := client.CheckBatchAvailability(result.SubmitMetaData.DAPath)
			require.NoError(t, checkResult.Error)
			require.Equal(t, da.StatusSuccess, checkResult.Code)

			// Retrieve batch
			retrieveResult := client.RetrieveBatches(result.SubmitMetaData.DAPath)
			require.NoError(t, retrieveResult.Error)
			require.Equal(t, da.StatusSuccess, retrieveResult.Code)
			require.Len(t, retrieveResult.Batches, 1)

			// Compare submitted and retrieved batches
			submittedData, err := tc.batch.MarshalBinary()
			require.NoError(t, err)

			retrievedData, err := retrieveResult.Batches[0].MarshalBinary()
			require.NoError(t, err)

			require.Equal(t, submittedData, retrievedData, "submitted and retrieved batches should be identical")
		})
	}
}
