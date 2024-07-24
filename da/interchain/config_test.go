package interchain_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/da/interchain"
	interchainda "github.com/dymensionxyz/dymint/types/pb/interchain_da"
)

func TestDAConfig_Verify(t *testing.T) {
	defaultConfig := interchain.DefaultDAConfig()

	tests := map[string]struct {
		DaConfig      func(interchain.DAConfig) interchain.DAConfig
		expectedError error
	}{
		"valid configuration": {
			DaConfig: func(interchain.DAConfig) interchain.DAConfig {
				return interchain.DAConfig{
					ClientID:       "test client id",
					ChainID:        "test chain id",
					KeyringBackend: "test keyring backend",
					KeyringHomeDir: "test keyring home dir",
					AddressPrefix:  "/",
					AccountName:    "test account name",
					NodeAddress:    "test node address",
					GasLimit:       1,
					GasPrices:      "1.00",
					GasFees:        "",
					DAParams:       interchainda.Params{},
					RetryMaxDelay:  time.Second * 1,
					RetryMinDelay:  time.Second * 1,
					RetryAttempts:  1,
				}
			},
			expectedError: nil,
		},

		"default is valid": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				return d
			},
			expectedError: nil,
		},
		"missing client ID": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.ClientID = ""
				return d
			},
			expectedError: errors.New("missing IBC client ID"),
		},
		"missing chain ID": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.ChainID = ""
				return d
			},
			expectedError: errors.New("missing chain ID of the DA layer"),
		},
		"missing keyring backend": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.KeyringBackend = ""
				return d
			},
			expectedError: errors.New("missing keyring backend"),
		},
		"missing keyring home dir": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.KeyringHomeDir = ""
				return d
			},
			expectedError: errors.New("missing keyring home dir"),
		},
		"missing address prefix": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.AddressPrefix = ""
				return d
			},
			expectedError: errors.New("missing address prefix"),
		},
		"missing account name": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.AccountName = ""
				return d
			},
			expectedError: errors.New("missing account name"),
		},
		"missing node address": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.NodeAddress = ""
				return d
			},
			expectedError: errors.New("missing node address"),
		},
		"both gas prices and gas fees are missing": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.GasPrices = ""
				d.GasFees = ""
				return d
			},
			expectedError: errors.New("either gas prices or gas_prices are required"),
		},
		"both gas prices and gas fees are specified": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.GasPrices = "10stake"
				d.GasFees = "10stake"
				return d
			},
			expectedError: errors.New("cannot provide both fees and gas prices"),
		},
		"missing retry min delay": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.RetryMinDelay = 0
				return d
			},
			expectedError: errors.New("missing retry min delay"),
		},
		"missing retry max delay": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.RetryMaxDelay = 0
				return d
			},
			expectedError: errors.New("missing retry max delay"),
		},
		"missing retry attempts": {
			DaConfig: func(d interchain.DAConfig) interchain.DAConfig {
				d.RetryAttempts = 0
				return d
			},
			expectedError: errors.New("missing retry attempts"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Generate test case config
			tcConf := tc.DaConfig(defaultConfig)

			err := tcConf.Verify()

			if tc.expectedError != nil {
				require.ErrorContains(t, err, tc.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test(t *testing.T) {
	conf := interchain.DefaultDAConfig()
	data, err := json.Marshal(conf)
	require.NoError(t, err)
	t.Log(string(data))
}
