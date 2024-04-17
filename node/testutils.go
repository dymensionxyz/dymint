package node

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
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
)

// TODO: should be moved to testutils
func CreateNode(isAggregator bool, blockManagerConfig *config.BlockManagerConfig) (*Node, error) {
	app := testutil.GetAppMock()
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	signingKey, pubkey, _ := crypto.GenerateEd25519Key(rand.Reader)
	pubkeyBytes, _ := pubkey.Raw()

	// Node config
	nodeConfig := config.DefaultNodeConfig

	if blockManagerConfig == nil {
		blockManagerConfig = &config.BlockManagerConfig{
			BlockBatchSize:          1,
			BlockTime:               100 * time.Millisecond,
			BatchSubmitMaxTime:      60 * time.Second,
			BlockBatchMaxSizeBytes:  1000,
			GossipedBlocksCacheSize: 50,
		}
	}
	nodeConfig.BlockManagerConfig = *blockManagerConfig
	nodeConfig.Aggregator = isAggregator

	rollappID := "rollapp_1234-1"

	// SL config
	nodeConfig.SettlementConfig = settlement.Config{ProposerPubKey: hex.EncodeToString(pubkeyBytes), RollappID: rollappID}

	node, err := NewNode(
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
