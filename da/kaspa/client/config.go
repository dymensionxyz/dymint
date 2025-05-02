package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	defaultRpcRetryDelay    = 3 * time.Second
	defaultRpcRetryAttempts = 5
	maxAddressesUtxo        = uint32(100)
	MaxBlobSizeBytes        = 500000
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// Config stores Sui DALC configuration parameters.
type Config struct {
	APIUrl        string               `json:"api_url,omitempty"`
	GrpcAddress   string               `json:"grpc_address,omitempty"`
	Timeout       time.Duration        `json:"timeout,omitempty"`
	MnemonicEnv   string               `json:"mnemonic_env,omitempty"`
	RetryAttempts *int                 `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration        `json:"retry_delay,omitempty"`
	Address       string               `json:"address,omitempty"`
	Network       string               `json:"network,omitempty"`
	Backoff       uretry.BackoffConfig `json:"backoff,omitempty"`
}

var TestConfig = Config{
	APIUrl:      "https://api-tn10.kaspa.org",
	GrpcAddress: "localhost:16210",
	Timeout:     5 * time.Second,
	MnemonicEnv: "KASPA_MNEMONIC",
	Network:     "testnet",
	Address:     "kaspatest:qzwyrgapjnhtjqkxdrmp7fpm3yddw296v2ajv9nmgmw5k3z0r38guevxyk7j0",
}

func CreateConfig(bz []byte) (c Config, err error) {
	if len(bz) <= 0 {
		return c, errors.New("supplied config is empty")
	}
	err = json.Unmarshal(bz, &c)
	if err != nil {
		return c, fmt.Errorf("json unmarshal: %w", err)
	}

	if c.RetryDelay == 0 {
		c.RetryDelay = defaultRpcRetryDelay
	}
	if c.Backoff == (uretry.BackoffConfig{}) {
		c.Backoff = defaultSubmitBackoff
	}
	if c.RetryAttempts == nil {
		attempts := defaultRpcRetryAttempts
		c.RetryAttempts = &attempts
	}
	return c, nil
}
