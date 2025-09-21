package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dymensionxyz/dymint/config"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"
)

var (
	tmconfig  = cfg.DefaultConfig()
	dymconfig = config.DefaultNodeConfig
	logger    = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
)

func init() {
	registerFlagsRootCmd(RootCmd)
}

func registerFlagsRootCmd(cmd *cobra.Command) {
	cmd.PersistentFlags().String("log_level", tmconfig.LogLevel, "log level")
}

// ParseConfig retrieves the default environment configuration,
// sets up the Dymint root and ensures that the root exists
func ParseConfig(cmd *cobra.Command) (*cfg.Config, error) {
	conf := cfg.DefaultConfig()
	err := viper.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	var home string
	if os.Getenv("DYMINTHOME") != "" {
		home = os.Getenv("DYMINTHOME")
	} else {
		home, err = cmd.Flags().GetString(cli.HomeFlag)
		if err != nil {
			return nil, err
		}
	}

	conf.RootDir = home

	conf.SetRoot(conf.RootDir)
	cfg.EnsureRoot(conf.RootDir)
	if err := conf.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("in config file: %w", err)
	}

	cfg := config.DefaultConfig(home)
	config.EnsureRoot(conf.RootDir, cfg)
	return conf, nil
}

// RootCmd is the root command for Dymint core.
var RootCmd = &cobra.Command{
	Use:   "dymint",
	Short: "ABCI-client implementation for dymension's autonomous rollapps",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		v := viper.GetViper()

		// cmd.Flags() includes flags from this command and all persistent flags from the parent
		if err := v.BindPFlags(cmd.Flags()); err != nil {
			return err
		}

		tmconfig, err = ParseConfig(cmd)
		if err != nil {
			return err
		}

		if tmconfig.LogFormat == cfg.LogFormatJSON {
			logger = log.NewTMJSONLogger(log.NewSyncWriter(os.Stdout))
		}

		logger, err = tmflags.ParseLogLevel(tmconfig.LogLevel, logger, cfg.DefaultLogLevel)
		if err != nil {
			return err
		}

		if viper.GetBool(cli.TraceFlag) {
			logger = log.NewTracingLogger(logger)
		}

		logger = logger.With("module", "main")
		return nil
	},
}
