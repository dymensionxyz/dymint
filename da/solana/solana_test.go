package solana_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/dymensionxyz/dymint/da/solana"
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

}
