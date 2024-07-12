package main

import (
	"net/http"
	"os"
	"path/filepath"

	_ "net/http/pprof"

	"github.com/dymensionxyz/dymint/cmd/dymint/commands"
	"github.com/dymensionxyz/dymint/config"
	"github.com/tendermint/tendermint/cmd/cometbft/commands/debug"
	"github.com/tendermint/tendermint/libs/cli"
)

func main() {
	go func() {
		// start a server on default serve mux
		// pprof will use default serve mux to serve profiles
		http.ListenAndServe("localhost:6060", nil)
	}()

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
