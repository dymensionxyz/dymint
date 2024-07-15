package main

import (
	"os"
	"path/filepath"

	_ "net/http/pprof" // #nosec G108

	"github.com/dymensionxyz/dymint/cmd/dymint/commands"
	"github.com/dymensionxyz/dymint/config"
	"github.com/tendermint/tendermint/cmd/cometbft/commands/debug"
	"github.com/tendermint/tendermint/libs/cli"
)

func main() {
	rootCmd := commands.RootCmd
	rootCmd.AddCommand(
		commands.InitFilesCmd,
		commands.ShowSequencer,
		commands.ShowNodeIDCmd,
		debug.DebugCmd,
		cli.NewCompletionCmd(rootCmd, true),
	)

	// Create & start node
	rootCmd.AddCommand(commands.NewRunNodeCmd())

	cmd := cli.PrepareBaseCmd(rootCmd, "DM", os.ExpandEnv(filepath.Join("$HOME", config.DefaultDymintDir)))
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
