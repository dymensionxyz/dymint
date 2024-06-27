package interchain

import (
	interchainda "github.com/dymensionxyz/dymint/types/pb/interchain_da"
)

type DAConfig struct {
	ClientID       string              `mapstructure:"client_id"` // This is the IBC client ID on Dymension hub for the DA chain
	ChainID        string              `mapstructure:"chain_id"`  // The chain ID of the DA chain
	KeyringBackend string              `mapstructure:"keyring_backend"`
	KeyringHomeDir string              `mapstructure:"keyring_home_dir"`
	AddressPrefix  string              `mapstructure:"address_prefix"`
	AccountName    string              `mapstructure:"account_name"`
	NodeAddress    string              `mapstructure:"node_address"`
	GasLimit       uint64              `mapstructure:"gas_limit"`
	GasPrices      string              `mapstructure:"gas_prices"`
	GasFees        string              `mapstructure:"gas_fees"`
	DAParams       interchainda.Params `mapstructure:"da_params"`
}
