package aptos_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cosmossdk.io/math"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/aptos"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	// Minimum APT balance required for testing
	minAPTBalance = 1000000 // 0.001 APT
	// Test private key (testnet only - DO NOT USE IN PRODUCTION)
	testPrivateKey = "0x638802252197206baa5160bf2ac60e0b95491d2128a265e6ee51e0c1b0a59d9f"
)

func TestDataAvailabilityClient(t *testing.T) {
	t.Skip("Skipping Aptos client tests")

	// Create temporary key file with test private key
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "aptos_key.json")
	keyJSON := `{"private_key": "` + testPrivateKey + `"}`
	err := os.WriteFile(keyFile, []byte(keyJSON), 0600)
	require.NoError(t, err)

	// Create test config with the temp key file path
	config := aptos.TestConfig
	config.KeyPath = keyFile
	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new Aptos client
	client := &aptos.DataAvailabilityLayerClient{}
	err = client.Init(configBytes, nil, nil, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)

	err = client.Start()
	require.NoError(t, err)
	defer client.Stop()

	// Ensure that the client has enough APT tokens to submit batches
	balance, err := client.GetSignerBalance()
	require.NoError(t, err)
	if balance.Amount.LT(math.NewInt(minAPTBalance)) {
		t.Log("Insufficient APT balance for testing: ", balance)
		t.Fail()
	}

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
			name:  "mid-size batch 1: 20KB",
			batch: testutil.GenerateBatchWithBlocks(40, proposerKey),
		},
		{
			name:  "mid-size batch 2: 48KB",
			batch: testutil.GenerateBatchWithBlocks(80, proposerKey),
		},
		{
			name:  "mid-size batch 2: almost 64KB",
			batch: testutil.GenerateBatchWithBlocks(107, proposerKey),
		},
		// Error: EXCEEDED_MAX_TRANSACTION_SIZE. it does not error but time out
		/*{
			name:    "big-size batch 2: 72KB",
			batch:   testutil.GenerateBatchWithBlocks(120, proposerKey),
			wantErr: true,
		},*/
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
