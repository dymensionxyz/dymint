package config

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	tmos "github.com/tendermint/tendermint/libs/os"
)

// DefaultDirPerm is the default permissions used when creating directories.
const DefaultDirPerm = 0o700

var configTemplate *template.Template

func init() {
	var err error
	tmpl := template.New("configFileTemplate").Funcs(template.FuncMap{
		"StringsJoin": strings.Join,
	})
	if configTemplate, err = tmpl.Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

/****** these are for production settings ***********/

// EnsureRoot creates the root, config, and data directories if they don't exist,
// and panics if it fails.
func EnsureRoot(rootDir string, defaultConfig *NodeConfig) {
	if err := tmos.EnsureDir(rootDir, DefaultDirPerm); err != nil {
		panic(err.Error())
	}
	if err := tmos.EnsureDir(filepath.Join(rootDir, DefaultConfigDirName), DefaultDirPerm); err != nil {
		panic(err.Error())
	}

	if defaultConfig == nil {
		return
	}

	configFilePath := filepath.Join(rootDir, DefaultConfigDirName, DefaultConfigFileName)

	// Write default config file if missing.
	if !tmos.FileExists(configFilePath) {
		WriteConfigFile(configFilePath, defaultConfig)
	}
}

// WriteConfigFile renders config using the template and writes it to configFilePath.
func WriteConfigFile(configFilePath string, config *NodeConfig) {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	tmos.MustWriteFile(configFilePath, buffer.Bytes(), 0o644)
}

// Note: any changes to the comments/variables/mapstructure
// must be reflected in the appropriate struct in config/config.go
const defaultConfigTemplate = `
#######################################################
###       Dymint Configuration Options     ###
#######################################################
# block production interval
block_time = "{{ .BlockManagerConfig.BlockTime }}"
# block production interval in case of no transactions ("0s" produces empty blocks)
max_idle_time = "{{ .BlockManagerConfig.MaxIdleTime }}"
max_proof_time = "{{ .BlockManagerConfig.MaxProofTime }}"
max_supported_batch_skew = {{ .BlockManagerConfig.MaxSupportedBatchSkew }}


# triggers to submit batch to DA and settlement (both required)
batch_submit_max_time = "{{ .BlockManagerConfig.BatchSubmitMaxTime }}"

# max size of batch in bytes that can be accepted by DA
block_batch_max_size_bytes = {{ .BlockManagerConfig.BlockBatchMaxSizeBytes }}

### da config ###
da_layer = "{{ .DALayer }}" # mock, celestia, avail
namespace_id = "{{ .BlockManagerConfig.NamespaceID }}"
# this should be json matching the celestia.Config type
da_config = "{{ .DAConfig }}"


### p2p config ###

# p2p listen address in the format of /ip4/ip_address/tcp/tcp_port
p2p_listen_address = "{{ .P2PConfig.ListenAddress }}"

# list of nodes used for P2P bootstrapping in the format of /ip4/ip_address/tcp/port/p2p/ID
p2p_bootstrap_nodes = "{{ .P2PConfig.BootstrapNodes }}"

# max number of cached messages by gossipsub protocol
p2p_gossiped_blocks_cache_size = {{ .P2PConfig.GossipedBlocksCacheSize }}

# time interval to check if no p2p nodes are connected to bootstrap again
p2p_bootstrap_retry_time = "{{ .P2PConfig.BootstrapRetryTime }}"

# set to false to disable advertising the node to the P2P network
p2p_advertising_enabled= "{{ .P2PConfig.AdvertisingEnabled }}"

#celestia config example:
# da_config = "{\"base_url\":\"http:\/\/127.0.0.1:26658\",\"timeout\":5000000000,\"gas_prices\":0.1,\"auth_token\":\"TOKEN\",\"backoff\":{\"initial_delay\":6000000000,\"max_delay\":6000000000,\"growth_factor\":2},\"retry_attempts\":4,\"retry_delay\":3000000000}"
# Avail config example:
# da_config = "{\"seed\": \"MNEMONIC\", \"api_url\": \"wss://kate.avail.tools/ws\", \"app_id\": 0, \"tip\":10}"

### settlement config ###
settlement_layer = "{{ .SettlementLayer }}" # mock, dymension

# dymension config
rollapp_id = "{{ .SettlementConfig.RollappID }}"
node_address = "{{ .SettlementConfig.NodeAddress }}"
gas_limit = {{ .SettlementConfig.GasLimit }}
gas_prices = "{{ .SettlementConfig.GasPrices }}"
gas_fees = "{{ .SettlementConfig.GasFees }}"
retry_max_delay = "{{ .SettlementConfig.RetryMaxDelay }}"
retry_min_delay = "{{ .SettlementConfig.RetryMinDelay }}"
retry_attempts = "{{ .SettlementConfig.RetryAttempts }}"
batch_acceptance_timeout = "{{ .SettlementConfig.BatchAcceptanceTimeout }}"

#keyring and key name to be used for sequencer 
keyring_backend = "{{ .SettlementConfig.KeyringBackend }}"
keyring_home_dir = "{{ .SettlementConfig.KeyringHomeDir }}"
dym_account_name = "{{ .SettlementConfig.DymAccountName }}"


#######################################################
###       Instrumentation Configuration Options     ###
#######################################################
[instrumentation]

# When true, Prometheus metrics are served under /metrics on
# PrometheusListenAddr.
# Check out the documentation for the list of available metrics.
prometheus = {{ .Instrumentation.Prometheus }}

# Address to listen for Prometheus collector(s) connections
prometheus_listen_addr = "{{ .Instrumentation.PrometheusListenAddr }}"


`
