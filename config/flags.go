package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tmcmd "github.com/tendermint/tendermint/cmd/cometbft/commands"
)

const (
	flagAggregator                   = "dymint.aggregator"
	flagDALayer                      = "dymint.da_layer"
	flagDAConfig                     = "dymint.da_config"
	flagBlockTime                    = "dymint.block_time"
	flagEmptyBlocksMaxTime           = "dymint.empty_blocks_max_time"
	flagBatchSubmitMaxTime           = "dymint.batch_submit_max_time"
	flagNamespaceID                  = "dymint.namespace_id"
	flagBlockBatchSize               = "dymint.block_batch_size"
	flagBlockBatchMaxSizeBytes       = "dymint.block_batch_max_size_bytes"
	flagSimulateWrongCommitmentFraud = "dymint.simulate_wrongcommitment_fraud"
	flagSimulateNonInclusionFraud    = "dymint.simulate_noninclusion_fraud"
	flagEnableInclusionProof         = "dymint.enable_inclusion_proof"
)

const (
	flagSettlementLayer  = "dymint.settlement_layer"
	flagSLNodeAddress    = "dymint.settlement_config.node_address"
	flagSLKeyringBackend = "dymint.settlement_config.keyring_backend"
	flagSLKeyringHomeDir = "dymint.settlement_config.keyring_home_dir"
	flagSLDymAccountName = "dymint.settlement_config.dym_account_name"
	flagSLGasLimit       = "dymint.settlement_config.gas_limit"
	flagSLGasPrices      = "dymint.settlement_config.gas_prices"
	flagSLGasFees        = "dymint.settlement_config.gas_fees"
	flagRollappID        = "dymint.settlement_config.rollapp_id"
)

// AddFlags adds Dymint specific configuration options to cobra Command.
//
// This function is called in cosmos-sdk.
func AddNodeFlags(cmd *cobra.Command) {
	//Add tendermint default flags
	tmcmd.AddNodeFlags(cmd)

	def := DefaultNodeConfig

	cmd.Flags().Bool(flagAggregator, false, "run node in aggregator mode")
	cmd.Flags().String(flagDALayer, def.DALayer, "Data Availability Layer Client name (mock or grpc")
	cmd.Flags().String(flagDAConfig, def.DAConfig, "Data Availability Layer Client config")
	cmd.Flags().Duration(flagBlockTime, def.BlockTime, "block time (for aggregator mode)")
	cmd.Flags().Duration(flagEmptyBlocksMaxTime, def.EmptyBlocksMaxTime, "max time for empty blocks (for aggregator mode)")
	cmd.Flags().Duration(flagBatchSubmitMaxTime, def.BatchSubmitMaxTime, "max time for batch submit (for aggregator mode)")
	cmd.Flags().String(flagNamespaceID, def.NamespaceID, "namespace identifies (8 bytes in hex)")
	cmd.Flags().Uint64(flagBlockBatchSize, def.BlockBatchSize, "block batch size")
	cmd.Flags().Uint64(flagBlockBatchMaxSizeBytes, def.BlockBatchMaxSizeBytes, "block batch size in bytes")

	cmd.Flags().String(flagSettlementLayer, def.SettlementLayer, "Settlement Layer Client name")
	cmd.Flags().String(flagSLNodeAddress, def.SettlementConfig.NodeAddress, "Settlement Layer RPC node address")
	cmd.Flags().String(flagSLKeyringBackend, def.SettlementConfig.KeyringBackend, "Sequencer keyring backend")
	cmd.Flags().String(flagSLKeyringHomeDir, def.SettlementConfig.KeyringHomeDir, "Sequencer keyring path")
	cmd.Flags().String(flagSLDymAccountName, def.SettlementConfig.DymAccountName, "Sequencer account name in keyring")
	cmd.Flags().String(flagSLGasFees, def.SettlementConfig.GasFees, "Settlement Layer gas fees")
	cmd.Flags().String(flagSLGasPrices, def.SettlementConfig.GasPrices, "Settlement Layer gas prices")
	cmd.Flags().Uint64(flagSLGasLimit, def.SettlementConfig.GasLimit, "Settlement Layer batch submit gas limit")
	cmd.Flags().String(flagRollappID, def.SettlementConfig.RollappID, "The chainID of the rollapp")

	//simulate fraud
	cmd.Flags().Bool(flagSimulateWrongCommitmentFraud, false, "simulate wrong commitment fraund")
	cmd.Flags().Bool(flagSimulateNonInclusionFraud, false, "simulate non-inclusion fraud")
	cmd.Flags().Bool(flagEnableInclusionProof, false, "enable inclusion proof generation")
}

