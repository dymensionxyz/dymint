package solana

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/da"
)

const (
	MaxBlobSizeBytes      = 500000
	defaultProgramAddress = "5cfjxBnFMoqdbZXTMHaoXfQm7obMpYMnkT681sRd95Qo"
)

// Config stores Solana client configuration parameters.
type Config struct {
	da.BaseConfig          `json:",inline"`
	KeyPathEnv             string `json:"keypath_env,omitempty"`     // env var for key file path
	ApiKeyEnv              string `json:"apikey_env,omitempty"`      // env var for API key
	Endpoint               string `json:"endpoint,omitempty"`        // rpc endpoint
	ProgramAddress         string `json:"program_address,omitempty"` // address of the Solana program used to write/read data
	SubmitTxRatePerSecond  *int   `json:"tx_rate_second,omitempty"`  // rate limit to send transactions
	RequestTxRatePerSecond *int   `json:"req_rate_second,omitempty"` // rate limit for querying transactions
}

var TestConfig = Config{
	KeyPathEnv:     "SOLANA_KEYPATH",
	ApiKeyEnv:      "API_KEY",
	Endpoint:       "https://api.devnet.solana.com/",
	ProgramAddress: defaultProgramAddress,
}

// createConfig generates config from da_config field received in DA client Init()
func createConfig(bz []byte) (c Config, err error) {
	if len(bz) <= 0 {
		return c, errors.New("supplied config is empty")
	}
	err = json.Unmarshal(bz, &c)
	if err != nil {
		return c, fmt.Errorf("json unmarshal: %w", err)
	}

	if c.SubmitTxRatePerSecond != nil && *c.SubmitTxRatePerSecond <= 0 {
		return c, errors.New("tx rate must be positive")
	}

	if c.RequestTxRatePerSecond != nil && *c.RequestTxRatePerSecond <= 0 {
		return c, errors.New("request rate must be positive")
	}

	if c.ProgramAddress == "" {
		c.ProgramAddress = defaultProgramAddress
	}

	// Set common defaults (retry, backoff, timeout)
	c.BaseConfig.SetDefaults()

	return c, nil
}
