package node

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"
)

func CreateNode(isAggregator bool, blockManagerConfig *config.BlockManagerConfig) (*Node, error) {
	app := testutil.GetAppMock()
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	signingKey, pubkey, _ := crypto.GenerateEd25519Key(rand.Reader)
	pubkeyBytes, _ := pubkey.Raw()

	// Node config
	nodeConfig := config.DefaultNodeConfig

	if blockManagerConfig == nil {
		blockManagerConfig = &config.BlockManagerConfig{BatchSyncInterval: time.Second * 5, BlockTime: 100 * time.Millisecond}
	}
	nodeConfig.BlockManagerConfig = *blockManagerConfig
	nodeConfig.Aggregator = isAggregator

	// SL config
	nodeConfig.SettlementConfig = settlement.Config{ProposerPubKey: hex.EncodeToString(pubkeyBytes)}

	node, err := NewNode(context.Background(), nodeConfig, key, signingKey, proxy.NewLocalClientCreator(app), &types.GenesisDoc{ChainID: "test"}, log.TestingLogger())
	if err != nil {
		return nil, err
	}
	return node, nil
}
