package walrus

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	defaultRetryDelay = 3 * time.Second
)

var (
	defaultRetryAttempts = 5
	defaultSubmitBackoff = uretry.NewBackoffConfig(
		uretry.WithInitialDelay(time.Second*6),
		uretry.WithMaxDelay(time.Second*6),
	)
	defaultTimeout = 5 * time.Minute
)

type Config struct {
	PublisherUrl        string               `json:"publisher_url,omitempty"`
	AggregatorUrl       string               `json:"aggregator_url,omitempty"`
	BlobOwnerAddr       string               `json:"blob_owner_addr,omitempty"`
	StoreDurationEpochs int                  `json:"store_duration_epochs,omitempty"`
	Backoff             uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts       *int                 `json:"retry_attempts,omitempty"`
	RetryDelay          time.Duration        `json:"retry_delay,omitempty"`
	Timeout             time.Duration        `json:"timeout,omitempty"`
}

var TestConfig = Config{
	PublisherUrl:        "https://publisher.walrus-testnet.walrus.space",
	AggregatorUrl:       "https://aggregator.walrus-testnet.walrus.space",
	BlobOwnerAddr:       "0xcc7f20e6ca6d5b9076068bf9b40421218fdf2cfa6316f48c428c8b6716db9c05",
	StoreDurationEpochs: 180,
	RetryDelay:          defaultRetryDelay,
	RetryAttempts:       &defaultRetryAttempts,
	Backoff:             defaultSubmitBackoff,
	Timeout:             defaultTimeout,
}

// createConfig creates a new Config from the provided bytes and sets default values if needed
func createConfig(bz []byte) (c Config, err error) {
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
	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}
	return c, nil
}
