package interchain

import (
	"time"

	interchainda "github.com/dymensionxyz/dymint/types/pb/interchain_da"
)

type DAConfig struct {
	ClientID string `mapstructure:"client_id"` // IBC client ID between the Hub and DA layer
	ChainID  string `mapstructure:"chain_id"`  // Chain ID of the DA layer

	KeyringBackend string              `mapstructure:"keyring_backend"`
	KeyringHomeDir string              `mapstructure:"keyring_home_dir"`
	AddressPrefix  string              `mapstructure:"address_prefix"`
	AccountName    string              `mapstructure:"account_name"`
	NodeAddress    string              `mapstructure:"node_address"`
	GasLimit       uint64              `mapstructure:"gas_limit"`
	GasPrices      string              `mapstructure:"gas_prices"`
	GasFees        string              `mapstructure:"gas_fees"`
	DAParams       interchainda.Params `mapstructure:"da_params"`

	RetryMinDelay time.Duration `mapstructure:"retry_min_delay"`
	RetryMaxDelay time.Duration `mapstructure:"retry_min_delay"`
	RetryAttempts uint          `mapstructure:"retry_attempts"`
}

func DefaultDAConfig() DAConfig {
	return DAConfig{
		ClientID:       "",
		ChainID:        "",
		KeyringBackend: "",
		KeyringHomeDir: "",
		AddressPrefix:  "",
		AccountName:    "",
		NodeAddress:    "",
		GasLimit:       0,
		GasPrices:      "",
		GasFees:        "",
		DAParams:       interchainda.Params{},
		RetryMinDelay:  100 * time.Millisecond,
		RetryMaxDelay:  2 * time.Second,
		RetryAttempts:  10,
	}
}
