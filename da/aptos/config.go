package aptos

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	defaultGasBudget        = "10000000" // 0.01 APT
	defaultRpcRetryDelay    = 3 * time.Second
	defaultRpcRetryAttempts = 5

	aptSymbol = "APT"
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// Config stores Aptos DALC configuration parameters.
type Config struct {
	Network       string               `json:"network,omitempty"`
	PriKeyEnv     string               `json:"pri_key_env,omitempty"`     // Environment variable name for private key (highest priority)
	PriKeyFile    string               `json:"pri_key_file,omitempty"`    // Path to file containing private key (second priority)
	PriKey        string               `json:"pri_key,omitempty"`         // Private key directly in config (lowest priority, fallback only)
	Backoff       uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts *int                 `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration        `json:"retry_delay,omitempty"`
}

var TestConfig = Config{
	Network:   "testnet",
	PriKeyEnv: "APT_PRIVATE_KEY",
}

func createConfig(bz []byte) (c Config, err error) {
	if len(bz) <= 0 {
		return c, errors.New("supplied config is empty")
	}
	err = json.Unmarshal(bz, &c)
	if err != nil {
		return c, fmt.Errorf("json unmarshal: %w", err)
	}

	if c.Network == "" {
		c.Network = "testnet"
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
