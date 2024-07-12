package main

import (
	"net/http"
	"os"
	"path/filepath"

	_ "net/http/pprof" // #nosec G108

	"github.com/dymensionxyz/dymint/cmd/dymint/commands"
	"github.com/dymensionxyz/dymint/config"
	"github.com/tendermint/tendermint/cmd/cometbft/commands/debug"
	"github.com/tendermint/tendermint/libs/cli"
)

func main() {
	if profile := os.Getenv("PROFILE_HOST_PORT"); profile != "" {
		go func() {
			// start a server on default serve mux
			// pprof will use default serve mux to serve profiles
			// profile can be e.g. "localhost:6060"
			//nolint:G114
			_ = http.ListenAndServe(profile, nil) // #nosec G114
		}()
	}

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
