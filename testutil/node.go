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

	"github.com/stretchr/testify/mock"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/settlement"
)

func CreateNode(isSequencer bool, blockManagerConfig *config.BlockManagerConfig, genesis *types.GenesisDoc) (*node.Node, error) {
	app := GetAppMock(EndBlock)
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	if err != nil {
		return nil, err
	}
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:         "celestia",
			DrsVersion: 0,
		},
		ConsensusParamUpdates: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas:   100,
				MaxBytes: 100,
			},
		},
	})

	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	signingKey, pubkey, _ := crypto.GenerateEd25519Key(rand.Reader)
	pubkeyBytes, _ := pubkey.Raw()

	// Node config
	nodeConfig := config.DefaultNodeConfig

	if blockManagerConfig == nil {
		blockManagerConfig = &config.BlockManagerConfig{
			BlockTime:                  100 * time.Millisecond,
			BatchSubmitTime:            60 * time.Second,
			BatchSubmitBytes:           1000,
			MaxSkewTime:                24 * time.Hour,
			SequencerSetUpdateInterval: config.DefaultSequencerSetUpdateInterval,
		}
	}
	nodeConfig.BlockManagerConfig = *blockManagerConfig

	// SL config
	nodeConfig.SettlementConfig = settlement.Config{ProposerPubKey: hex.EncodeToString(pubkeyBytes)}

	node, err := node.NewNode(
		context.Background(),
		nodeConfig,
		key,
		signingKey,
		proxy.NewLocalClientCreator(app),
		genesis,
		"",
		log.TestingLogger(),
		mempool.NopMetrics(),
	)
	if err != nil {
		return nil, err
	}
	return node, nil
}
