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
	"github.com/go-kit/log/term"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmnode "github.com/tendermint/tendermint/node"
	tmp2p "github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/proxy"

	kitlevel "github.com/go-kit/log/level"
)

var (
	genesisHash []byte
)

// NewRunNodeCmd returns the command that allows the CLI to start a node.
// It can be used with a custom PrivValidator and in-process ABCI application.
func NewRunNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"node", "run"},
		Short:   "Run the dymint node",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			homeDir := viper.GetString(cli.HomeFlag)
			err := dymconfig.GetViperConfig(cmd, homeDir)
			if err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkGenesisHash(tmconfig); err != nil {
				return err
			}
			//logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
			colorFn := func(keyvals ...interface{}) term.FgBgColor {
				if keyvals[0] != kitlevel.Key() {
					panic(fmt.Sprintf("expected level key to be first, got %v", keyvals[0]))
				}
				switch keyvals[1].(kitlevel.Value).String() {
				case "debug":
					return term.FgBgColor{Fg: term.Blue}
				case "error":
					return term.FgBgColor{Fg: term.Red}
				default:
					return term.FgBgColor{}
				}
			}
			logger := log.NewTMLoggerWithColorFn(log.NewSyncWriter(os.Stdout), colorFn)
			err := startInProcess(&dymconfig, tmconfig, logger)
			if err != nil {
				return err
			}
			return nil
		},
	}

	cfg.AddNodeFlags(cmd)
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
