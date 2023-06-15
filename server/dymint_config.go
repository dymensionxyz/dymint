package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmos "github.com/tendermint/tendermint/libs/os"

	dymintconf "github.com/dymensionxyz/dymint/config"
	dymintconv "github.com/dymensionxyz/dymint/conv"
	"github.com/dymensionxyz/dymint/settlement"
)

// DymintContextKey defines the context key used to retrieve a server.Context from
// a command's Context.
const (
	DymintContextKey = sdk.ContextKey("dymint.context")
)

type DymintContext struct {
	Viper  *viper.Viper
	Config *dymintconf.NodeConfig
}

func NewDefaultContext() *DymintContext {
	return NewContext(viper.New(), &dymintconf.DefaultNodeConfig)
}

func NewContext(v *viper.Viper, config *dymintconf.NodeConfig) *DymintContext {
	return &DymintContext{v, config}
}

// GetDymintContextFromCmd returns a Context from a command or an empty Context
// if it has not been set.
func GetDymintContextFromCmd(cmd *cobra.Command) *DymintContext {
	if v := cmd.Context().Value(DymintContextKey); v != nil {
		dymintCtxPtr := v.(*DymintContext)
		return dymintCtxPtr
	}

	return NewDefaultContext()
}

// SetCmdDymintContext sets a command's Context value to the provided argument.
func SetCmdDymintContext(cmd *cobra.Command, dymintCtx *DymintContext) error {

	v := context.WithValue(cmd.Context(), DymintContextKey, dymintCtx)
	cmd.SetContext(v)

	return nil
}

func DymintConfigPreRunHandler(cmd *cobra.Command) error {
	dymintCtx := NewDefaultContext()
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}

	// Bind command-line flags to Viper
	if err := dymintCtx.Viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	if err := dymintCtx.Viper.BindPFlags(cmd.PersistentFlags()); err != nil {
		return err
	}

	rootDir := dymintCtx.Viper.GetString(flags.FlagHome)
	configPath := filepath.Join(rootDir, "config")
	dymintCfgFile := filepath.Clean(filepath.Join(filepath.Clean(configPath), "dymint.toml"))

	//FIXME: bind Dymint flags as well to allow overriding config file values

	//prepare default settlement config
	//TODO: move to dymint config package
	defaultSLconfig := settlement.Config{
		KeyringBackend: "test",
		NodeAddress:    "http://127.0.0.1:36657",
		RollappID:      clientCtx.ChainID,
		KeyringHomeDir: rootDir,
		DymAccountName: "sequencer",
		GasPrices:      "0.025udym",
	}
	dymintCtx.Config.SettlementConfig = defaultSLconfig

	dymintCtx.Viper.SetConfigType("toml")
	dymintCtx.Viper.SetConfigName("dymint")
	dymintCtx.Viper.AddConfigPath(configPath)
	dymintCtx.Viper.SetEnvPrefix("DYMINT")
	dymintCtx.Viper.AutomaticEnv()

	err = CheckAndCreateConfigFile(dymintCfgFile, *dymintCtx.Config)
	if err != nil {
		return err
	}

	if err := dymintCtx.Viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read in %s: %w", dymintCfgFile, err)
	}

	// Unmarshal configuration into struct
	err = dymintCtx.Viper.Unmarshal(dymintCtx.Config)
	if err != nil {
		fmt.Printf("Error unmarshaling config: %s\n", err)
	}

	err = dymintconv.GetNodeConfig(dymintCtx.Config, server.GetServerContextFromCmd(cmd).Config)
	if err != nil {
		return err
	}

	return SetCmdDymintContext(cmd, dymintCtx)
}

/* -------------------------------------------------------------------------- */
/*                                    utils                                   */
/* -------------------------------------------------------------------------- */

// CheckAndCreateFile checks if the file exists, if not it tries to create it.
func CheckAndCreateConfigFile(configFilePath string, config dymintconf.NodeConfig) error {
	// Check if file exists
	_, err := os.Stat(configFilePath)
	if os.IsNotExist(err) {
		// If file does not exist, check if directory exists
		dir := filepath.Dir(configFilePath)
		if _, err := os.Stat(dir); err != nil {
			// If directory also does not exist, return error
			return errors.New("directory does not exist")
		}

		// If directory exists, create file
		file, err := os.Create(filepath.Clean(configFilePath))
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		writeDefaultConfigFile(configFilePath, config)

	} else if err != nil {
		// If there was an error other than IsNotExist
		return err
	}

	return nil
}

// XXX: this func should probably be called by cmd/tendermint/commands/init.go
// alongside the writing of the genesis.json and priv_validator.json
func writeDefaultConfigFile(configFilePath string, config dymintconf.NodeConfig) {
	var buffer bytes.Buffer

	parseTemplate()

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	tmos.MustWriteFile(configFilePath, buffer.Bytes(), 0o644)
}

var configTemplate *template.Template

func parseTemplate() {
	var err error
	tmpl := template.New("configFileTemplate").Funcs(template.FuncMap{
		"StringsJoin": strings.Join,
	})
	if configTemplate, err = tmpl.Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

// func (c dymintconf.NodeConfig)

// Note: any changes to the comments/variables/mapstructure
// must be reflected in the appropriate struct in config/config.go
const defaultConfigTemplate = `
#######################################################
###       Dymint Configuration Options     ###
#######################################################
aggregator = "{{ .Aggregator }}"

# block production interval
block_time = "{{ .BlockManagerConfig.BlockTime }}"
# block time for empty blocks (block time in case of no transactions)
empty_blocks_max_time = "{{ .BlockManagerConfig.EmptyBlocksMaxTime }}"

# triggers to submit batch to DA and settlement (both required)
block_batch_size = {{ .BlockManagerConfig.BlockBatchSize }}
batch_submit_max_time = "{{ .BlockManagerConfig.BatchSubmitMaxTime }}"

### da config ###
da_layer = "{{ .DALayer }}" # mock, celestia
da_start_height = {{ .BlockManagerConfig.DAStartHeight }}
namespace_id = "{{ .BlockManagerConfig.NamespaceID }}"

# max size of batch in bytes that can be accepted by DA
block_batch_size_bytes = {{ .BlockManagerConfig.BlockBatchSizeBytes }}

#celestia config
# da_config = "{\"base_url\": \"http://127.0.0.1:26659\", \"timeout\": 60000000000, \"fee\":20000, \"gas_limit\": 20000000, \"namespace_id\":\"000000000000ffff\"}"
da_config = "{{ .DAConfig }}"

### settlement config ###
settlement_layer = "{{ .SettlementLayer }}" # mock, dymension

# dymension config
node_address = "{{ .SettlementConfig.NodeAddress }}"
gas_limit = {{ .SettlementConfig.GasLimit }}
gas_prices = "{{ .SettlementConfig.GasPrices }}"
gas_fees = "{{ .SettlementConfig.GasFees }}"

#keyring and key name to be used for sequencer 
keyring_backend = "{{ .SettlementConfig.KeyringBackend }}"
keyring_home_dir = "{{ .SettlementConfig.KeyringHomeDir }}"
dym_account_name = "{{ .SettlementConfig.DymAccountName }}"
`
