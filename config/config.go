package config

import (
	"path/filepath"
	"time"

	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/libs/cli"

	"github.com/dymensionxyz/dymint/settlement"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// DefaultDymintDir is the default directory for dymint
	DefaultDymintDir      = ".dymint"
	DefaultConfigDirName  = "config"
	DefaultConfigFileName = "dymint.toml"
)

const (
	flagAggregator          = "dymint.aggregator"
	flagDALayer             = "dymint.da_layer"
	flagDAConfig            = "dymint.da_config"
	flagBlockTime           = "dymint.block_time"
	flagEmptyBlocksMaxTime  = "dymint.empty_blocks_max_time"
	flagBatchSubmitMaxTime  = "dymint.batch_submit_max_time"
	flagDAStartHeight       = "dymint.da_start_height"
	flagNamespaceID         = "dymint.namespace_id"
	flagBlockBatchSize      = "dymint.block_batch_size"
	flagBlockBatchSizeBytes = "dymint.block_batch_size_bytes"
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
)

// NodeConfig stores Dymint node configuration.
type NodeConfig struct {
	// parameters below are translated from existing config
	RootDir string
	DBPath  string
	P2P     P2PConfig
	RPC     RPCConfig
	// parameters below are dymint specific and read from config
	Aggregator         bool `mapstructure:"aggregator"`
	BlockManagerConfig `mapstructure:",squash"`
	DALayer            string            `mapstructure:"da_layer"`
	DAConfig           string            `mapstructure:"da_config"`
	SettlementLayer    string            `mapstructure:"settlement_layer"`
	SettlementConfig   settlement.Config `mapstructure:",squash"`
}

// BlockManagerConfig consists of all parameters required by BlockManagerConfig
type BlockManagerConfig struct {
	// BlockTime defines how often new blocks are produced
	BlockTime time.Duration `mapstructure:"block_time"`
	// EmptyBlocksMaxTime defines how long should block manager wait for new transactions before producing empty block
	EmptyBlocksMaxTime time.Duration `mapstructure:"empty_blocks_max_time"`
	// BatchSubmitMaxTime defines how long should block manager wait for before submitting batch
	BatchSubmitMaxTime time.Duration `mapstructure:"batch_submit_max_time"`
	// DAStartHeight allows skipping first DAStartHeight-1 blocks when querying for blocks.
	DAStartHeight uint64 `mapstructure:"da_start_height"`
	NamespaceID   string `mapstructure:"namespace_id"`
	// The size of the batch in blocks. Every batch we'll write to the DA and the settlement layer.
	BlockBatchSize uint64 `mapstructure:"block_batch_size"`
	// The size of the batch in Bytes. Every batch we'll write to the DA and the settlement layer.
	BlockBatchSizeBytes uint64 `mapstructure:"block_batch_size_bytes"`
}

// GetViperConfig reads configuration parameters from Viper instance.
//
// This method is called in cosmos-sdk.
func (nc *NodeConfig) GetViperConfig(cmd *cobra.Command, v *viper.Viper) error {
	homeDir := v.GetString(cli.HomeFlag)

	//create dymint toml config file
	EnsureRoot(v.GetString(cli.HomeFlag))
	v.SetConfigName("dymint")
	v.AddConfigPath(homeDir)                          // search root directory
	v.AddConfigPath(filepath.Join(homeDir, "config")) // search root directory /config

	// bind flags so we could override config file with flags
	err := BindDymintFlags(cmd, v)
	if err != nil {
		return err
	}

	// Read viper config
	err = v.ReadInConfig()
	if err != nil {
		// ignore not found error, return other errors
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	err = viper.Unmarshal(&nc)
	if err != nil {
		return err
	}

	// nc.Aggregator = v.GetBool(flagAggregator)
	// nc.DALayer = v.GetString(flagDALayer)
	// nc.DAConfig = v.GetString(flagDAConfig)
	// nc.SettlementLayer = v.GetString(flagSettlementLayer)
	// nc.DAStartHeight = v.GetUint64(flagDAStartHeight)
	// nc.BlockTime = v.GetDuration(flagBlockTime)
	// nc.EmptyBlocksMaxTime = v.GetDuration(flagEmptyBlocksMaxTime)
	// nc.BatchSubmitMaxTime = v.GetDuration(flagBatchSubmitMaxTime)
	// nc.BlockBatchSize = v.GetUint64(flagBlockBatchSize)
	// nc.BlockBatchSizeBytes = v.GetUint64(flagBlockBatchSizeBytes)
	// nc.NamespaceID = v.GetString(flagNamespaceID)

	// nc.SettlementConfig.NodeAddress = v.GetString(flagSLNodeAddress)
	// nc.SettlementConfig.KeyringBackend = v.GetString(flagSLKeyringBackend)
	// nc.SettlementConfig.KeyringHomeDir = v.GetString(flagSLKeyringHomeDir)
	// nc.SettlementConfig.DymAccountName = v.GetString(flagSLDymAccountName)
	// nc.SettlementConfig.GasLimit = v.GetUint64(flagSLGasLimit)
	// nc.SettlementConfig.GasPrices = v.GetString(flagSLGasPrices)
	// nc.SettlementConfig.GasFees = v.GetString(flagSLGasFees)

	return nil
}

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
	cmd.Flags().Uint64(flagDAStartHeight, def.DAStartHeight, "starting DA block height (for syncing)")
	cmd.Flags().String(flagNamespaceID, def.NamespaceID, "namespace identifies (8 bytes in hex)")
	cmd.Flags().Uint64(flagBlockBatchSize, def.BlockBatchSize, "block batch size")
	cmd.Flags().Uint64(flagBlockBatchSizeBytes, def.BlockBatchSizeBytes, "block batch size in bytes")

	cmd.Flags().String(flagSettlementLayer, def.SettlementLayer, "Settlement Layer Client name")
	cmd.Flags().String(flagSLNodeAddress, def.SettlementConfig.NodeAddress, "Settlement Layer RPC node address")
	cmd.Flags().String(flagSLKeyringBackend, def.SettlementConfig.KeyringBackend, "Sequencer keyring backend")
	cmd.Flags().String(flagSLKeyringHomeDir, def.SettlementConfig.KeyringHomeDir, "Sequencer keyring path")
	cmd.Flags().String(flagSLDymAccountName, def.SettlementConfig.DymAccountName, "Sequencer account name in keyring")
	cmd.Flags().String(flagSLGasFees, def.SettlementConfig.GasFees, "Settlement Layer gas fees")
	cmd.Flags().String(flagSLGasPrices, def.SettlementConfig.GasPrices, "Settlement Layer gas prices")
	cmd.Flags().Uint64(flagSLGasLimit, def.SettlementConfig.GasLimit, "Settlement Layer batch submit gas limit")
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
	if err := v.BindPFlag("da_start_height", cmd.Flags().Lookup(flagDAStartHeight)); err != nil {
		return err
	}
	if err := v.BindPFlag("namespace_id", cmd.Flags().Lookup(flagNamespaceID)); err != nil {
		return err
	}
	if err := v.BindPFlag("block_batch_size", cmd.Flags().Lookup(flagBlockBatchSize)); err != nil {
		return err
	}
	if err := v.BindPFlag("block_batch_size_bytes", cmd.Flags().Lookup(flagBlockBatchSizeBytes)); err != nil {
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
	if err := v.BindPFlag("keyringhomedir", cmd.Flags().Lookup(flagSLKeyringHomeDir)); err != nil {
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

	return nil
}
