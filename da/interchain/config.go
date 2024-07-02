package interchain

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"

	interchainda "github.com/dymensionxyz/dymint/types/pb/interchain_da"
)

type DAConfig struct {
	ClientID string `mapstructure:"client_id" json:"client_id,omitempty"` // IBC client ID between the Hub and DA layer
	ChainID  string `mapstructure:"chain_id" json:"chain_id,omitempty"`   // Chain ID of the DA layer

	KeyringBackend string              `mapstructure:"keyring_backend" json:"keyring_backend,omitempty"`
	KeyringHomeDir string              `mapstructure:"keyring_home_dir" json:"keyring_home_dir,omitempty"`
	AddressPrefix  string              `mapstructure:"address_prefix" json:"address_prefix,omitempty"`
	AccountName    string              `mapstructure:"account_name" json:"account_name,omitempty"`
	NodeAddress    string              `mapstructure:"node_address" json:"node_address,omitempty"`
	GasLimit       uint64              `mapstructure:"gas_limit" json:"gas_limit,omitempty"`
	GasPrices      string              `mapstructure:"gas_prices" json:"gas_prices,omitempty"`
	GasFees        string              `mapstructure:"gas_fees" json:"gas_fees,omitempty"`
	DAParams       interchainda.Params `mapstructure:"da_params" json:"da_params"`

	RetryMinDelay time.Duration `mapstructure:"retry_min_delay" json:"retry_min_delay,omitempty"`
	RetryMaxDelay time.Duration `mapstructure:"retry_min_delay" json:"retry_max_delay,omitempty"`
	RetryAttempts uint          `mapstructure:"retry_attempts" json:"retry_attempts,omitempty"`
}

func (c *DAConfig) Verify() error {
	if c.ClientID == "" {
		return errors.New("missing IBC client ID")
	}

	if c.ChainID == "" {
		return errors.New("missing chain ID of the DA layer")
	}

	if c.KeyringBackend == "" {
		return errors.New("missing keyring backend")
	}

	if c.KeyringHomeDir == "" {
		return errors.New("missing keyring home dir")
	}

	if c.AddressPrefix == "" {
		return errors.New("missing address prefix")
	}

	if c.AccountName == "" {
		return errors.New("missing account name")
	}

	if c.NodeAddress == "" {
		return errors.New("missing node address")
	}

	// GasLimit may be 0

	if c.GasPrices == "" && c.GasFees == "" {
		return errors.New("either gas prices or gas_prices are required")
	}

	if c.GasPrices != "" && c.GasFees != "" {
		return errors.New("cannot provide both fees and gas prices")
	}

	// DAParams are set during Init

	if c.RetryMinDelay.Nanoseconds() == 0 {
		return errors.New("missing retry min delay")
	}

	if c.RetryMaxDelay.Nanoseconds() == 0 {
		return errors.New("missing retry max delay")
	}

	if c.RetryAttempts == 0 {
		return errors.New("missing retry attempts")
	}

	return nil
}

func DefaultDAConfig() DAConfig {
	home, _ := os.UserHomeDir()
	keyringHomeDir := filepath.Join(home, ".simapp")
	return DAConfig{
		ClientID:       "dym-interchain",
		ChainID:        "interchain-da-test",
		KeyringBackend: keyring.BackendTest,
		KeyringHomeDir: keyringHomeDir,
		AddressPrefix:  sdk.Bech32MainPrefix,
		AccountName:    "sequencer",
		NodeAddress:    "http://127.0.0.1:36657",
		GasLimit:       0,
		GasPrices:      "10stake",
		GasFees:        "",
		DAParams:       interchainda.Params{},
		RetryMinDelay:  100 * time.Millisecond,
		RetryMaxDelay:  2 * time.Second,
		RetryAttempts:  10,
	}
}
