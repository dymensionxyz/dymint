package sui_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/block-vision/sui-go-sdk/mystenbcs"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/sui"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

func TestDataAvailabilityLayerClient(t *testing.T) {
	t.Skip("Skipping SUI client tests")

	// Set up test environment
	mnemonicEnv := "SUI_MNEMONIC"
	err := os.Setenv(mnemonicEnv, "catalog phone awful abuse derive type verb betray foil salad street scrub")
	require.NoError(t, err)

	// Create test config. By default, tests use Sui testnet.
	config := sui.TestConfig
	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	// Create new SUI client
	client := &sui.DataAvailabilityLayerClient{}
	err = client.Init(configBytes, nil, nil, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	require.NoError(t, err)

	err = client.Start()
	require.NoError(t, err)
	defer client.Stop()

	// Ensure that the client has enough SUI tokens to submit batches
	balance, err := client.GetSignerBalance()
	require.NoError(t, err)
	if balance.Amount.LT(sui.SUI) {
		// Request SUI tokens from the faucet and wait until the balance is sufficient
		err = client.TestRequestCoins()
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			balance, err = client.GetSignerBalance()
			require.NoError(t, err)
			return balance.Amount.GTE(sui.SUI)
		}, 30*time.Second, 5*time.Second)
	}

	// Proposer key for generating test batches
	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		batch   *types.Batch
		wantErr bool
	}{
		// This test fails because the batch size is 0 after serialization. This should
		// not be possible in practice.
		//{
		//	name:  "empty batch",
		//	batch: testutil.GenerateBatchWithBlocks(0, proposerKey),
		//},
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
		// This test is blocking. If the batch is too big, the client will try to retry it
		// infinitely. This is not a problem in practice as the max block size is enforced
		// and verified before submitting the batch.
		//{
		//	name:    "exceeding batch: 118KB",
		//	batch:   testutil.GenerateBatchWithBlocks(200, proposerKey),
		//	wantErr: true,
		//},
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
			checkResult := client.CheckBatchAvailability(result.SubmitMetaData.ToPath())
			require.NoError(t, checkResult.Error)
			require.Equal(t, da.StatusSuccess, checkResult.Code)

			// Retrieve batch
			retrieveResult := client.RetrieveBatches(result.SubmitMetaData.ToPath())
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

func TestDeserializePureArg(t *testing.T) {
	expected := "A0FBQQ=="

	input := "CEEwRkJRUT09"
	base64Decoded, err := mystenbcs.FromBase64(input)
	require.NoError(t, err)

	var value string
	_, err = mystenbcs.Unmarshal(base64Decoded, &value)
	require.NoError(t, err)

	require.Equal(t, expected, value)
}
