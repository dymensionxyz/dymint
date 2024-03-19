package config

import (
	"path/filepath"
	"time"

	"github.com/dymensionxyz/dymint/da/grpc"
	"github.com/dymensionxyz/dymint/settlement"
)

const (
	// DefaultListenAddress is a default listen address for P2P client.
	DefaultListenAddress = "/ip4/0.0.0.0/tcp/7676"

	DefaultHomeDir = "sequencer_keys"
	DefaultChainID = "dymint-testnet"
)

// DefaultNodeConfig keeps default values of NodeConfig
var DefaultNodeConfig = *DefaultConfig("", "")

// DefaultConfig returns a default configuration for dymint node.
func DefaultConfig(home, chainId string) *NodeConfig {
	cfg := &NodeConfig{
		P2P: P2PConfig{
			ListenAddress: DefaultListenAddress,
			Seeds:         ""},
		Aggregator: true,
		BlockManagerConfig: BlockManagerConfig{
			BlockTime:               200 * time.Millisecond,
			EmptyBlocksMaxTime:      3 * time.Second,
			BatchSubmitMaxTime:      100 * time.Second,
			NamespaceID:             "0000000000000000ffff",
			BlockBatchSize:          500,
			BlockBatchMaxSizeBytes:  500000,
			GossipedBlocksCacheSize: 50},
		DALayer:         "mock",
		SettlementLayer: "mock",
		Instrumentation: &InstrumentationConfig{
			Prometheus:           false,
			PrometheusListenAddr: ":2112",
		},
		BootstrapTime: 30 * time.Second,
	}

	if home == "" {
		home = "/tmp"
	}
	keyringDir := filepath.Join(home, DefaultHomeDir)
	if chainId == "" {
		chainId = DefaultChainID
	}

	//Setting default params for sl grpc mock
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
		GasPrices:      "0.025udym",
		SLGrpc:         defaultSlGrpcConfig,
	}
	cfg.SettlementConfig = defaultSLconfig

	// set operator address
	cfg.OperatorKeyringBackend = defaultSLconfig.KeyringBackend
	cfg.OperatorKeyringHomeDir = defaultSLconfig.KeyringHomeDir
	cfg.OperatorAccountName = defaultSLconfig.DymAccountName

	//Setting default params for da grpc mock
	defaultDAGrpc := grpc.Config{
		Host: "127.0.0.1",
		Port: 7980,
	}
	cfg.DAGrpc = defaultDAGrpc

	return cfg
}
