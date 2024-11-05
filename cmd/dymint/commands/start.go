package commands

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmnode "github.com/tendermint/tendermint/node"
	tmp2p "github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/proxy"

	cfg "github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/conv"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/rpc"
)

var genesisHash []byte

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
			logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
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
		return fmt.Errorf("load or gen node key %s: %w", tmConfig.NodeKeyFile(), err)
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

	genesisChecksum, err := ComputeGenesisHash(tmConfig.GenesisFile())
	if err != nil {
		return fmt.Errorf("failed to compute genesis checksum: %w", err)
	}

	dymintNode, err := node.NewNode(
		context.Background(),
		*config,
		p2pKey,
		signingKey,
		proxy.DefaultClientCreator(tmConfig.ProxyApp, tmConfig.ABCI, tmConfig.DBDir()),
		genesis,
		genesisChecksum,
		logger,
		mempool.PrometheusMetrics("dymint"),
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
	if config.Genesis == "" {
		return fmt.Errorf("genesis file is not set")
	}

	if len(genesisHash) == 0 {
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
		return fmt.Errorf("when hashing genesis file: %w", err)
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

func ComputeGenesisHash(genesisFilePath string) (string, error) {
	fileContent, err := os.ReadFile(filepath.Clean(genesisFilePath))
	if err != nil {
		return "", err
	}

	var jsonObject map[string]interface{}
	err = json.Unmarshal(fileContent, &jsonObject)
	if err != nil {
		return "", err
	}

	keys := make([]string, 0, len(jsonObject))
	for k := range jsonObject {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sortedJSON, err := json.Marshal(jsonObject)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(sortedJSON)
	return hex.EncodeToString(hash[:]), nil
}
