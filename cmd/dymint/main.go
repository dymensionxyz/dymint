package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dymensionxyz/dymint/cmd/dymint/commands"
	"github.com/dymensionxyz/dymint/config"
	"github.com/tendermint/tendermint/cmd/cometbft/commands/debug"
	"github.com/tendermint/tendermint/libs/cli"
)

// The main entry point for the dymint CLI application.
func main() {
	// Initialize the root command structure
	rootCmd := commands.RootCmd
	
	// Add static commands (init, show info, debug, completion)
	rootCmd.AddCommand(
		commands.InitFilesCmd,      // Command to initialize configuration and data files
		commands.ShowSequencer,     // Command to display the sequencer ID
		commands.ShowNodeIDCmd,     // Command to display the node ID
		commands.Run3dMigrationCmd(), // Command for 3D migration process
		debug.DebugCmd,             // Integration with CometBFT's debug commands
		cli.NewCompletionCmd(rootCmd, true), // Shell completion command
	)

	// Add the primary command to run the Dymint node
	rootCmd.AddCommand(commands.NewRunNodeCmd())

	// Prepare the base command configuration.
	// Sets the application name ('DM') and the default root directory ($HOME/.dymint).
	defaultHome := os.ExpandEnv(filepath.Join("$HOME", config.DefaultDymintDir))
	cmd := cli.PrepareBaseCmd(rootCmd, "DM", defaultHome)
	
	// Execute the root command.
	if err := cmd.Execute(); err != nil {
		// Optimized: Instead of panic(), print the error to Stderr and exit gracefully 
		// with a non-zero exit code (standard CLI practice).
		_, _ = fmt.Fprintln(os.Stderr, "Error executing command:", err)
		os.Exit(1)
	}
}
