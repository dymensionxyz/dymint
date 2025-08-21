package settlement

import (
	"errors"
	"time"
)

// Config for the DymensionLayerClient
type Config struct {
	KeyringBackend          string        `mapstructure:"keyring_backend"`
	NodeAddress             string        `mapstructure:"settlement_node_address"`
	KeyringHomeDir          string        `mapstructure:"keyring_home_dir"`
	DymAccountName          string        `mapstructure:"dym_account_name"`
	GasLimit                uint64        `mapstructure:"settlement_gas_limit"`
	GasPrices               string        `mapstructure:"settlement_gas_prices"`
	GasFees                 string        `mapstructure:"settlement_gas_fees"`
	RetryAttempts           uint          `mapstructure:"retry_attempts"`
	RetryMaxDelay           time.Duration `mapstructure:"retry_max_delay"`
	RetryMinDelay           time.Duration `mapstructure:"retry_min_delay"`
	BatchAcceptanceTimeout  time.Duration `mapstructure:"batch_acceptance_timeout"`
	BatchAcceptanceAttempts uint          `mapstructure:"batch_acceptance_attempts"`
	VerboseDebugFile        string        `mapstructure:"verbose_debug_file"`
	// For testing only. probably should be refactored
	ProposerPubKey string `json:"proposer_pub_key"`
	// Config used for sl shared grpc mock
	SLGrpc GrpcConfig `mapstructure:",squash"`
}

type GrpcConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	RefreshTime int    `json:"refresh_time"`
}

func (c Config) Validate() error {
	if c.GasPrices != "" && c.GasFees != "" {
		return errors.New("cannot provide both fees and gas prices")
	}

	if c.GasPrices == "" && c.GasFees == "" {
		return errors.New("must provide either fees or gas prices")
	}

	return nil
}
