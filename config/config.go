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
	DefaultDymintDir      = ".dymint"
	DefaultConfigDirName  = "config"
	DefaultConfigFileName = "dymint.toml"
	MinBlockTime          = 200 * time.Millisecond
	MaxBlockTime          = 6 * time.Second
	MaxBatchSubmitTime    = 1 * time.Hour
)

type NodeConfig struct {
	RootDir       string
	DBPath        string
	RPC           RPCConfig
	MempoolConfig tmcfg.MempoolConfig

	BlockManagerConfig `mapstructure:",squash"`
	DAConfig           string                 `mapstructure:"da_config"`
	SettlementLayer    string                 `mapstructure:"settlement_layer"`
	SettlementConfig   settlement.Config      `mapstructure:",squash"`
	Instrumentation    *InstrumentationConfig `mapstructure:"instrumentation"`

	DAGrpc grpc.Config `mapstructure:",squash"`

	P2PConfig `mapstructure:",squash"`

	DBConfig `mapstructure:"db"`
}

type BlockManagerConfig struct {
	BlockTime time.Duration `mapstructure:"block_time"`

	// if zero, always produced empty blocks
	// if non zero, reschedules an empty block after this duration, after each block produced
	MaxIdleTime time.Duration `mapstructure:"max_idle_time"`

	MaxProofTime time.Duration `mapstructure:"max_proof_time"`

	BatchSubmitTime time.Duration `mapstructure:"batch_submit_time"`

	MaxSkewTime time.Duration `mapstructure:"max_skew_time"`

	BatchSubmitBytes uint64 `mapstructure:"batch_submit_bytes"`

	SequencerSetUpdateInterval time.Duration `mapstructure:"sequencer_update_interval"`
}

func (nc *NodeConfig) GetViperConfig(cmd *cobra.Command, homeDir string) error {
	v := viper.GetViper()

	EnsureRoot(homeDir, nil)
	v.SetConfigName("dymint")
	v.AddConfigPath(homeDir)
	v.AddConfigPath(filepath.Join(homeDir, DefaultConfigDirName))

	err := BindDymintFlags(cmd, v)
	if err != nil {
		return err
	}

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

	if err := nc.DBConfig.Validate(); err != nil {
		return fmt.Errorf("db config: %w", err)
	}

	return nil
}

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

type InstrumentationConfig struct {
	Prometheus bool `mapstructure:"prometheus"`

	PrometheusListenAddr string `mapstructure:"prometheus_listen_addr"`
}

func (ic InstrumentationConfig) Validate() error {
	if ic.Prometheus && ic.PrometheusListenAddr == "" {
		return fmt.Errorf("PrometheusListenAddr cannot be empty")
	}

	return nil
}

type DBConfig struct {
	SyncWrites bool `mapstructure:"sync_writes"`

	InMemory bool `mapstructure:"in_memory"`
}

func (dbc DBConfig) Validate() error {
	return nil
}
