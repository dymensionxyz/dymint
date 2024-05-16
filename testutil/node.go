package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/settlement"
)

func CreateNode(isSequencer bool, blockManagerConfig *config.BlockManagerConfig) (*node.Node, error) {
	app := GetAppMock()
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	signingKey, pubkey, _ := crypto.GenerateEd25519Key(rand.Reader)
	pubkeyBytes, _ := pubkey.Raw()

	// Node config
	nodeConfig := config.DefaultNodeConfig

	if blockManagerConfig == nil {
		blockManagerConfig = &config.BlockManagerConfig{
			BlockTime:              100 * time.Millisecond,
			BatchSubmitMaxTime:     60 * time.Second,
			BlockBatchMaxSizeBytes: 1000,
			MaxSupportedBatchSkew:  10,
		}
	}
	nodeConfig.BlockManagerConfig = *blockManagerConfig

	rollappID := "rollapp_1234-1"

	// SL config
	nodeConfig.SettlementConfig = settlement.Config{ProposerPubKey: hex.EncodeToString(pubkeyBytes), RollappID: rollappID}

	node, err := node.NewNode(
		context.Background(),
		nodeConfig,
		key,
		signingKey,
		proxy.NewLocalClientCreator(app),
		&types.GenesisDoc{ChainID: rollappID},
		log.TestingLogger(),
		mempool.NopMetrics(),
	)
	if err != nil {
		return nil, err
	}
	return node, nil
}
