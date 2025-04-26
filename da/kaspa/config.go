package kaspa

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	defaultRpcRetryDelay      = 3 * time.Second
	defaultRpcRetryAttempts   = 5
	maxBlobSizeBytes          = 100000
	defaultBatchRetryDelay    = 10 * time.Second
	defaultBatchRetryAttempts = 10
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// Config stores Sui DALC configuration parameters.
type Config struct {
	RPCURL        string               `json:"rpc_url,omitempty"`
	GrpcAddress   string               `json:"grpc_address,omitempty"`
	Timeout       time.Duration        `json:"timeout,omitempty"`
	MnemonicEnv   string               `json:"mnemonic_env,omitempty"`
	Backoff       uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts *int                 `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration        `json:"retry_delay,omitempty"`
	KeysPath      string               `json:"keys_path,omitempty"`
}

var TestConfig = Config{
	RPCURL:      "https://api-tn10.kaspa.org",
	GrpcAddress: "localhost:16210",
	Timeout:     5 * time.Second,
	MnemonicEnv: "KASPA_MNEMONIC",
}

func createConfig(bz []byte) (c Config, err error) {
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
