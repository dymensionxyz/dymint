package aptos

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/da"
)

const aptSymbol = "APT"

// Config stores Aptos DALC configuration parameters.
type Config struct {
	da.BaseConfig `json:",inline"`
	da.KeyConfig  `json:",inline"`
	Network       string `json:"network,omitempty"`
}

var TestConfig = Config{
	Network: "testnet",
	KeyConfig: da.KeyConfig{
		KeyPath: "/tmp/aptos_key.json",
	},
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

	// Set common defaults (retry, backoff, timeout)
	c.BaseConfig.SetDefaults()

	return c, nil
}
