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
			BlockTime:              200 * time.Millisecond,
			MaxIdleTime:            3600 * time.Second,
			MaxProofTime:           100 * time.Second,
			BatchSubmitMaxTime:     3600 * time.Second,
			MaxSupportedBatchSkew:  20,
			NamespaceID:            "0000000000000000ffff",
			BlockBatchMaxSizeBytes: 500000,
		},
		DALayer:         "mock",
		SettlementLayer: "mock",
		Instrumentation: &InstrumentationConfig{
			Prometheus:           false,
			PrometheusListenAddr: ":2112",
		},
		P2PConfig: P2PConfig{
			GossipedBlocksCacheSize: 50,
			BootstrapRetryTime:      30 * time.Second,
			ListenAddress:           DefaultListenAddress,
			BootstrapNodes:          "",
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
		KeyringBackend: "test",
		NodeAddress:    "http://127.0.0.1:36657",
		RollappID:      chainId,
		KeyringHomeDir: keyringDir,
		DymAccountName: "sequencer",
		GasPrices:      "1000000000adym",
		SLGrpc:         defaultSlGrpcConfig,
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
