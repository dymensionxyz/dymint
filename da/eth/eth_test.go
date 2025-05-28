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
	//t.Skip("Skipping BNB client tests")

	// Create test config. By default, tests use BNB devnet.
	config := eth.EthConfig{
		Endpoint:   "https://ethereum-sepolia-rpc.publicnode.com",
		PrivateKey: "f3459c9fb5b720f52968f97e0dd895fa1caf3fe4a521fbc6395380bc50b0a234",
		Timeout:    500000000000,
		ChainId:    11155111,
	}
	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new Ethereum client
	client := &eth.DataAvailabilityLayerClient{}
	err = client.Init(configBytes, nil, nil, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)

	err = client.Start()
	require.NoError(t, err)
	defer client.Stop()

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

func TestDataAvailabilityLayerClientRetrieval(t *testing.T) {
	//t.Skip("Skipping BNB client tests")

	// Create test config. By default, tests use BNB devnet.
	config := eth.EthConfig{
		Endpoint:   "https://ethereum-sepolia-rpc.publicnode.com",
		PrivateKey: "f3459c9fb5b720f52968f97e0dd895fa1caf3fe4a521fbc6395380bc50b0a234",
		Timeout:    500000000000,
		ChainId:    11155111,
		ApiUrl:     "https://ethereum-sepolia-beacon-api.publicnode.com",
	}
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
		TxHash:     "txhash",
		Commitment: []byte("commitment"),
		Proof:      []byte("proof"),
		BlockId:    8412136,
	}
	result := client.RetrieveBatches(meta.ToPath())
	require.NoError(t, result.Error)
	require.Equal(t, da.StatusSuccess, result.Code)

}
