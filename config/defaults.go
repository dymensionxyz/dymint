package config

import (
	"path/filepath"
	"time"

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
			BlockTime:              200 * time.Millisecond,
			EmptyBlocksMaxTime:     3 * time.Second,
			BatchSubmitMaxTime:     30 * time.Second,
			NamespaceID:            "000000000000ffff",
			BlockBatchSize:         500,
			BlockBatchMaxSizeBytes: 500000},
		DALayer:         "mock",
		SettlementLayer: "mock",
		Instrumentation: &InstrumentationConfig{
			Prometheus:           false,
			PrometheusListenAddr: ":2112",
		},
	}

	if home == "" {
		home = "/tmp"
	}
	keyringDir := filepath.Join(home, DefaultHomeDir)
	if chainId == "" {
		chainId = DefaultChainID
	}

	defaultSLconfig := settlement.Config{
		KeyringBackend: "test",
		NodeAddress:    "http://127.0.0.1:36657",
		RollappID:      chainId,
		KeyringHomeDir: keyringDir,
		DymAccountName: "sequencer",
		GasPrices:      "0.025udym",
	}
	cfg.SettlementConfig = defaultSLconfig

	return cfg
}
