package sui

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	defaultGasBudget        = "10000000" // 0.01 SUI
	defaultRpcRetryDelay    = 3 * time.Second
	defaultRpcRetryAttempts = 5

	suiSymbol = "SUI"
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// Config stores Sui DALC configuration parameters.
type Config struct {
	RPCURL              string               `json:"rpc_url,omitempty"`
	NoopContractAddress string               `json:"noop_contract_address,omitempty"`
	GasBudget           string               `json:"gas_budget,omitempty"`
	Timeout             time.Duration        `json:"timeout,omitempty"`
	MnemonicEnv         string               `json:"mnemonic_env,omitempty"`
	Backoff             uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts       *int                 `json:"retry_attempts,omitempty"`
	RetryDelay          time.Duration        `json:"retry_delay,omitempty"`
}

var TestConfig = Config{
	RPCURL:              "https://fullnode.devnet.sui.io:443",
	NoopContractAddress: "0x45d86eb334f15b3a5145c0b7012dae3bf16de58ab4777ae31d184e9baf91c420",
	GasBudget:           "10000000", // 0.01 SUI
	Timeout:             5 * time.Second,
	MnemonicEnv:         "SUI_MNEMONIC",
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
