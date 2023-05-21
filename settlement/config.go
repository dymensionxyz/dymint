package settlement

// Config for the DymensionLayerClient
type Config struct {
	NodeAddress    string `mapstructure:"node_address"`
	KeyRingHomeDir string `mapstructure:"keyring_home_dir"`
	DymAccountName string `mapstructure:"dym_account_name"`
	RollappID      string `mapstructure:"rollapp_id"`
	GasLimit       uint64 `mapstructure:"gas_limit"`
	GasPrices      string `mapstructure:"gas_prices"`
	GasFees        string `mapstructure:"gas_fees"`

	//For testing only. probably should be refactored
	ProposerPubKey string `json:"proposer_pub_key"`
}
