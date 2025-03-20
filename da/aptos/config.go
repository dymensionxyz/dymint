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
	ChainID       int64                `json:"chain_id,omitempty"`
	GasBudget     string               `json:"gas_budget,omitempty"`
	Timeout       time.Duration        `json:"timeout,omitempty"`
	PriKeyEnv     string               `json:"mnemonic_env,omitempty"`
	Backoff       uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts *int                 `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration        `json:"retry_delay,omitempty"`
}

var TestConfig = Config{
	GasBudget: "10000000", // 0.01 APT
	Timeout:   5 * time.Second,
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

	if c.GasBudget == "" {
		c.GasBudget = defaultGasBudget
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
