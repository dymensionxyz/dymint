package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	defaultRetryDelay    = 5 * time.Second
	defaultRetryAttempts = uint(5)
	maxAddressesUtxo     = uint32(100)
	MaxBlobSizeBytes     = 500000
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// Config stores Kaspa client configuration parameters.
type Config struct {
	APIUrl        string               `json:"api_url,omitempty"`        // Kaspa REST-API server (https://api.kaspa.org/docs), used to retrieve txs. It requires indexer+archival node.
	GrpcAddress   string               `json:"grpc_address,omitempty"`   // Kaspa node address+port used to submit txs using GRPC
	Timeout       time.Duration        `json:"timeout,omitempty"`        // Timeout used in http requests
	MnemonicEnv   string               `json:"mnemonic_env,omitempty"`   // mnemonic used to generate key
	RetryAttempts *uint                `json:"retry_attempts,omitempty"` // num retries before failing when submitting or retrieving blobs
	RetryDelay    time.Duration        `json:"retry_delay,omitempty"`    // waiting time after failing before failing when submitting or retrieving blobs
	Address       string               `json:"address,omitempty"`        // Address with funds used to send Kaspa Txs
	Network       string               `json:"network,omitempty"`        // mainnet or testnet
	Backoff       uretry.BackoffConfig `json:"backoff,omitempty"`        // backoff function used before retrying after all retries failed when submitting
}

var TestConfig = Config{
	APIUrl:      "https://api-tn10.kaspa.org",
	GrpcAddress: "localhost:16210",
	Timeout:     5 * time.Second,
	MnemonicEnv: "KASPA_MNEMONIC",
	Network:     "kaspa-testnet-10",
	Address:     "kaspatest:qzwyrgapjnhtjqkxdrmp7fpm3yddw296v2ajv9nmgmw5k3z0r38guevxyk7j0",
}

// CreateConfig, generates config from da_config field received in DA client Init()
func CreateConfig(bz []byte) (c Config, err error) {
	if len(bz) <= 0 {
		return c, errors.New("supplied config is empty")
	}
	err = json.Unmarshal(bz, &c)
	if err != nil {
		return c, fmt.Errorf("json unmarshal: %w", err)
	}

	if c.RetryDelay == 0 {
		c.RetryDelay = defaultRetryDelay
	}
	if c.Backoff == (uretry.BackoffConfig{}) {
		c.Backoff = defaultSubmitBackoff
	}
	if c.RetryAttempts == nil {
		attempts := defaultRetryAttempts
		c.RetryAttempts = &attempts
	}
	return c, nil
}
