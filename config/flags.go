package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tmcmd "github.com/tendermint/tendermint/cmd/cometbft/commands"
)

const (
	FlagAggregator             = "dymint.aggregator"
	FlagDALayer                = "dymint.da_layer"
	FlagDAConfig               = "dymint.da_config"
	FlagBlockTime              = "dymint.block_time"
	FlagEmptyBlocksMaxTime     = "dymint.empty_blocks_max_time"
	FlagBatchSubmitMaxTime     = "dymint.batch_submit_max_time"
	FlagNamespaceID            = "dymint.namespace_id"
	FlagBlockBatchSize         = "dymint.block_batch_size"
	FlagBlockBatchMaxSizeBytes = "dymint.block_batch_max_size_bytes"
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
	FlagRollappID        = "dymint.settlement_config.rollapp_id"
)

// AddFlags adds Dymint specific configuration options to cobra Command.
//
// This function is called in cosmos-sdk.
func AddNodeFlags(cmd *cobra.Command) {
	//Add tendermint default flags
	tmcmd.AddNodeFlags(cmd)

	def := DefaultNodeConfig

	cmd.Flags().Bool(FlagAggregator, false, "run node in aggregator mode")
	cmd.Flags().String(FlagDALayer, def.DALayer, "Data Availability Layer Client name (mock or grpc")
	cmd.Flags().String(FlagDAConfig, def.DAConfig, "Data Availability Layer Client config")
	cmd.Flags().Duration(FlagBlockTime, def.BlockTime, "block time (for aggregator mode)")
	cmd.Flags().Duration(FlagEmptyBlocksMaxTime, def.EmptyBlocksMaxTime, "max time for empty blocks (for aggregator mode)")
	cmd.Flags().Duration(FlagBatchSubmitMaxTime, def.BatchSubmitMaxTime, "max time for batch submit (for aggregator mode)")
	cmd.Flags().String(FlagNamespaceID, def.NamespaceID, "namespace identifies (8 bytes in hex)")
	cmd.Flags().Uint64(FlagBlockBatchSize, def.BlockBatchSize, "block batch size")
	cmd.Flags().Uint64(FlagBlockBatchMaxSizeBytes, def.BlockBatchMaxSizeBytes, "block batch size in bytes")

	cmd.Flags().String(FlagSettlementLayer, def.SettlementLayer, "Settlement Layer Client name")
	cmd.Flags().String(FlagSLNodeAddress, def.SettlementConfig.NodeAddress, "Settlement Layer RPC node address")
	cmd.Flags().String(FlagSLKeyringBackend, def.SettlementConfig.KeyringBackend, "Sequencer keyring backend")
	cmd.Flags().String(FlagSLKeyringHomeDir, def.SettlementConfig.KeyringHomeDir, "Sequencer keyring path")
	cmd.Flags().String(FlagSLDymAccountName, def.SettlementConfig.DymAccountName, "Sequencer account name in keyring")
	cmd.Flags().String(FlagSLGasFees, def.SettlementConfig.GasFees, "Settlement Layer gas fees")
	cmd.Flags().String(FlagSLGasPrices, def.SettlementConfig.GasPrices, "Settlement Layer gas prices")
	cmd.Flags().Uint64(FlagSLGasLimit, def.SettlementConfig.GasLimit, "Settlement Layer batch submit gas limit")
	cmd.Flags().String(FlagRollappID, def.SettlementConfig.RollappID, "The chainID of the rollapp")
}

func BindDymintFlags(cmd *cobra.Command, v *viper.Viper) error {
	if err := v.BindPFlag("aggregator", cmd.Flags().Lookup(FlagAggregator)); err != nil {
		return err
	}
	if err := v.BindPFlag("da_layer", cmd.Flags().Lookup(FlagDALayer)); err != nil {
		return err
	}
	if err := v.BindPFlag("da_config", cmd.Flags().Lookup(FlagDAConfig)); err != nil {
		return err
	}
	if err := v.BindPFlag("block_time", cmd.Flags().Lookup(FlagBlockTime)); err != nil {
		return err
	}
	if err := v.BindPFlag("empty_blocks_max_time", cmd.Flags().Lookup(FlagEmptyBlocksMaxTime)); err != nil {
		return err
	}
	if err := v.BindPFlag("batch_submit_max_time", cmd.Flags().Lookup(FlagBatchSubmitMaxTime)); err != nil {
		return err
	}
	if err := v.BindPFlag("namespace_id", cmd.Flags().Lookup(FlagNamespaceID)); err != nil {
		return err
	}
	if err := v.BindPFlag("block_batch_size", cmd.Flags().Lookup(FlagBlockBatchSize)); err != nil {
		return err
	}
	if err := v.BindPFlag("block_batch_max_size_bytes", cmd.Flags().Lookup(FlagBlockBatchMaxSizeBytes)); err != nil {
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
	if err := v.BindPFlag("rollapp_id", cmd.Flags().Lookup(FlagRollappID)); err != nil {
		return err
	}
	return nil
}
