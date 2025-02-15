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
# block production interval after block with no transactions
max_proof_time = "{{ .BlockManagerConfig.MaxProofTime }}"
# maximum time the node will produce blocks without submitting to SL before stopping block production
max_skew_time = "{{ .BlockManagerConfig.MaxSkewTime }}"


# triggers to submit batch to DA and settlement (both required)
# max time between two batch submissions. submission will be triggered if there is no previous submission in batch_submit_time
batch_submit_time = "{{ .BlockManagerConfig.BatchSubmitTime }}"

# max size of batch in bytes. submission will be triggered after accumulating blocks for batch_submit_bytes
batch_submit_bytes = {{ .BlockManagerConfig.BatchSubmitBytes }}

### da config ###
# this should be json matching the celestia.Config type
da_config = "{{ .DAConfig }}"

# Celestia config example:
# da_config = "{\"base_url\":\"http:\/\/127.0.0.1:26658\",\"timeout\":30000000000,\"gas_prices\":0.1,\"auth_token\":\"TOKEN\",\"backoff\":{\"initial_delay\":6000000000,\"max_delay\":6000000000,\"growth_factor\":2},\"retry_attempts\":4,\"retry_delay\":3000000000}"
# Avail config example:
# da_config = "{\"seed\": \"MNEMONIC\", \"endpoint\": \"https://turing-rpc.avail.so/rpc\", \"app_id\": 0}"
# WeaveVM config example:
# da_config = "{\"endpoint\":\"https://testnet-rpc.wvm.dev\",\"chain_id\":9496,\"timeout\":\"30s\",\"private_key_hex\":\"PRIVATE_KEY_HEX\"}"
# Or with web3signer:
# da_config = "{\"endpoint\":\"https://testnet-rpc.wvm.dev\",\"chain_id\":9496,\"timeout\":\"30s\",\"web3_signer_endpoint\":\"http://localhost:9000\"}"

### p2p config ###

# p2p listen address in the format of /ip4/ip_address/tcp/tcp_port
p2p_listen_address = "{{ .P2PConfig.ListenAddress }}"

# list of nodes used for P2P bootstrapping in the format of /ip4/ip_address/tcp/port/p2p/ID
p2p_bootstrap_nodes = "{{ .P2PConfig.BootstrapNodes }}"

# list of persistent peers in the format of /ip4/ip_address/tcp/port/p2p/ID
p2p_persistent_nodes = "{{ .P2PConfig.PersistentNodes }}"

# max number of cached messages by gossipsub protocol
p2p_gossip_cache_size = {{ .P2PConfig.GossipSubCacheSize }}

# time interval to check if no p2p nodes are connected to bootstrap again
p2p_bootstrap_retry_time = "{{ .P2PConfig.BootstrapRetryTime }}"

# set to false to disable P2P nodes discovery used to auto-connect to other peers in the P2P network
p2p_discovery_enabled= "{{ .P2PConfig.DiscoveryEnabled }}"

# set to false to disable block syncing from p2p
p2p_blocksync_enabled= "{{ .P2PConfig.BlockSyncEnabled }}"

# time interval used to periodically check for missing blocks and retrieve it from other peers on demand using P2P
p2p_blocksync_block_request_interval= "{{ .P2PConfig.BlockSyncRequestIntervalTime }}"

### settlement config ###
settlement_layer = "{{ .SettlementLayer }}" # mock, dymension

# dymension config
settlement_node_address = "{{ .SettlementConfig.NodeAddress }}"
settlement_gas_limit = {{ .SettlementConfig.GasLimit }}
settlement_gas_prices = "{{ .SettlementConfig.GasPrices }}"
settlement_gas_fees = "{{ .SettlementConfig.GasFees }}"
retry_max_delay = "{{ .SettlementConfig.RetryMaxDelay }}"
retry_min_delay = "{{ .SettlementConfig.RetryMinDelay }}"
retry_attempts = "{{ .SettlementConfig.RetryAttempts }}"
batch_acceptance_timeout = "{{ .SettlementConfig.BatchAcceptanceTimeout }}"
batch_acceptance_attempts = "{{ .SettlementConfig.BatchAcceptanceAttempts }}"

#keyring and key name to be used for sequencer 
keyring_backend = "{{ .SettlementConfig.KeyringBackend }}"
keyring_home_dir = "{{ .SettlementConfig.KeyringHomeDir }}"
dym_account_name = "{{ .SettlementConfig.DymAccountName }}"


############################
###       DB options     ###
############################
[db]

# When true, the database will write synchronously.
# should be set to true for sequencer node
sync_writes = {{ .DBConfig.SyncWrites }}

# When true, the database will run in-memory only (FOR EXPERIMENTAL USE ONLY)
in_memory = {{ .DBConfig.InMemory }}

# When zero/empty, uses default
badger_num_compactors = {{ .DBConfig.BadgerCompactors }}

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
