package eth

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
	defaultGasLimit         = uint64(21000) // set to 21k because its the necessary gas when no calldata or smart contract call (as is the case with simple blob EIP-4844 tx)
	defaultPrivateKeyEnv    = "ETH_PRIVATE_KEY"
	defaultTimeout          = 500000000000
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// Config stores Eth DALC configuration parameters.
type EthConfig struct {
	Timeout       time.Duration        `json:"timeout,omitempty"`
	Endpoint      string               `json:"endpoint"`
	PrivateKeyEnv string               `json:"private_key_env"`
	ChainId       uint64               `json:"chain_id"`
	ApiUrl        string               `json:"api_url"`
	Backoff       uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts *int                 `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration        `json:"retry_delay,omitempty"`
	GasLimit      *uint64              `json:"gas_limit"`
}

var TestConfig = EthConfig{
	Timeout:       500000000000,
	PrivateKeyEnv: "ETH_PRIVATE_KEY",
	Endpoint:      "https://ethereum-sepolia-rpc.publicnode.com",
	ChainId:       11155111,
	ApiUrl:        "https://ethereum-sepolia-beacon-api.publicnode.com",
}

func createConfig(bz []byte) (c EthConfig, err error) {
	if len(bz) <= 0 {
		return c, errors.New("supplied config is empty")
	}
	err = json.Unmarshal(bz, &c)
	if err != nil {
		return c, fmt.Errorf("json unmarshal: %w", err)
	}

	if c.PrivateKeyEnv == "" {
		c.PrivateKeyEnv = defaultPrivateKeyEnv
	}

	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
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
	if c.GasLimit == nil {
		gasLimit := defaultGasLimit
		c.GasLimit = &gasLimit
	}
	return c, nil
}
