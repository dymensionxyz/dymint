package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	defaultRpcRetryDelay    = 3 * time.Second
	defaultRpcRetryAttempts = 5
	MaxBlobSizeBytes        = 22000
)

// Config stores Sui DALC configuration parameters.
type Config struct {
	APIUrl        string        `json:"api_url,omitempty"`
	GrpcAddress   string        `json:"grpc_address,omitempty"`
	Timeout       time.Duration `json:"timeout,omitempty"`
	MnemonicEnv   string        `json:"mnemonic_env,omitempty"`
	RetryAttempts *int          `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration `json:"retry_delay,omitempty"`
	FromAddress   string        `json:"from_address,omitempty"`
	Network       string        `json:"network,omitempty"`
}

var TestConfig = Config{
	APIUrl:      "https://api-tn10.kaspa.org",
	GrpcAddress: "localhost:16210",
	Timeout:     5 * time.Second,
	MnemonicEnv: "KASPA_MNEMONIC",
	Network:     "testnet",
	FromAddress: "kaspatest:qp75u7cuphjwyq9j6ghe2v0j3gtvxlppyurq279h4ckpdc7umdh6vrusw9c7d",
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

	if c.RetryAttempts == nil {
		attempts := defaultRpcRetryAttempts
		c.RetryAttempts = &attempts
	}
	return c, nil
}
