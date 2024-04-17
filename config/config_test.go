package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/dymensionxyz/dymint/da/grpc"
	"github.com/dymensionxyz/dymint/settlement"
)

func TestViperAndCobra(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	cmd := &cobra.Command{}
	AddNodeFlags(cmd)

	dir := t.TempDir()
	nc := DefaultConfig("", "")
	EnsureRoot(dir, nc)

	assert.NoError(cmd.Flags().Set(flagAggregator, "true"))
	assert.NoError(cmd.Flags().Set(flagDALayer, "foobar"))
	assert.NoError(cmd.Flags().Set(flagDAConfig, `{"json":true}`))
	assert.NoError(cmd.Flags().Set(flagBlockTime, "1234s"))
	assert.NoError(cmd.Flags().Set(flagEmptyBlocksMaxTime, "2000s"))
	assert.NoError(cmd.Flags().Set(flagBatchSubmitMaxTime, "3000s"))
	assert.NoError(cmd.Flags().Set(flagNamespaceID, "0102030405060708"))
	assert.NoError(cmd.Flags().Set(flagBlockBatchSize, "10"))

	assert.NoError(nc.GetViperConfig(cmd, dir))

	assert.Equal(true, nc.Aggregator)
	assert.Equal("foobar", nc.DALayer)
	assert.Equal(`{"json":true}`, nc.DAConfig)
	assert.Equal(1234*time.Second, nc.BlockTime)
	assert.Equal("0102030405060708", nc.NamespaceID)
	assert.Equal(uint64(10), nc.BlockBatchSize)
}

func TestNodeConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		malleate func(*NodeConfig)

		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "full config",
			wantErr: assert.NoError,
		}, {
			name: "missing block time",
			malleate: func(nc *NodeConfig) {
				nc.BlockTime = 0
			},
			wantErr: assert.Error,
		}, {
			name: "missing empty blocks max time",
			malleate: func(nc *NodeConfig) {
				nc.EmptyBlocksMaxTime = -1
			},
			wantErr: assert.Error,
		}, {
			name: "missing batch submit max time",
			malleate: func(nc *NodeConfig) {
				nc.BlockManagerConfig.BatchSubmitMaxTime = 0
			},
			wantErr: assert.Error,
		}, {
			name: "empty_blocks_max_time not greater than block_time",
			malleate: func(nc *NodeConfig) {
				nc.BlockManagerConfig.EmptyBlocksMaxTime = 1
				nc.BlockManagerConfig.BlockTime = 2
			},
			wantErr: assert.Error,
		}, {
			name: "batch_submit_max_time not greater than block_time",
			malleate: func(nc *NodeConfig) {
				nc.BlockManagerConfig.BatchSubmitMaxTime = 1
				nc.BlockManagerConfig.BlockTime = 2
			},
			wantErr: assert.Error,
		}, {
			name: "missing block batch size",
			malleate: func(nc *NodeConfig) {
				nc.BlockManagerConfig.BlockBatchSize = 0
			},
			wantErr: assert.Error,
		}, {
			name: "missing block batch max size bytes",
			malleate: func(nc *NodeConfig) {
				nc.BlockManagerConfig.BlockBatchMaxSizeBytes = 0
			},
			wantErr: assert.Error,
		}, {
			name: "missing gossiped blocks cache size",
			malleate: func(nc *NodeConfig) {
				nc.BlockManagerConfig.GossipedBlocksCacheSize = 0
			},
			wantErr: assert.Error,
		}, {
			name: "empty settlement layer",
			malleate: func(nc *NodeConfig) {
				nc.SettlementLayer = ""
			},
			wantErr: assert.Error,
		}, {
			name: "settlement: provide both fees and gas prices",
			malleate: func(nc *NodeConfig) {
				nc.SettlementConfig.GasPrices = "1"
				nc.SettlementConfig.GasFees = "1"
			},
			wantErr: assert.Error,
		}, {
			name: "settlement: provide neither fees nor gas prices",
			malleate: func(nc *NodeConfig) {
				nc.SettlementConfig.GasPrices = ""
				nc.SettlementConfig.GasFees = ""
			},
			wantErr: assert.Error,
		}, {
			name: "settlement: missing rollapp id",
			malleate: func(nc *NodeConfig) {
				nc.SettlementConfig.RollappID = ""
			},
			wantErr: assert.Error,
		}, {
			name: "settlement: mock",
			malleate: func(nc *NodeConfig) {
				nc.SettlementLayer = "mock"
			},
			wantErr: assert.NoError,
		}, {
			name: "DALayer: empty",
			malleate: func(nc *NodeConfig) {
				nc.DALayer = ""
			},
			wantErr: assert.Error,
		}, {
			name: "DALayer: mock",
			malleate: func(nc *NodeConfig) {
				nc.DALayer = "mock"
			},
			wantErr: assert.NoError,
		}, {
			name: "DAConfig: empty",
			malleate: func(nc *NodeConfig) {
				nc.DAConfig = ""
			},
			wantErr: assert.Error,
		}, {
			name: "DAGrpc.Host empty",
			malleate: func(nc *NodeConfig) {
				nc.DAGrpc.Host = ""
			},
			wantErr: assert.Error,
		}, {
			name: "DAGrpc.Port 0",
			malleate: func(nc *NodeConfig) {
				nc.DAGrpc.Port = 0
			},
			wantErr: assert.Error,
		}, {
			name: "instrumentation: missing prometheus listen addr",
			malleate: func(nc *NodeConfig) {
				nc.Instrumentation.PrometheusListenAddr = ""
			},
			wantErr: assert.Error,
		}, {
			name: "instrumentation: prometheus enabled, but listen addr empty",
			malleate: func(nc *NodeConfig) {
				nc.Instrumentation.Prometheus = true
				nc.Instrumentation.PrometheusListenAddr = ""
			},
			wantErr: assert.Error,
		}, {
			name: "instrumentation: prometheus disabled, listen addr empty",
			malleate: func(nc *NodeConfig) {
				nc.Instrumentation.Prometheus = false
				nc.Instrumentation.PrometheusListenAddr = ""
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nc := fullNodeConfig()
			if tt.malleate != nil {
				tt.malleate(&nc)
			}
			tt.wantErr(t, nc.Validate(), fmt.Sprintf("Validate()"))
		})
	}
}

func fullNodeConfig() NodeConfig {
	return NodeConfig{
		BlockManagerConfig: BlockManagerConfig{
			BlockTime:               1 * time.Second,
			EmptyBlocksMaxTime:      2 * time.Second,
			BatchSubmitMaxTime:      1 * time.Second,
			NamespaceID:             "test",
			BlockBatchSize:          1,
			BlockBatchMaxSizeBytes:  1,
			GossipedBlocksCacheSize: 1,
		},
		DALayer:         "celestia",
		DAConfig:        "da-config",
		SettlementLayer: "dymension",
		SettlementConfig: settlement.Config{
			KeyringBackend: "test",
			NodeAddress:    "http://localhost:26657",
			KeyringHomeDir: "/tmp/keyring-test",
			DymAccountName: "test",
			RollappID:      "test_123-1",
			GasLimit:       120,
			GasPrices:      "0.025stake",
			GasFees:        "",
			ProposerPubKey: "test",
			SLGrpc: settlement.GrpcConfig{
				Host:        "localhost",
				Port:        9090,
				RefreshTime: 1,
			},
		},
		Instrumentation: &InstrumentationConfig{
			Prometheus:           true,
			PrometheusListenAddr: "localhost:9090",
		},
		DAGrpc: grpc.Config{
			Host: "localhost",
			Port: 9090,
		},
	}
}
