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
aggregator = "{{ .Aggregator }}"

# block production interval
block_time = "{{ .BlockManagerConfig.BlockTime }}"
# block production interval in case of no transactions ("0s" produces empty blocks)
empty_blocks_max_time = "{{ .BlockManagerConfig.EmptyBlocksMaxTime }}"

# triggers to submit batch to DA and settlement (both required)
block_batch_size = {{ .BlockManagerConfig.BlockBatchSize }}
batch_submit_max_time = "{{ .BlockManagerConfig.BatchSubmitMaxTime }}"

### da config ###
da_layer = "{{ .DALayer }}" # mock, celestia, avail
namespace_id = "{{ .BlockManagerConfig.NamespaceID }}"
da_config = "{{ .DAConfig }}"

# max size of batch in bytes that can be accepted by DA
block_batch_max_size_bytes = {{ .BlockManagerConfig.BlockBatchMaxSizeBytes }}

#celestia config example:
# da_config = "{\"base_url\": \"http://127.0.0.1:26659\", \"timeout\": 60000000000, \"gas_prices\":0.1, \"gas_adjustment\": 1.3, \"namespace_id\":\"000000000000ffff\"}"
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
