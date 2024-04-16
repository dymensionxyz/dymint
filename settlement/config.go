package settlement

import "errors"

// Config for the DymensionLayerClient
type Config struct {
	KeyringBackend string `mapstructure:"keyring_backend"`
	NodeAddress    string `mapstructure:"node_address"`
	KeyringHomeDir string `mapstructure:"keyring_home_dir"`
	DymAccountName string `mapstructure:"dym_account_name"`
	RollappID      string `mapstructure:"rollapp_id"`
	GasLimit       uint64 `mapstructure:"gas_limit"`
	GasPrices      string `mapstructure:"gas_prices"`
	GasFees        string `mapstructure:"gas_fees"`

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

	if c.RollappID == "" {
		return errors.New("must provide rollapp id")
	}

	return nil
}
