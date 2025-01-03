package commands

import (
	"fmt"

	cfg "github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/store"
	"github.com/spf13/cobra"
)

var mainPrefix = []byte{0}

// run3dMigrationCmd
func Run3dMigrationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run-3d-migration",
		Aliases: []string{"run-3d-migration"},
		Short:   "Show this node's ID",
		RunE:    run3dMigration,
	}
	cfg.AddNodeFlags(cmd)
	return cmd
}

func run3dMigration(cmd *cobra.Command, args []string) error {

	baseKV := store.NewDefaultKVStore(tmconfig.RootDir, tmconfig.DBPath, "dymint")
	s := store.New(store.NewPrefixKV(baseKV, mainPrefix))

	err := s.Run3DMigration()
	if err != nil {
		fmt.Println("3D dymint store migration failed. Err:", err)
	}
	fmt.Println("3D dymint store migration successful")
	return nil
}
