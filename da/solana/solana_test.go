package solana_test

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/dymensionxyz/dymint/da/solana"
	"github.com/dymensionxyz/dymint/testutil"
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

	// Create test config. By default, tests use Sui testnet.
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

	// Ensure that the client has enough SUI tokens to submit batches
	balance, err := client.GetSignerBalance()
	require.NoError(t, err)
	assert.Greater(t, balance.Amount.BigInt().Cmp(big.NewInt(5000)), 0, "at least balance of 5000 lamport required to send a tx")

	// Proposer key for generating test batches
	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	batch := testutil.GenerateBatchWithBlocks(1, proposerKey)
	result := client.SubmitBatch(batch)
	require.NoError(t, result.Error)

}
