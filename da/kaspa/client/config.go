package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/da"
)

const (
	maxAddressesUtxo = uint32(100)
	MaxBlobSizeBytes = 500000
)

// Config stores Kaspa client configuration parameters.
type Config struct {
	da.BaseConfig `json:",inline"`
	da.KeyConfig  `json:",inline"`
	APIUrl        string `json:"api_url,omitempty"`      // Kaspa REST-API server (https://api.kaspa.org/docs), used to retrieve txs. It requires indexer+archival node.
	GrpcAddress   string `json:"grpc_address,omitempty"` // Kaspa node address+port used to submit txs using GRPC
	Address       string `json:"address,omitempty"`      // Address with funds used to send Kaspa Txs
	Network       string `json:"network,omitempty"`      // mainnet or testnet
}

var TestConfig = Config{
	BaseConfig: da.BaseConfig{
		Timeout: 5 * time.Second,
	},
	KeyConfig: da.KeyConfig{
		MnemonicPath: "/tmp/kaspa_mnemonic",
	},
	APIUrl:      "https://kaspa-testnet-rest.mzonder.com",
	GrpcAddress: "rpc.tn.kaspa.rollapp.network:443",
	Network:     "kaspa-testnet-10",
	Address:     "kaspatest:qzwyrgapjnhtjqkxdrmp7fpm3yddw296v2ajv9nmgmw5k3z0r38guevxyk7j0",
}

// CreateConfig generates config from da_config field received in DA client Init()
func CreateConfig(bz []byte) (c Config, err error) {
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
