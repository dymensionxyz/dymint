package config

import (
	"time"

	"github.com/dymensionxyz/dymint/settlement"
)

const (
	// DefaultListenAddress is a default listen address for P2P client.
	DefaultListenAddress = "/ip4/0.0.0.0/tcp/7676"
	// Version is a default dymint version for P2P client.
	Version = "0.2.2"
)

// DefaultNodeConfig keeps default values of NodeConfig
var DefaultNodeConfig = NodeConfig{
	RootDir:            "",
	DBPath:             "",
	P2P:                P2PConfig{ListenAddress: DefaultListenAddress, Seeds: ""},
	RPC:                RPCConfig{},
	Aggregator:         true,
	BlockManagerConfig: BlockManagerConfig{BlockTime: 200 * time.Millisecond, NamespaceID: "000000000000ffff", BatchSyncInterval: time.Second * 30, DABlockTime: 30 * time.Second, BlockBatchSize: 500, BlockBatchSizeBytes: 1500000},
	DALayer:            "mock",
	DAConfig:           "",
	SettlementLayer:    "mock",
	SettlementConfig: settlement.Config{
		KeyringBackend: "test",
		NodeAddress:    "",
		KeyRingHomeDir: "",
		DymAccountName: "",
		RollappID:      "",
		GasLimit:       0,
		GasPrices:      "",
		GasFees:        "",
	},
}

//FIXME: default config with mocks should be enoguh to run
