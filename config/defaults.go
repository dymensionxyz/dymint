package config

import "time"

const (
	// DefaultListenAddress is a default listen address for P2P client.
	DefaultListenAddress = "/ip4/0.0.0.0/tcp/7676"
	// Version is a default dymint version for P2P client.
	Version = "0.2.2"
)

// DefaultNodeConfig keeps default values of NodeConfig
var DefaultNodeConfig = NodeConfig{
	P2P: P2PConfig{
		ListenAddress: DefaultListenAddress,
		Seeds:         "",
	},
	Aggregator: true,
	BlockManagerConfig: BlockManagerConfig{
		BlockTime:         200 * time.Millisecond,
		NamespaceID:       [8]byte{},
		BatchSyncInterval: time.Second * 30,
		BlockBatchSize:    500,
	},
	DALayer:         "mock",
	SettlementLayer: "mock",
	DAConfig:        "",
}
