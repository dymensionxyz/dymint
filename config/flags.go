package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tmcmd "github.com/tendermint/tendermint/cmd/cometbft/commands"
)

const (
	FlagBlockTime            = "dymint.block_time"
	FlagMaxIdleTime          = "dymint.max_idle_time"
	FlagBatchSubmitTime      = "dymint.batch_submit_time"
	FlagBatchSubmitBytes     = "dymint.batch_submit_bytes"
	FlagSkipValidationHeight = "dymint.skip_validation_height"
)

const (
	FlagSettlementLayer  = "dymint.settlement_layer"
	FlagSLNodeAddress    = "dymint.settlement_config.node_address"
	FlagSLKeyringBackend = "dymint.settlement_config.keyring_backend"
	FlagSLKeyringHomeDir = "dymint.settlement_config.keyring_home_dir"
	FlagSLDymAccountName = "dymint.settlement_config.dym_account_name"
	FlagSLGasLimit       = "dymint.settlement_config.gas_limit"
	FlagSLGasPrices      = "dymint.settlement_config.gas_prices"
	FlagSLGasFees        = "dymint.settlement_config.gas_fees"
)

const (
	FlagP2PListenAddress      = "dymint.p2p_config.listen_address"
	FlagP2PBootstrapNodes     = "dymint.p2p_config.bootstrap_nodes"
	FlagP2PGossipCacheSize    = "dymint.p2p_config.gossip_cache_size"
	FlagP2PBootstrapRetryTime = "dymint.p2p_config.bootstrap_retry_time"
)

// AddNodeFlags adds Dymint specific configuration options to cobra Command.
//
// This function is called in cosmos-sdk.
func AddNodeFlags(cmd *cobra.Command) {
	// Add tendermint default flags
	tmcmd.AddNodeFlags(cmd)

	def := DefaultNodeConfig

	cmd.Flags().Duration(FlagBlockTime, def.BlockTime, "block time (for sequencer mode)")
	cmd.Flags().Duration(FlagMaxIdleTime, def.MaxIdleTime, "max time for empty blocks (for sequencer mode)")
	cmd.Flags().Duration(FlagBatchSubmitTime, def.BatchSubmitTime, "max time for batch submit (for sequencer mode)")
	cmd.Flags().Uint64(FlagBatchSubmitBytes, def.BatchSubmitBytes, "block batch size in bytes")
	cmd.Flags().String(FlagSettlementLayer, def.SettlementLayer, "Settlement Layer Client name")
	cmd.Flags().String(FlagSLNodeAddress, def.SettlementConfig.NodeAddress, "Settlement Layer RPC node address")
	cmd.Flags().String(FlagSLKeyringBackend, def.SettlementConfig.KeyringBackend, "Sequencer keyring backend")
	cmd.Flags().String(FlagSLKeyringHomeDir, def.SettlementConfig.KeyringHomeDir, "Sequencer keyring path")
	cmd.Flags().String(FlagSLDymAccountName, def.SettlementConfig.DymAccountName, "Sequencer account name in keyring")
	cmd.Flags().String(FlagSLGasFees, def.SettlementConfig.GasFees, "Settlement Layer gas fees")
	cmd.Flags().String(FlagSLGasPrices, def.SettlementConfig.GasPrices, "Settlement Layer gas prices")
	cmd.Flags().Uint64(FlagSLGasLimit, def.SettlementConfig.GasLimit, "Settlement Layer batch submit gas limit")
	cmd.Flags().String(FlagP2PListenAddress, def.P2PConfig.ListenAddress, "P2P listen address")
	cmd.Flags().String(FlagP2PBootstrapNodes, def.P2PConfig.BootstrapNodes, "P2P bootstrap nodes")
	cmd.Flags().Duration(FlagP2PBootstrapRetryTime, def.P2PConfig.BootstrapRetryTime, "P2P bootstrap time")
	cmd.Flags().Uint64(FlagP2PGossipCacheSize, uint64(def.P2PConfig.GossipSubCacheSize), "P2P Gossiped blocks cache size") //nolint:gosec // GossipSubCacheSize should be always positive
	cmd.Flags().Uint64(FlagSkipValidationHeight, def.SkipValidationHeight, "Full-node validation will be skipped for the specified height")
}

func BindDymintFlags(cmd *cobra.Command, v *viper.Viper) error {
	if err := v.BindPFlag("block_time", cmd.Flags().Lookup(FlagBlockTime)); err != nil {
		return err
	}
	if err := v.BindPFlag("max_idle_time", cmd.Flags().Lookup(FlagMaxIdleTime)); err != nil {
		return err
	}
	if err := v.BindPFlag("batch_submit_time", cmd.Flags().Lookup(FlagBatchSubmitTime)); err != nil {
		return err
	}
	if err := v.BindPFlag("batch_submit_bytes", cmd.Flags().Lookup(FlagBatchSubmitBytes)); err != nil {
		return err
	}
	if err := v.BindPFlag("skip_validation_height", cmd.Flags().Lookup(FlagSkipValidationHeight)); err != nil {
		return err
	}
	if err := v.BindPFlag("settlement_layer", cmd.Flags().Lookup(FlagSettlementLayer)); err != nil {
		return err
	}
	if err := v.BindPFlag("node_address", cmd.Flags().Lookup(FlagSLNodeAddress)); err != nil {
		return err
	}
	if err := v.BindPFlag("keyring_backend", cmd.Flags().Lookup(FlagSLKeyringBackend)); err != nil {
		return err
	}
	if err := v.BindPFlag("keyring_home_dir", cmd.Flags().Lookup(FlagSLKeyringHomeDir)); err != nil {
		return err
	}
	if err := v.BindPFlag("dym_account_name", cmd.Flags().Lookup(FlagSLDymAccountName)); err != nil {
		return err
	}
	if err := v.BindPFlag("gas_fees", cmd.Flags().Lookup(FlagSLGasFees)); err != nil {
		return err
	}
	if err := v.BindPFlag("gas_prices", cmd.Flags().Lookup(FlagSLGasPrices)); err != nil {
		return err
	}
	if err := v.BindPFlag("gas_limit", cmd.Flags().Lookup(FlagSLGasLimit)); err != nil {
		return err
	}
	if err := v.BindPFlag("p2p_listen_address", cmd.Flags().Lookup(FlagP2PListenAddress)); err != nil {
		return err
	}
	if err := v.BindPFlag("p2p_gossiped_blocks_cache_size", cmd.Flags().Lookup(FlagP2PGossipCacheSize)); err != nil {
		return err
	}
	if err := v.BindPFlag("p2p_bootstrap_retry_time", cmd.Flags().Lookup(FlagP2PBootstrapRetryTime)); err != nil {
		return err
	}
	if err := v.BindPFlag("p2p_bootstrap_nodes", cmd.Flags().Lookup(FlagP2PBootstrapNodes)); err != nil {
		return err
	}

	return nil
}
