package config

import (
	"fmt"
	"path/filepath"
	"slices"
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
	MinBlockTime          = 200 * time.Millisecond
	MaxBlockTime          = 6 * time.Second
	MaxBatchSubmitTime    = 1 * time.Hour
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
	DALayer            []string               `mapstructure:"da_layer"`
	DAConfig           []string               `mapstructure:"da_config"`
	SettlementLayer    string                 `mapstructure:"settlement_layer"`
	SettlementConfig   settlement.Config      `mapstructure:",squash"`
	Instrumentation    *InstrumentationConfig `mapstructure:"instrumentation"`
	// Config params for mock grpc da
	DAGrpc grpc.Config `mapstructure:",squash"`
	// P2P Options
	P2PConfig `mapstructure:",squash"`
	// DB Options
	DBConfig `mapstructure:"db"`
}

// BlockManagerConfig consists of all parameters required by BlockManagerConfig
type BlockManagerConfig struct {
	// BlockTime defines how often new blocks are produced
	BlockTime time.Duration `mapstructure:"block_time"`
	// MaxIdleTime defines how long should block manager wait for new transactions before producing empty block
	MaxIdleTime time.Duration `mapstructure:"max_idle_time"`
	// MaxProofTime defines the max time to be idle, if txs that requires proof were included in last block
	MaxProofTime time.Duration `mapstructure:"max_proof_time"`
	// BatchSubmitTime is how long should block manager wait for before submitting batch
	BatchSubmitTime time.Duration `mapstructure:"batch_submit_time"`
	// MaxSkewTime is the number of batches waiting to be submitted. Block production will be paused if this limit is reached.
	MaxSkewTime time.Duration `mapstructure:"max_skew_time"`
	// The size of the batch of blocks and commits in Bytes. We'll write every batch to the DA and the settlement layer.
	BatchSubmitBytes uint64 `mapstructure:"batch_submit_bytes"`
	// SequencerSetUpdateInterval defines the interval at which to fetch sequencer updates from the settlement layer
	SequencerSetUpdateInterval time.Duration `mapstructure:"sequencer_update_interval"`
	// TEE configuration for attestation submission
	TeeEnabled bool `mapstructure:"tee_enabled"`
	// Actually require a GCP attestation token?
	TeeDry bool `mapstructure:"tee_dry"`
	// TeeSidecarURL is the URL of the TEE sidecar's RPC endpoint
	TeeSidecarURL string `mapstructure:"tee_sidecar_url"`
	// TeeInterval is how often to fetch and submit attestations
	TeeInterval time.Duration `mapstructure:"tee_interval"`
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
		return fmt.Errorf("instrumentation: %w", err)
	}

	if err := nc.DBConfig.Validate(); err != nil {
		return fmt.Errorf("db config: %w", err)
	}

	return nil
}

// Validate BlockManagerConfig
func (c BlockManagerConfig) Validate() error {
	if c.BlockTime < MinBlockTime {
		return fmt.Errorf("block_time cannot be less than %s", MinBlockTime)
	}

	if c.BlockTime > MaxBlockTime {
		return fmt.Errorf("block_time cannot be greater than %s", MaxBlockTime)
	}

	if c.MaxIdleTime < 0 {
		return fmt.Errorf("max_idle_time must be positive or zero to disable")
	}
	// MaxIdleTime zero disables adaptive block production.
	if c.MaxIdleTime != 0 {
		if c.MaxIdleTime <= c.BlockTime || c.MaxIdleTime > MaxBatchSubmitTime {
			return fmt.Errorf("max_idle_time must be greater than block_time and not greater than %s", MaxBatchSubmitTime)
		}
		if c.MaxProofTime <= 0 || c.MaxProofTime > c.MaxIdleTime {
			return fmt.Errorf("max_proof_time must be positive and not greater than max_idle_time")
		}
	}

	if c.BatchSubmitTime < c.MaxIdleTime {
		return fmt.Errorf("batch_submit_time must be greater than max_idle_time")
	}

	if c.BatchSubmitTime > MaxBatchSubmitTime {
		return fmt.Errorf("batch_submit_time must be not greater than %s", MaxBatchSubmitTime)
	}

	if c.BatchSubmitBytes <= 0 {
		return fmt.Errorf("batch_submit_bytes must be positive")
	}

	if c.MaxSkewTime < c.BatchSubmitTime {
		return fmt.Errorf("max_skew_time cannot be less than batch_submit_time. max_skew_time: %s batch_submit_time: %s", c.MaxSkewTime, c.BatchSubmitTime)
	}

	if c.SequencerSetUpdateInterval <= 0 {
		return fmt.Errorf("sequencer_update_interval must be positive")
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
	if slices.Contains(nc.DALayer, "") {
		return fmt.Errorf("DALayer cannot be empty")
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

// DBConfig holds configuration for the database.
type DBConfig struct {
	// SyncWrites makes sure that data is written to disk before returning from a write operation.
	SyncWrites bool `mapstructure:"sync_writes"`
	// InMemory sets the database to run in-memory, without touching the disk.
	InMemory bool `mapstructure:"in_memory"`
	// if zero, default is used
	BadgerCompactors int `mapstructure:"badger_num_compactors"`
}

func (dbc DBConfig) Validate() error {
	return nil
}
