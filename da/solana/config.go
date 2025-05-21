package solana

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	defaultRetryDelay    = 3 * time.Second
	defaultRetryAttempts = uint(5)
	MaxBlobSizeBytes     = 500000
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// Config stores Solana client configuration parameters.
type Config struct {
	Timeout        time.Duration        `json:"timeout,omitempty"`         // Timeout used in http requests
	KeyPathEnv     string               `json:"keypath_env,omitempty"`     // mnemonic used to generate key
	RetryAttempts  *uint                `json:"retry_attempts,omitempty"`  // num retries before failing when submitting or retrieving blobs
	RetryDelay     time.Duration        `json:"retry_delay,omitempty"`     // waiting time after failing before failing when submitting or retrieving blobs
	Backoff        uretry.BackoffConfig `json:"backoff,omitempty"`         // backoff function used before retrying after all retries failed when submitting
	Endpoint       string               `json:"endpoint,omitempty"`        // rpc endpoint
	ProgramAddress string               `json:"program_address,omitempty"` // address of the Solana program used to write/read data
}

var TestConfig = Config{
	Timeout:        5 * time.Second,
	KeyPathEnv:     "SOLANA_KEYPATH",
	Endpoint:       "http://barcelona:8899",
	ProgramAddress: "3ZjisFKx4KGHg3yRnq6FX7izAnt6gzyKiVfJz66Tdyqc",
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
