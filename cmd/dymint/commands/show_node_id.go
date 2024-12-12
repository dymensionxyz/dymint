package commands

import (
	"fmt"

	"github.com/dymensionxyz/dymint/conv"
	"github.com/libp2p/go-libp2p"
	"github.com/spf13/cobra"

	"github.com/tendermint/tendermint/p2p"
)


var ShowNodeIDCmd = &cobra.Command{
	Use:     "show-node-id",
	Aliases: []string{"show_node_id"},
	Short:   "Show this node's ID",
	RunE:    showNodeID,
}

func showNodeID(cmd *cobra.Command, args []string) error {
	nodeKey, err := p2p.LoadNodeKey(tmconfig.NodeKeyFile())
	if err != nil {
		return err
	}
	signingKey, err := conv.GetNodeKey(nodeKey)
	if err != nil {
		return err
	}
	
	host, err := libp2p.New(libp2p.Identity(signingKey))
	if err != nil {
		return err
	}

	fmt.Println(host.ID())
	return nil
}
