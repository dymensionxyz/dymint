package sui

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/da"
)

const (
	defaultGasBudget = "10000000" // 0.01 SUI
	suiSymbol        = "SUI"
)

// Config stores Sui DALC configuration parameters.
type Config struct {
	da.BaseConfig       `json:",inline"`
	da.KeyConfig        `json:",inline"`
	Endpoint            string `json:"endpoint,omitempty"`
	NoopContractAddress string `json:"noop_contract_address,omitempty"`
	GasBudget           string `json:"gas_budget,omitempty"`
}

var TestConfig = Config{
	BaseConfig: da.BaseConfig{
		Timeout: 5 * time.Second,
	},
	KeyConfig: da.KeyConfig{
		MnemonicPath: "/tmp/sui_mnemonic",
	},
	Endpoint:            "https://fullnode.devnet.sui.io:443",
	NoopContractAddress: "0x3dbdaa3db8d587deb38be3d4825ff434f1723054a6f43e04c0623f2c21a3f8a2",
	GasBudget:           defaultGasBudget,
}

func createConfig(bz []byte) (c Config, err error) {
	if len(bz) <= 0 {
		return c, errors.New("supplied config is empty")
	}
	err = json.Unmarshal(bz, &c)
	if err != nil {
		return c, fmt.Errorf("json unmarshal: %w", err)
	}

	if c.GasBudget == "" {
		c.GasBudget = defaultGasBudget
	}

	// Set common defaults (retry, backoff, timeout)
	c.BaseConfig.SetDefaults()

	return c, nil
}
