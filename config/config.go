package config

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dymensionxyz/dymint/da/grpc"
	"github.com/dymensionxyz/dymint/settlement"
	tmcfg "github.com/tendermint/tendermint/config"
)

const (
	// DefaultDymintDir is the default directory for dymint
	DefaultDymintDir      = ".dymint"
	DefaultConfigDirName  = "config"
	DefaultConfigFileName = "dymint.toml"
)

// NodeConfig stores Dymint node configuration.
type NodeConfig struct {
	// parameters below are translated from existing config
	RootDir       string
	DBPath        string
	RPC           RPCConfig
	MempoolConfig tmcfg.MempoolConfig

	// parameters below are dymint specific and read from config
	BlockManagerConfig `mapstructure:",squash"`
	DALayer            string                 `mapstructure:"da_layer"`
	DAConfig           string                 `mapstructure:"da_config"`
	SettlementLayer    string                 `mapstructure:"settlement_layer"`
	SettlementConfig   settlement.Config      `mapstructure:",squash"`
	Instrumentation    *InstrumentationConfig `mapstructure:"instrumentation"`
	// Config params for mock grpc da
	DAGrpc grpc.Config `mapstructure:",squash"`
	// P2P Options
	P2PConfig `mapstructure:",squash"`
}

// BlockManagerConfig consists of all parameters required by BlockManagerConfig
type BlockManagerConfig struct {
	// BlockTime defines how often new blocks are produced
	BlockTime time.Duration `mapstructure:"block_time"`
	// MaxIdleTime defines how long should block manager wait for new transactions before producing empty block
	MaxIdleTime time.Duration `mapstructure:"max_idle_time"`
	// MaxProofTime defines the max time to be idle, if txs that requires proof were included in last block
	MaxProofTime time.Duration `mapstructure:"max_proof_time"`
	// BatchSubmitMaxTime defines how long should block manager wait for before submitting batch
	BatchSubmitMaxTime time.Duration `mapstructure:"batch_submit_max_time"`
	// Max amount of pending batches to be submitted. block production will be paused if this limit is reached.
	MaxSupportedBatchSkew uint64 `mapstructure:"max_supported_batch_skew"`
	// The size of the batch in Bytes. Every batch we'll write to the DA and the settlement layer.
	BlockBatchMaxSizeBytes uint64 `mapstructure:"block_batch_max_size_bytes"`
}

// GetViperConfig reads configuration parameters from Viper instance.
func (nc *NodeConfig) GetViperConfig(cmd *cobra.Command, homeDir string) error {
	v := viper.GetViper()

	// Loads dymint toml config file
	EnsureRoot(homeDir, nil)
	v.SetConfigName("dymint")
	v.AddConfigPath(homeDir)                                      // search root directory
	v.AddConfigPath(filepath.Join(homeDir, DefaultConfigDirName)) // search root directory /config

	// bind flags so we could override config file with flags
	err := BindDymintFlags(cmd, v)
	if err != nil {
		return err
	}

	// Read viper config
	err = v.ReadInConfig()
	if err != nil {
		return err
	}

	err = viper.Unmarshal(&nc)
	if err != nil {
		return err
	}

	err = nc.Validate()
	if err != nil {
		return err
	}

	return nil
}

func (nc NodeConfig) Validate() error {
	if err := nc.BlockManagerConfig.Validate(); err != nil {
		return fmt.Errorf("BlockManagerConfig: %w", err)
	}

	if err := nc.P2PConfig.Validate(); err != nil {
		return fmt.Errorf("p2p config: %w", err)
	}

	if err := nc.validateSettlementLayer(); err != nil {
		return fmt.Errorf("SettlementLayer: %w", err)
	}

	if err := nc.validateDALayer(); err != nil {
		return fmt.Errorf("DALayer: %w", err)
	}

	if err := nc.validateInstrumentation(); err != nil {
		return fmt.Errorf("Instrumentation: %w", err)
	}

	return nil
}

// Validate BlockManagerConfig
func (c BlockManagerConfig) Validate() error {
	if c.BlockTime <= 0 {
		return fmt.Errorf("block_time must be positive")
	}

	if c.MaxIdleTime < 0 {
		return fmt.Errorf("max_idle_time must be positive or zero to disable")
	}
	// MaxIdleTime zero disables adaptive block production.
	if c.MaxIdleTime != 0 {
		if c.MaxIdleTime <= c.BlockTime {
			return fmt.Errorf("max_idle_time must be greater than block_time")
		}
		if c.MaxProofTime <= 0 || c.MaxProofTime > c.MaxIdleTime {
			return fmt.Errorf("max_proof_time must be positive and not greater than max_idle_time")
		}
	}

	if c.BatchSubmitMaxTime <= 0 {
		return fmt.Errorf("batch_submit_max_time must be positive")
	}

	if c.BatchSubmitMaxTime < c.BlockTime {
		return fmt.Errorf("batch_submit_max_time must be greater than block_time")
	}

	if c.BatchSubmitMaxTime < c.MaxIdleTime {
		return fmt.Errorf("batch_submit_max_time must be greater than max_idle_time")
	}

	if c.BlockBatchMaxSizeBytes <= 0 {
		return fmt.Errorf("block_batch_size_bytes must be positive")
	}

	if c.MaxSupportedBatchSkew <= 0 {
		return fmt.Errorf("max_supported_batch_skew must be positive")
	}

	return nil
}

func (nc NodeConfig) validateSettlementLayer() error {
	if nc.SettlementLayer == "" {
		return fmt.Errorf("SettlementLayer cannot be empty")
	}

	if nc.SettlementLayer == "mock" {
		return nil
	}

	return nc.SettlementConfig.Validate()
}

func (nc NodeConfig) validateDALayer() error {
	if nc.DALayer == "" {
		return fmt.Errorf("DALayer cannot be empty")
	}

	if nc.DALayer == "mock" {
		return nil
	}

	if nc.DAConfig == "" {
		return fmt.Errorf("DAConfig cannot be empty")
	}
	if nc.DAGrpc.Host == "" {
		return fmt.Errorf("DAGrpc.Host cannot be empty")
	}
	if nc.DAGrpc.Port == 0 {
		return fmt.Errorf("DAGrpc.Port cannot be 0")
	}

	return nil
}

func (nc NodeConfig) validateInstrumentation() error {
	if nc.Instrumentation == nil {
		return nil
	}

	return nc.Instrumentation.Validate()
}

// InstrumentationConfig defines the configuration for metrics reporting.
type InstrumentationConfig struct {
	// When true, Prometheus metrics are served under /metrics on
	// PrometheusListenAddr.
	// Check out the documentation for the list of available metrics.
	Prometheus bool `mapstructure:"prometheus"`

	// Address to listen for Prometheus collector(s) connections.
	PrometheusListenAddr string `mapstructure:"prometheus_listen_addr"`
}

func (ic InstrumentationConfig) Validate() error {
	if ic.Prometheus && ic.PrometheusListenAddr == "" {
		return fmt.Errorf("PrometheusListenAddr cannot be empty")
	}

	return nil
}
