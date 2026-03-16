package bnb_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/bnb"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataAvailabilityLayerClient(t *testing.T) {
	t.Skip("Skipping BNB client test")

	// Create temp file with test private key in JSON format
	keyFile, err := os.CreateTemp("", "bnb_test_key")
	require.NoError(t, err)
	defer os.Remove(keyFile.Name())
	_, err = keyFile.WriteString(`{"private_key": "459c954ea1366473e0edfa4cf55b8028fa6405e44959cfcaa1771890dc527275"}`)
	require.NoError(t, err)
	keyFile.Close()

	// Create test config
	config := bnb.TestConfig
	config.KeyConfig.KeyPath = keyFile.Name()

	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new BNB client
	client := &bnb.DataAvailabilityLayerClient{}
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
		daError error
	}{
		{
			name:  "small batch: 1KB",
			batch: testutil.GenerateBatchWithBlocks(1, proposerKey),
		},
		{
			name:  "mid-size batch: 75KB",
			batch: testutil.GenerateBatchWithBlocks(100, proposerKey),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.SubmitBatch(tc.batch)

			require.NoError(t, result.Error)
			require.Equal(t, da.StatusSuccess, result.Code)

			// Check batch availability
			checkResult := client.CheckBatchAvailability(result.SubmitMetaData.DAPath)
			require.NoError(t, checkResult.Error)
			require.Equal(t, da.StatusSuccess, checkResult.Code)

			// Retrieve batch
			retrieveResult := client.RetrieveBatches(result.SubmitMetaData.DAPath)
			if tc.daError != nil {
				assert.ErrorIs(t, retrieveResult.Error, da.ErrBlobNotFound)
			} else {
				require.NoError(t, retrieveResult.Error)
				require.Equal(t, da.StatusSuccess, retrieveResult.Code)
				require.Len(t, retrieveResult.Batches, 1)
			}

			// Compare submitted and retrieved batches
			submittedData, err := tc.batch.MarshalBinary()
			require.NoError(t, err)

			retrievedData, err := retrieveResult.Batches[0].MarshalBinary()
			require.NoError(t, err)

			require.Equal(t, submittedData, retrievedData, "submitted and retrieved batches should be identical")
		})
	}
}
