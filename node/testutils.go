package node

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/config"
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
	// TODO(omritoptix): Test with and without aggregator mode.
	pubkeyBytes, _ := pubkey.Raw()
	mockConfigFmt := `
	{"proposer_pub_key": "%s"}
	`
	if blockManagerConfig == nil {
		blockManagerConfig = &config.BlockManagerConfig{BlockBatchSize: 1, BatchSyncInterval: time.Second * 5, BlockTime: 100 * time.Millisecond}
	}
	mockConfig := fmt.Sprintf(mockConfigFmt, hex.EncodeToString(pubkeyBytes))
	nodeConfig := config.DefaultNodeConfig
	nodeConfig.Aggregator = isAggregator
	nodeConfig.BlockManagerConfig = *blockManagerConfig
	nodeConfig.SettlementConfig = mockConfig
	node, err := NewNode(context.Background(), nodeConfig, key, signingKey, proxy.NewLocalClientCreator(app), &types.GenesisDoc{ChainID: "test"}, log.TestingLogger())
	if err != nil {
		return nil, err
	}
	return node, nil
}
