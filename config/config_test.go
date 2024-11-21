package config_test

import (
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/dymensionxyz/dymint/da/grpc"
	"github.com/dymensionxyz/dymint/settlement"
)

func TestViperAndCobra(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	cmd := &cobra.Command{}
	config.AddNodeFlags(cmd)

	dir := t.TempDir()
	nc := config.DefaultConfig("")
	config.EnsureRoot(dir, nc)

	assert.NoError(cmd.Flags().Set(config.FlagDAConfig, `{"json":true}`))
	assert.NoError(cmd.Flags().Set(config.FlagBlockTime, "4s"))
	assert.NoError(cmd.Flags().Set(config.FlagMaxIdleTime, "2000s"))
	assert.NoError(cmd.Flags().Set(config.FlagBatchSubmitTime, "3000s"))
	assert.NoError(cmd.Flags().Set(config.FlagBatchSubmitBytes, "1000"))

	assert.NoError(nc.GetViperConfig(cmd, dir))

	assert.Equal(`{"json":true}`, nc.DAConfig)
	assert.Equal(4*time.Second, nc.BlockTime)
	assert.Equal(uint64(1000), nc.BlockManagerConfig.BatchSubmitBytes)
}

func TestNodeConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		malleate func(*config.NodeConfig)

		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "full config",
			wantErr: assert.NoError,
		}, {
			name: "missing block time",
			malleate: func(nc *config.NodeConfig) {
				nc.BlockTime = 0
			},
			wantErr: assert.Error,
		}, {
			name: "missing empty blocks max time",
			malleate: func(nc *config.NodeConfig) {
				nc.MaxIdleTime = -1
			},
			wantErr: assert.Error,
		}, {
			name: "missing batch submit max time",
			malleate: func(nc *config.NodeConfig) {
				nc.BlockManagerConfig.BatchSubmitTime = 0
			},
			wantErr: assert.Error,
		}, {
			name: "block_time too small",
			malleate: func(nc *config.NodeConfig) {
				nc.BlockManagerConfig.BlockTime = 10 * time.Millisecond
			},
			wantErr: assert.Error,
		}, {
			name: "block_time greater than limit",
			malleate: func(nc *config.NodeConfig) {
				nc.BlockManagerConfig.BlockTime = 10 * time.Second
			},
			wantErr: assert.Error,
		}, {
			name: "max_idle_time not greater than block_time",
			malleate: func(nc *config.NodeConfig) {
				nc.BlockManagerConfig.MaxIdleTime = 1
				nc.BlockManagerConfig.MaxProofTime = 1
				nc.BlockManagerConfig.BlockTime = 2
			},
			wantErr: assert.Error,
		}, {
			name: "batch_submit_time not greater than block_time",
			malleate: func(nc *config.NodeConfig) {
				nc.BlockManagerConfig.BatchSubmitTime = 1
				nc.BlockManagerConfig.BlockTime = 2
			},
			wantErr: assert.Error,
		}, {
			name: "batch_submit_time greater than 1 hour",
			malleate: func(nc *config.NodeConfig) {
				nc.BlockManagerConfig.BatchSubmitTime = 2 * time.Hour
			},
			wantErr: assert.Error,
		}, {
			name: "max_skew_time 0",
			malleate: func(nc *config.NodeConfig) {
				nc.BlockManagerConfig.MaxSkewTime = 0
			},
			wantErr: assert.Error,
		}, {
			name: "missing block batch max size bytes",
			malleate: func(nc *config.NodeConfig) {
				nc.BlockManagerConfig.BatchSubmitBytes = 0
			},
			wantErr: assert.Error,
		}, {
			name: "empty settlement layer",
			malleate: func(nc *config.NodeConfig) {
				nc.SettlementLayer = ""
			},
			wantErr: assert.Error,
		}, {
			name: "settlement: provide both fees and gas prices",
			malleate: func(nc *config.NodeConfig) {
				nc.SettlementConfig.GasPrices = "1"
				nc.SettlementConfig.GasFees = "1"
			},
			wantErr: assert.Error,
		}, {
			name: "settlement: provide neither fees nor gas prices",
			malleate: func(nc *config.NodeConfig) {
				nc.SettlementConfig.GasPrices = ""
				nc.SettlementConfig.GasFees = ""
			},
			wantErr: assert.Error,
		}, {
			name: "settlement: mock",
			malleate: func(nc *config.NodeConfig) {
				nc.SettlementLayer = "mock"
			},
			wantErr: assert.NoError,
		}, {
			name: "DAGrpc.Host empty",
			malleate: func(nc *config.NodeConfig) {
				nc.DAGrpc.Host = ""
			},
			wantErr: assert.Error,
		}, {
			name: "DAGrpc.Port 0",
			malleate: func(nc *config.NodeConfig) {
				nc.DAGrpc.Port = 0
			},
			wantErr: assert.Error,
		}, {
			name: "instrumentation: missing prometheus listen addr",
			malleate: func(nc *config.NodeConfig) {
				nc.Instrumentation.PrometheusListenAddr = ""
			},
			wantErr: assert.Error,
		}, {
			name: "instrumentation: prometheus enabled, but listen addr empty",
			malleate: func(nc *config.NodeConfig) {
				nc.Instrumentation.Prometheus = true
				nc.Instrumentation.PrometheusListenAddr = ""
			},
			wantErr: assert.Error,
		}, {
			name: "instrumentation: prometheus disabled, listen addr empty",
			malleate: func(nc *config.NodeConfig) {
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
			tt.wantErr(t, nc.Validate(), "Validate()")
		})
	}
}

func fullNodeConfig() config.NodeConfig {
	return config.NodeConfig{
		BlockManagerConfig: config.BlockManagerConfig{
			BlockTime:                  1 * time.Second,
			MaxIdleTime:                20 * time.Second,
			MaxProofTime:               20 * time.Second,
			BatchSubmitTime:            20 * time.Second,
			MaxSkewTime:                24 * 7 * time.Hour,
			BatchSubmitBytes:           10000,
			SequencerSetUpdateInterval: config.DefaultSequencerSetUpdateInterval,
		},
		DAConfig:        "da-config",
		SettlementLayer: "dymension",
		SettlementConfig: settlement.Config{
			KeyringBackend: "test",
			NodeAddress:    "http://localhost:26657",
			KeyringHomeDir: "/tmp/keyring-test",
			DymAccountName: "test",
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
		Instrumentation: &config.InstrumentationConfig{
			Prometheus:           true,
			PrometheusListenAddr: "localhost:9090",
		},
		DAGrpc: grpc.Config{
			Host: "localhost",
			Port: 9090,
		},
		P2PConfig: config.P2PConfig{
			GossipSubCacheSize:           50,
			BootstrapRetryTime:           30 * time.Second,
			BlockSyncRequestIntervalTime: 30 * time.Second,
			ListenAddress:                config.DefaultListenAddress,
			BootstrapNodes:               "",
		},
	}
}
