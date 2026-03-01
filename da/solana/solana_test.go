package solana_test

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/solana"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

// Test private key (devnet only - DO NOT USE IN PRODUCTION)
// This is a base58-encoded Solana keypair (64 bytes: 32-byte private key + 32-byte public key)
// Corresponds to address: 7NXRH5ciAnAPk2tAZvokeApjZZkqaPMdU8riZaRLZxFF
const testPrivateKey = "5N3YikeamLMj6FGqCSBBskbDooYUcPeKYeoQxb7XMJ1LREUfPQAKqHUtzB8HDKBnsw5UPJygcDZTFHDxs9pN1UJH"

func TestDataAvailabilityLayerClient(t *testing.T) {
	t.Skip("Skipping Solana client tests - requires live network connection")

	// Create temporary key file with test private key in JSON format
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "solana_key.json")
	keyJSON := `{"private_key": "` + testPrivateKey + `"}`
	err := os.WriteFile(keyFile, []byte(keyJSON), 0600)
	require.NoError(t, err)

	// Create test config with the temp key file path
	config := solana.TestConfig
	config.KeyPath = keyFile
	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new Solana client
	client := &solana.DataAvailabilityLayerClient{}
	err = client.Init(configBytes, nil, nil, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)

	err = client.Start()
	require.NoError(t, err)
	defer client.Stop()

	// Ensure that the client has enough SOL tokens to submit batches
	balance, err := client.GetSignerBalance()
	require.NoError(t, err)
	assert.Greater(t, balance.Amount.BigInt().Cmp(big.NewInt(5000000)), 0, "at least enough balance required to send txs (5000 lamport per tx)")

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
			name:  "mid-size batch: 20KB",
			batch: testutil.GenerateBatchWithBlocks(40, proposerKey),
		},
		{
			name:  "big batch: 88KB",
			batch: testutil.GenerateBatchWithBlocks(150, proposerKey),
		},
		{
			name:  "huge batch: 362KB",
			batch: testutil.GenerateBatchWithBlocks(600, proposerKey),
		},
		{
			name:    "not available batch",
			batch:   testutil.GenerateBatchWithBlocks(1, proposerKey),
			daError: da.ErrBlobNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.SubmitBatch(tc.batch)
			require.NoError(t, result.Error)
			require.Equal(t, da.StatusSuccess, result.Code)

			if tc.daError != nil {
				result.SubmitMetaData.DAPath = "3ZuGNM1NMRA4JLYMq3eLtBXtcWVjm19fGXhptDUHHie8gBPWXXk5ZSXtN6aP8A4NH84gvz8CVuazTxqkFnVpRi4f|3297d423b9d44ce38a287acf3b43239f0234517811fd60c1e884cb3fba780a5f"
			}

			// Retrieve batch
			retrieveResult := client.RetrieveBatches(result.SubmitMetaData.DAPath)
			if tc.daError != nil {
				assert.ErrorIs(t, retrieveResult.Error, tc.daError)
			} else {
				require.NoError(t, retrieveResult.Error)
				require.Equal(t, da.StatusSuccess, retrieveResult.Code)
				require.Len(t, retrieveResult.Batches, 1)
			}

			// Check batch availability
			checkResult := client.CheckBatchAvailability(result.SubmitMetaData.DAPath)
			if tc.daError != nil {
				assert.ErrorIs(t, checkResult.Error, tc.daError)
				return
			}
			require.NoError(t, checkResult.Error)
			require.Equal(t, da.StatusSuccess, checkResult.Code)

			// Compare submitted and retrieved batches
			submittedData, err := tc.batch.MarshalBinary()
			require.NoError(t, err)

			retrievedData, err := retrieveResult.Batches[0].MarshalBinary()
			require.NoError(t, err)

			require.Equal(t, submittedData, retrievedData, "submitted and retrieved batches should be identical")
		})
	}
}
