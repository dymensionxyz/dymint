package avail

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/da"
)

// Config stores Avail DALC configuration parameters.
type Config struct {
	da.BaseConfig `json:",inline"`
	da.KeyConfig  `json:",inline"`
	RpcEndpoint   string `json:"endpoint,omitempty"`
	AppID         uint32 `json:"app_id,omitempty"`
}

var TestConfig = Config{
	RpcEndpoint: "wss://turing-rpc.avail.so/ws",
	AppID:       1,
	KeyConfig: da.KeyConfig{
		MnemonicPath: "/tmp/avail_mnemonic",
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

	// Set common defaults (retry, backoff, timeout)
	c.BaseConfig.SetDefaults()

	return c, nil
}
