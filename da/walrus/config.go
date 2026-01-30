package walrus

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/da"
)

const maxTestnetEpochs = 53

var defaultTimeout = 5 * time.Minute

// Config stores Walrus DALC configuration parameters.
type Config struct {
	da.BaseConfig       `json:",inline"`
	PublisherUrl        string `json:"publisher_url,omitempty"`
	AggregatorUrl       string `json:"aggregator_url,omitempty"`
	BlobOwnerAddr       string `json:"blob_owner_addr,omitempty"`
	StoreDurationEpochs int    `json:"store_duration_epochs,omitempty"`
}

var TestConfig = Config{
	BaseConfig: da.BaseConfig{
		Timeout: defaultTimeout,
	},
	PublisherUrl:        "https://publisher.walrus-testnet.walrus.space",
	AggregatorUrl:       "https://aggregator.walrus-testnet.walrus.space",
	BlobOwnerAddr:       "0xcc7f20e6ca6d5b9076068bf9b40421218fdf2cfa6316f48c428c8b6716db9c05",
	StoreDurationEpochs: maxTestnetEpochs,
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

	// Set common defaults (retry, backoff, timeout)
	c.BaseConfig.SetDefaults()

	// Override default timeout for Walrus (longer than standard)
	if c.Timeout == da.DefaultTimeout {
		c.Timeout = defaultTimeout
	}

	return c, nil
}
