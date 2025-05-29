package eth_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/eth"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/test-go/testify/require"
)

func TestDataAvailabilityLayerClient(t *testing.T) {
	t.Skip("Skipping Eth client tests")

	// Set up test environment
	priKeyEnv := "ETH_PRIVATE_KEY"
	err := os.Setenv(priKeyEnv, "f3459c9fb5b720f52968f97e0dd895fa1caf3fe4a521fbc6395380bc50b0a234")
	require.NoError(t, err)

	// Create test config. By default, tests use BNB devnet.
	config := eth.TestConfig

	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new Ethereum client
	client := &eth.DataAvailabilityLayerClient{}
	err = client.Init(configBytes, nil, nil, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)

	// Start DA Eth client
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
			name:    "huge batch: 362KB",
			batch:   testutil.GenerateBatchWithBlocks(600, proposerKey),
			wantErr: true,
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

func TestDataAvailabilityLayerClientRetrieval(t *testing.T) {
	t.Skip("Skipping Eth client tests")

	// Set up test environment
	priKeyEnv := "ETH_PRIVATE_KEY"
	err := os.Setenv(priKeyEnv, "f3459c9fb5b720f52968f97e0dd895fa1caf3fe4a521fbc6395380bc50b0a234")
	require.NoError(t, err)

	// Create test config. By default, tests use Sepo devnet.
	config := eth.TestConfig
	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new Ethereum client
	client := &eth.DataAvailabilityLayerClient{}
	err = client.Init(configBytes, nil, nil, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)

	err = client.Start()
	require.NoError(t, err)
	defer client.Stop()

	meta := &eth.SubmitMetaData{
		Commitment: []byte("0x9779eecd4626de77fda70a3cc30e2417e973eee5a9d26c9788a3a0f83f53cfe7c51f84bc71ffb8d9872a3e0093e4ab25"),
		Proof:      []byte("0xb1cec0984b065b25ee955b60927712bbc9915026ee072f198991203499458f3a6b7f3f0562fb95149e1a4647eec31cb6"),
		Slot:       "7723828",
	}

	result := client.CheckBatchAvailability(meta.ToPath())
	require.NoError(t, result.Error)

	rresult := client.RetrieveBatches(meta.ToPath())
	require.NoError(t, rresult.Error)
	require.Equal(t, da.StatusSuccess, rresult.Code)

}
