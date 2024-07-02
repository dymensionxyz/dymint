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

func (c *DAConfig) Verify() error {
	if c.ClientID == "" {
		return errors.New("missing IBC client ID")
	}

	if c.ChainID == "" {
		return errors.New("missing Chain ID of the DA layer")
	}

	if c.KeyringBackend == "" {
		return errors.New("missing Keyring Backend")
	}

	if c.KeyringHomeDir == "" {
		return errors.New("missing Keyring HomeDir")
	}

	if c.AddressPrefix == "" {
		return errors.New("missing Address Prefix")
	}

	if c.AccountName == "" {
		return errors.New("missing Account Name")
	}

	if c.NodeAddress == "" {
		return errors.New("missing Node Address")
	}

	if c.GasLimit == 0 {
		return errors.New("missing Gas Limit")
	}

	if c.GasPrices == "" && c.GasFees == "" {
		return errors.New("either gas prices or gas_prices are required")
	}

	if c.GasPrices != "" && c.GasFees != "" {
		return errors.New("cannot provide both fees and gas prices")
	}

	// DAParams are set during Init

	if c.RetryMinDelay.Nanoseconds() == 0 {
		return errors.New("missing Retry MinDelay")
	}

	if c.RetryMaxDelay.Nanoseconds() == 0 {
		return errors.New("missing Retry MaxDelay")
	}

	if c.RetryAttempts == 0 {
		return errors.New("missing Retry Attempts")
	}

	return nil
}

func DefaultDAConfig() DAConfig {
	home, _ := os.UserHomeDir()
	keyringHomeDir := filepath.Join(home, ".simapp")
	return DAConfig{
		ClientID:       "",
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
