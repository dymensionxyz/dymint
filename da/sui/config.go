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
	RPCURL              string `json:"rpc_url,omitempty"`
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
	RPCURL:              "https://fullnode.devnet.sui.io:443",
	NoopContractAddress: "0x45d86eb334f15b3a5145c0b7012dae3bf16de58ab4777ae31d184e9baf91c420",
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