func BindDymintFlags(cmd *cobra.Command, v *viper.Viper) error {
	if err := v.BindPFlag("aggregator", cmd.Flags().Lookup(flagAggregator)); err != nil {
		return err
	}
	if err := v.BindPFlag("da_layer", cmd.Flags().Lookup(flagDALayer)); err != nil {
		return err
	}
	if err := v.BindPFlag("da_config", cmd.Flags().Lookup(flagDAConfig)); err != nil {
		return err
	}
	if err := v.BindPFlag("block_time", cmd.Flags().Lookup(flagBlockTime)); err != nil {
		return err
	}
	if err := v.BindPFlag("empty_blocks_max_time", cmd.Flags().Lookup(flagEmptyBlocksMaxTime)); err != nil {
		return err
	}
	if err := v.BindPFlag("batch_submit_max_time", cmd.Flags().Lookup(flagBatchSubmitMaxTime)); err != nil {
		return err
	}
	if err := v.BindPFlag("namespace_id", cmd.Flags().Lookup(flagNamespaceID)); err != nil {
		return err
	}
	if err := v.BindPFlag("block_batch_size", cmd.Flags().Lookup(flagBlockBatchSize)); err != nil {
		return err
	}
	if err := v.BindPFlag("block_batch_max_size_bytes", cmd.Flags().Lookup(flagBlockBatchMaxSizeBytes)); err != nil {
		return err
	}
	if err := v.BindPFlag("settlement_layer", cmd.Flags().Lookup(flagSettlementLayer)); err != nil {
		return err
	}
	if err := v.BindPFlag("node_address", cmd.Flags().Lookup(flagSLNodeAddress)); err != nil {
		return err
	}
	if err := v.BindPFlag("keyring_backend", cmd.Flags().Lookup(flagSLKeyringBackend)); err != nil {
		return err
	}
	if err := v.BindPFlag("keyring_home_dir", cmd.Flags().Lookup(flagSLKeyringHomeDir)); err != nil {
		return err
	}
	if err := v.BindPFlag("dym_account_name", cmd.Flags().Lookup(flagSLDymAccountName)); err != nil {
		return err
	}
	if err := v.BindPFlag("gas_fees", cmd.Flags().Lookup(flagSLGasFees)); err != nil {
		return err
	}
	if err := v.BindPFlag("gas_prices", cmd.Flags().Lookup(flagSLGasPrices)); err != nil {
		return err
	}
	if err := v.BindPFlag("gas_limit", cmd.Flags().Lookup(flagSLGasLimit)); err != nil {
		return err
	}
	if err := v.BindPFlag("rollapp_id", cmd.Flags().Lookup(flagRollappID)); err != nil {
		return err
	}
	if err := v.BindPFlag("simulate_noninclusion_fraud", cmd.Flags().Lookup(flagSimulateNonInclusionFraud)); err != nil {
		return err
	}
	if err := v.BindPFlag("simulate_wrongcommitment_fraud", cmd.Flags().Lookup(flagSimulateWrongCommitmentFraud)); err != nil {
		return err
	}
	if err := v.BindPFlag("enable_inclusion_proof", cmd.Flags().Lookup(flagEnableInclusionProof)); err != nil {
		return err
	}
	return nil
}
