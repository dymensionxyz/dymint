package config

import (
	"path/filepath"
	"time"

	"github.com/dymensionxyz/dymint/da/grpc"
	"github.com/dymensionxyz/dymint/settlement"
)

const (
	// DefaultListenAddress is a default listen address for P2P client.
	DefaultListenAddress = "/ip4/0.0.0.0/tcp/26656"

	DefaultHomeDir = "sequencer_keys"
	DefaultChainID = "dymint-testnet"
)

// DefaultNodeConfig keeps default values of NodeConfig
var DefaultNodeConfig = *DefaultConfig("", "")

// DefaultConfig returns a default configuration for dymint node.
func DefaultConfig(home, chainId string) *NodeConfig {
	cfg := &NodeConfig{
		BlockManagerConfig: BlockManagerConfig{
			BlockTime:          200 * time.Millisecond,
			MaxIdleTime:        3600 * time.Second,
			MaxProofTime:       100 * time.Second,
			BatchSubmitMaxTime: 3600 * time.Second,
			MaxBatchSkew:       20,
			BatchMaxSizeBytes:  500000,
		},
		DALayer:         "mock",
		SettlementLayer: "mock",
		Instrumentation: &InstrumentationConfig{
			Prometheus:           false,
			PrometheusListenAddr: ":2112",
		},
		P2PConfig: P2PConfig{
			GossipSubCacheSize:           50,
			BootstrapRetryTime:           30 * time.Second,
			BlockSyncRequestIntervalTime: 30 * time.Second,
			ListenAddress:                DefaultListenAddress,
			BootstrapNodes:               "",
			AdvertisingEnabled:           true,
			BlockSyncEnabled:             true,
		},
		DBConfig: DBConfig{
			SyncWrites: true,
			InMemory:   false,
		},
	}

	if home == "" {
		home = "/tmp"
	}
	keyringDir := filepath.Join(home, DefaultHomeDir)
	if chainId == "" {
		chainId = DefaultChainID
	}

	// Setting default params for sl grpc mock
	defaultSlGrpcConfig := settlement.GrpcConfig{
		Host:        "127.0.0.1",
		Port:        7981,
		RefreshTime: 1000,
	}

	defaultSLconfig := settlement.Config{
		KeyringBackend:          "test",
		NodeAddress:             "http://127.0.0.1:36657",
		RollappID:               chainId,
		KeyringHomeDir:          keyringDir,
		DymAccountName:          "sequencer",
		GasPrices:               "1000000000adym",
		SLGrpc:                  defaultSlGrpcConfig,
		RetryMinDelay:           1 * time.Second,
		RetryMaxDelay:           10 * time.Second,
		BatchAcceptanceTimeout:  120 * time.Second,
		BatchAcceptanceAttempts: 5,
		RetryAttempts:           10,
	}
	cfg.SettlementConfig = defaultSLconfig

	// Setting default params for da grpc mock
	defaultDAGrpc := grpc.Config{
		Host: "127.0.0.1",
		Port: 7980,
	}
	cfg.DAGrpc = defaultDAGrpc

	return cfg
}
