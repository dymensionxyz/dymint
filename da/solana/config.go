package solana

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	defaultRetryDelay    = 5 * time.Second
	defaultRetryAttempts = uint(10)
	MaxBlobSizeBytes     = 500000
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// Config stores Solana client configuration parameters.
type Config struct {
	KeyPathEnv             string               `json:"keypath_env,omitempty"`     // mnemonic used to generate key
	ApiKeyEnv              string               `json:"apikey_env,omitempty"`      // apikey used for the rpc client
	RetryAttempts          *uint                `json:"retry_attempts,omitempty"`  // num retries before failing when submitting or retrieving blobs
	RetryDelay             time.Duration        `json:"retry_delay,omitempty"`     // waiting time after failing before failing when submitting or retrieving blobs
	Backoff                uretry.BackoffConfig `json:"backoff,omitempty"`         // backoff function used before retrying after all retries failed when submitting
	Endpoint               string               `json:"endpoint,omitempty"`        // rpc endpoint
	ProgramAddress         string               `json:"program_address,omitempty"` // address of the Solana program used to write/read data
	SubmitTxRatePerSecond  *int                 `json:"tx_rate_second,omitempty"`  // rate limit to send transactions
	RequestTxRatePerSecond *int                 `json:"req_rate_second,omitempty"` // rate limit for querying transactions
}

var TestConfig = Config{
	KeyPathEnv: "SOLANA_KEYPATH",
	ApiKeyEnv:  "API_KEY",
	Endpoint:   "https://api.devnet.solana.com/",
	// ProgramAddress: "3ZjisFKx4KGHg3yRnq6FX7izAnt6gzyKiVfJz66Tdyqc",
	ProgramAddress: "5cfjxBnFMoqdbZXTMHaoXfQm7obMpYMnkT681sRd95Qo",
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

	if c.SubmitTxRatePerSecond != nil && *c.SubmitTxRatePerSecond <= 0 {
		return c, errors.New("rate must be positive")
	}

	if c.SubmitTxRatePerSecond != nil && *c.SubmitTxRatePerSecond <= 0 {
		return c, errors.New("rate must be positive")
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
