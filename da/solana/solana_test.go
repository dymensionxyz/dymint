package solana_test

import (
	"encoding/json"
	"math/big"
	"os"
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

func TestDataAvailabilityLayerClient(t *testing.T) {
	//t.Skip("Skipping Solana client tests")

	// Set up test environment
	mnemonicEnv := "SOLANA_KEYPATH"
	err := os.Setenv(mnemonicEnv, "/Users/sergi/wallet-keypair.json")
	require.NoError(t, err)

	// Create test config. By default, tests use Solana devnet.
	config := solana.TestConfig
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
	assert.Greater(t, balance.Amount.BigInt().Cmp(big.NewInt(5000)), 0, "at least balance of 5000 lamport required to send a tx")

	// Proposer key for generating test batches
	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	testCases := []struct {
		name  string
		batch *types.Batch
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
