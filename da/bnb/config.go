package bnb

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/da"
)

// Config stores BNB DALC configuration parameters.
type Config struct {
	da.BaseConfig `json:",inline"`
	Endpoint      string `json:"endpoint,omitempty"`
	PrivateKey    string `json:"private_key_hex,omitempty"`
	ChainId       uint64 `json:"chain_id,omitempty"`
}

var TestConfig = Config{
	Endpoint: "https://bsc-testnet-rpc.publicnode.com",
	ChainId:  97, // BSC testnet
}

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

	return c, nil
}
