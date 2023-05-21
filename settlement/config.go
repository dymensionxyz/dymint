package settlement

import "github.com/ignite/cli/ignite/pkg/cosmosaccount"

// Config for the DymensionLayerClient
type Config struct {
	KeyringBackend cosmosaccount.KeyringBackend `json:"keyring_backend"`
	NodeAddress    string                       `json:"node_address"`
	KeyRingHomeDir string                       `json:"keyring_home_dir"`
	DymAccountName string                       `json:"dym_account_name"`
	RollappID      string                       `json:"rollapp_id"`
	GasLimit       uint64                       `json:"gas_limit"`
	GasPrices      string                       `json:"gas_prices"`
	GasFees        string                       `json:"gas_fees"`

	//For testing only. probably should be refactored
	ProposerPubKey string `json:"proposer_pub_key"`
}
