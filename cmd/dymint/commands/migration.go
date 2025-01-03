package commands

import (
	"fmt"
	"os"

	dymintconf "github.com/dymensionxyz/dymint/config"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/dymensionxyz/dymint/store"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"
)

var mainPrefix = []byte{0}

// run3dMigrationCmd
var run3dMigrationCmd = &cobra.Command{
	Use:     "run-3d-migration",
	Aliases: []string{"run-3d-migration"},
	Short:   "Show this node's ID",
	RunE:    run3dMigration,
}

func run3dMigration(cmd *cobra.Command, args []string) error {

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	conf := dymintconf.DefaultConfig("")
	ctx := server.GetServerContextFromCmd(cmd)

	err := conf.GetViperConfig(cmd, ctx.Viper.GetString(flags.FlagHome))
	if err != nil {
		return err
	}
	baseKV := store.NewKVStore(conf.RootDir, conf.DBPath, "dymint", conf.DBConfig.SyncWrites, logger)
	s := store.New(store.NewPrefixKV(baseKV, mainPrefix))

	s.run3dMigration()
	fmt.Println("Store migration successful")
	return nil
}
