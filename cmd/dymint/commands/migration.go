package commands

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/server"
	cfg "github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/store"
	"github.com/spf13/cobra"
)

var mainPrefix = []byte{0}

const rollappParamDAFlag = "rollappparam-da-mock"

// Run3dMigrationCmd migrates store to 3D version (1.3.0) for old rollapps.
func Run3dMigrationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run-3d-migration",
		Aliases: []string{"run-3d-migration"},
		Short:   "Migrate dymint store to 3D",
		RunE:    run3dMigration,
	}
	cmd.Flags().Bool(rollappParamDAFlag, false, "migrate using mock DA")
	cfg.AddNodeFlags(cmd)
	return cmd
}

func run3dMigration(cmd *cobra.Command, args []string) error {
	serverCtx := server.GetServerContextFromCmd(cmd)
	cfg := serverCtx.Config

	baseKV := store.NewDefaultKVStore(cfg.RootDir, cfg.DBPath, "dymint")
	s := store.New(store.NewPrefixKV(baseKV, mainPrefix))

	daMock := serverCtx.Viper.GetBool(rollappParamDAFlag)
	da := "celestia"
	if daMock {
		da = "mock"
	}
	err := store.Run3DMigration(s, da)
	if err != nil {
		return fmt.Errorf("3D dymint store migration failed. err:%w", err)
	}
	fmt.Println("3D dymint store migration successful")
	return nil
}
