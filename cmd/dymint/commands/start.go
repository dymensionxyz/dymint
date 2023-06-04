package commands

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	cfg "github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/conv"
	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/rpc"
	"github.com/spf13/cobra"
	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmnode "github.com/tendermint/tendermint/node"
	tmp2p "github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/proxy"
)

var (
	genesisHash []byte
)

// AddNodeFlags exposes some common configuration options on the command-line
// These are exposed for convenience of commands embedding a dymint node
func AddNodeFlags(cmd *cobra.Command) {
	// bind flags
	cmd.Flags().String("moniker", tmconfig.Moniker, "node name")

	// priv val flags
	cmd.Flags().String(
		"priv_validator_laddr",
		tmconfig.PrivValidatorListenAddr,
		"socket address to listen on for connections from external priv_validator process")

	// node flags
	cmd.Flags().BytesHexVar(
		&genesisHash,
		"genesis_hash",
		[]byte{},
		"optional SHA-256 hash of the genesis file")

	// abci flags
	cmd.Flags().String(
		"proxy_app",
		tmconfig.ProxyApp,
		"proxy app address, or one of: 'kvstore',"+
			" 'persistent_kvstore' or 'noop' for local testing.")
	cmd.Flags().String("abci", tmconfig.ABCI, "specify abci transport (socket | grpc)")

	// rpc flags
	cmd.Flags().String("rpc.laddr", tmconfig.RPC.ListenAddress, "RPC listen address. Port required")
	cmd.Flags().String(
		"rpc.grpc_laddr",
		tmconfig.RPC.GRPCListenAddress,
		"GRPC listen address (BroadcastTx only). Port required")
	cmd.Flags().Bool("rpc.unsafe", tmconfig.RPC.Unsafe, "enabled unsafe rpc methods")
	cmd.Flags().String("rpc.pprof_laddr", tmconfig.RPC.PprofListenAddress, "pprof listen address (https://golang.org/pkg/net/http/pprof)")

	// p2p flags
	cmd.Flags().String(
		"p2p.laddr",
		tmconfig.P2P.ListenAddress,
		"node listen address. (0.0.0.0:0 means any interface, any port)")
	cmd.Flags().String("p2p.external-address", tmconfig.P2P.ExternalAddress, "ip:port address to advertise to peers for them to dial")
	cmd.Flags().String("p2p.seeds", tmconfig.P2P.Seeds, "comma-delimited ID@host:port seed nodes")
	cmd.Flags().String("p2p.persistent_peers", tmconfig.P2P.PersistentPeers, "comma-delimited ID@host:port persistent peers")
	cmd.Flags().String("p2p.unconditional_peer_ids",
		tmconfig.P2P.UnconditionalPeerIDs, "comma-delimited IDs of unconditional peers")
	cmd.Flags().Bool("p2p.upnp", tmconfig.P2P.UPNP, "enable/disable UPNP port forwarding")
	cmd.Flags().Bool("p2p.pex", tmconfig.P2P.PexReactor, "enable/disable Peer-Exchange")
	cmd.Flags().Bool("p2p.seed_mode", tmconfig.P2P.SeedMode, "enable/disable seed mode")
	cmd.Flags().String("p2p.private_peer_ids", tmconfig.P2P.PrivatePeerIDs, "comma-delimited private peer IDs")

}

// NewRunNodeCmd returns the command that allows the CLI to start a node.
// It can be used with a custom PrivValidator and in-process ABCI application.
func NewRunNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"node", "run"},
		Short:   "Run the dymint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkGenesisHash(tmconfig); err != nil {
				return err
			}
			logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
			err := startInProcess(&dymconfig, tmconfig, logger)
			if err != nil {
				return err
			}
			return nil
		},
	}

	cfg.AddFlags(cmd)
	AddNodeFlags(cmd)
	return cmd
}

func startInProcess(config *cfg.NodeConfig, tmConfig *tmcfg.Config, logger log.Logger) error {
	nodeKey, err := tmp2p.LoadOrGenNodeKey(tmConfig.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load or gen node key %s: %w", tmConfig.NodeKeyFile(), err)
	}
	privValKey, err := tmp2p.LoadOrGenNodeKey(tmConfig.PrivValidatorKeyFile())
	if err != nil {
		return err
	}
	genDocProvider := tmnode.DefaultGenesisDocProviderFunc(tmConfig)
	p2pKey, err := conv.GetNodeKey(nodeKey)
	if err != nil {
		return err
	}
	signingKey, err := conv.GetNodeKey(privValKey)
	if err != nil {
		return err
	}
	genesis, err := genDocProvider()
	if err != nil {
		return err
	}
	err = conv.GetNodeConfig(config, tmConfig)
	if err != nil {
		return err
	}
	logger.Info("starting node with ABCI dymint in-process", "conf", config)

	dymintNode, err := node.NewNode(
		context.Background(),
		*config,
		p2pKey,
		signingKey,
		proxy.DefaultClientCreator(tmConfig.ProxyApp, tmConfig.ABCI, tmConfig.DBDir()),
		genesis,
		logger,
	)
	if err != nil {
		return err
	}

	server := rpc.NewServer(dymintNode, tmConfig.RPC, logger)
	err = server.Start()
	if err != nil {
		return err
	}

	logger.Debug("initialization: dymint node created")
	if err := dymintNode.Start(); err != nil {
		return err
	}

	logger.Info("Started dymint node")

	// Stop upon receiving SIGTERM or CTRL-C.
	tmos.TrapSignal(logger, func() {
		logger.Info("Caught SIGTERM. Exiting...")
		if dymintNode.IsRunning() {
			if err := dymintNode.Stop(); err != nil {
				logger.Error("unable to stop the node", "error", err)
			}
		}
	})

	// Run forever.
	select {}
}

func checkGenesisHash(config *tmcfg.Config) error {
	if len(genesisHash) == 0 || config.Genesis == "" {
		return nil
	}

	// Calculate SHA-256 hash of the genesis file.
	f, err := os.Open(config.GenesisFile())
	if err != nil {
		return fmt.Errorf("can't open genesis file: %w", err)
	}
	defer func() {
		if tempErr := f.Close(); tempErr != nil {
			err = tempErr
		}
	}()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("error when hashing genesis file: %w", err)
	}
	actualHash := h.Sum(nil)

	// Compare with the flag.
	if !bytes.Equal(genesisHash, actualHash) {
		return fmt.Errorf(
			"--genesis_hash=%X does not match %s hash: %X",
			genesisHash, config.GenesisFile(), actualHash)
	}

	return nil
}
