package config

import (
	"encoding/hex"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagAggregator        = "dymint.aggregator"
	flagDALayer           = "dymint.da_layer"
	flagDAConfig          = "dymint.da_config"
	flagSettlementLayer   = "dymint.settlement_layer"
	flagSettlementConfig  = "dymint.settlement_config"
	flagBlockTime         = "dymint.block_time"
	flagDABlockTime       = "dymint.da_block_time"
	flagBatchSyncInterval = "dymint.batch_sync_interval"
	flagDAStartHeight     = "dymint.da_start_height"
	flagNamespaceID       = "dymint.namespace_id"
	flagBlockBatchSize    = "dymint.block_batch_size"
)

var (
	// DefaultDymintDir is the default directory for dymint
	DefaultDymintDir = ".dymint"
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
	DALayer            string `mapstructure:"da_layer"`
	DAConfig           string `mapstructure:"da_config"`
	SettlementLayer    string `mapstructure:"settlement_layer"`
	SettlementConfig   string `mapstructure:"settlement_config"`
}

// BlockManagerConfig consists of all parameters required by BlockManagerConfig
type BlockManagerConfig struct {
	// BlockTime defines how often new blocks are produced
	BlockTime time.Duration `mapstructure:"block_time"`
	// DABlockTime informs about block time of underlying data availability layer
	DABlockTime time.Duration `mapstructure:"da_block_time"`
	// BatchSyncInterval defines how often block manager should sync with the settlement layer
	BatchSyncInterval time.Duration `mapstructure:"batch_sync_interval"`
	// DAStartHeight allows skipping first DAStartHeight-1 blocks when querying for blocks.
	DAStartHeight uint64  `mapstructure:"da_start_height"`
	NamespaceID   [8]byte `mapstructure:"namespace_id"`
	// The size of the batch in blocks. Every batch we'll write to the DA and the settlement layer.
	BlockBatchSize uint64 `mapstructure:"block_batch_size"`
}

// GetViperConfig reads configuration parameters from Viper instance.
//
// This method is called in cosmos-sdk.
func (nc *NodeConfig) GetViperConfig(v *viper.Viper) error {
	nc.Aggregator = v.GetBool(flagAggregator)
	nc.DALayer = v.GetString(flagDALayer)
	nc.DAConfig = v.GetString(flagDAConfig)
	nc.SettlementLayer = v.GetString(flagSettlementLayer)
	nc.SettlementConfig = v.GetString(flagSettlementConfig)
	nc.DAStartHeight = v.GetUint64(flagDAStartHeight)
	nc.DABlockTime = v.GetDuration(flagDABlockTime)
	nc.BatchSyncInterval = v.GetDuration(flagBatchSyncInterval)
	nc.BlockTime = v.GetDuration(flagBlockTime)
	nc.BlockBatchSize = v.GetUint64(flagBlockBatchSize)
	nsID := v.GetString(flagNamespaceID)
	bytes, err := hex.DecodeString(nsID)
	if err != nil {
		return err
	}
	copy(nc.NamespaceID[:], bytes)
	return nil
}

// AddFlags adds Dymint specific configuration options to cobra Command.
//
// This function is called in cosmos-sdk.
func AddFlags(cmd *cobra.Command) {
	def := DefaultNodeConfig

	cmd.Flags().Bool(flagAggregator, false, "run node in aggregator mode")
	cmd.Flags().String(flagDALayer, def.DALayer, "Data Availability Layer Client name (mock or grpc")
	cmd.Flags().String(flagDAConfig, def.DAConfig, "Data Availability Layer Client config")
	cmd.Flags().String(flagSettlementLayer, def.SettlementLayer, "Settlement Layer Client name (currently only mock)")
	cmd.Flags().String(flagSettlementConfig, def.SettlementConfig, "Settlement Layer Client config")
	cmd.Flags().Duration(flagBlockTime, def.BlockTime, "block time (for aggregator mode)")
	cmd.Flags().Duration(flagDABlockTime, def.DABlockTime, "DA chain block time (for syncing)")
	cmd.Flags().Duration(flagBatchSyncInterval, def.BatchSyncInterval, "batch sync interval")
	cmd.Flags().Uint64(flagDAStartHeight, def.DAStartHeight, "starting DA block height (for syncing)")
	cmd.Flags().BytesHex(flagNamespaceID, def.NamespaceID[:], "namespace identifies (8 bytes in hex)")
	cmd.Flags().Uint64(flagBlockBatchSize, def.BlockBatchSize, "block batch size")
}
