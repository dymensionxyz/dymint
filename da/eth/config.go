package eth

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/da"
)

const (
	defaultGasLimit = uint64(21000) // standard gas for blob tx execution (blob gas is separate)
)

// Config stores Eth DALC configuration parameters.
type Config struct {
	da.BaseConfig `json:",inline"`
	da.KeyConfig  `json:",inline"`
	Endpoint      string  `json:"endpoint,omitempty"`
	ChainId       uint64  `json:"chain_id,omitempty"`
	ApiUrl        string  `json:"api_url,omitempty"`
	GasLimit      *uint64 `json:"gas_limit,omitempty"`
}

var TestConfig = Config{
	KeyConfig: da.KeyConfig{
		KeyPath: "/tmp/eth_key.json",
	},
	Endpoint: "https://ethereum-sepolia-rpc.publicnode.com",
	ChainId:  11155111,
	ApiUrl:   "https://ethereum-sepolia-beacon-api.publicnode.com",
}

func createConfig(bz []byte) (c Config, err error) {
	if len(bz) <= 0 {
		return c, errors.New("supplied config is empty")
	}
	err = json.Unmarshal(bz, &c)
	if err != nil {
		return c, fmt.Errorf("json unmarshal: %w", err)
	}

	if c.GasLimit == nil {
		gasLimit := defaultGasLimit
		c.GasLimit = &gasLimit
	}

	// Set common defaults (retry, backoff, timeout)
	c.BaseConfig.SetDefaults()

	return c, nil
}
