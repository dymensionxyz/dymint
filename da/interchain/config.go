package interchain

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	interchaindatypes "github.com/dymensionxyz/interchain-da/x/interchain_da/types"
)

type DAConfig struct {
	ChainID        string                   `mapstructure:"chain_id"`  // The chain ID of the DA chain
	ClientID       string                   `mapstructure:"client_id"` // This is the IBC client ID on Dymension hub for the DA chain
	KeyringBackend string                   `mapstructure:"keyring_backend"`
	KeyringHomeDir string                   `mapstructure:"keyring_home_dir"`
	AddressPrefix  string                   `mapstructure:"da_address_prefix"`
	AccountName    string                   `mapstructure:"da_account_name"`
	NodeAddress    string                   `mapstructure:"da_node_address"`
	GasLimit       uint64                   `mapstructure:"da_gas_limit"`
	GasPrices      string                   `mapstructure:"da_gas_prices"`
	GasFees        string                   `mapstructure:"da_gas_fees"`
	ChainParams    interchaindatypes.Params `mapstructure:"chain_params"` // The params of the DA chain
}

type EncodingConfig interface {
	TxConfig() client.TxConfig
	Codec() codec.Codec
}
