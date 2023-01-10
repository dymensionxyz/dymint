package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/privval"
)

// ShowSequencer adds capabilities for showing the validator info.
var ShowSequencer = &cobra.Command{
	Use:     "show-sequencer",
	Aliases: []string{"show_sequencer"},
	Short:   "Show this node's sequencer info",
	RunE:    showSequencer,
	// PreRun:  deprecateSnakeCase,
}

func showSequencer(cmd *cobra.Command, args []string) error {
	keyFilePath := tmconfig.PrivValidatorKeyFile()
	if !tmos.FileExists(keyFilePath) {
		return fmt.Errorf("sequencer file %s does not exist", keyFilePath)
	}

	pv := privval.LoadFilePV(keyFilePath, tmconfig.PrivValidatorStateFile())

	pubKey, err := pv.GetPubKey()
	if err != nil {
		return fmt.Errorf("can't get pubkey: %w", err)
	}

	bz, err := tmjson.Marshal(pubKey)
	if err != nil {
		return fmt.Errorf("failed to marshal sequencer pubkey: %w", err)
	}

	fmt.Println(string(bz))
	return nil
}
