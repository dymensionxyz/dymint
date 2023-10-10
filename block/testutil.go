package block

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	mockda "github.com/dymensionxyz/dymint/da/mock"
	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	nodemempool "github.com/dymensionxyz/dymint/node/mempool"
	slregistry "github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/dymensionxyz/dymint/store"
	tmcfg "github.com/tendermint/tendermint/config"
)

const (
	defaultBatchSize = 5
	batchLimitBytes  = 2000
)

/* -------------------------------------------------------------------------- */
/*                                    utils                                   */
/* -------------------------------------------------------------------------- */

func getManager(conf config.BlockManagerConfig, settlementlc settlement.LayerI, dalc da.DataAvailabilityLayerClient, genesisHeight int64, storeInitialHeight int64, storeLastBlockHeight int64, proxyAppConns proxy.AppConns, mockStore store.Store) (*Manager, error) {
	genesis := testutil.GenerateGenesis(genesisHeight)
	// Change the LastBlockHeight to avoid calling InitChainSync within the manager
	// And updating the state according to the genesis.
	state := testutil.GenerateState(storeInitialHeight, storeLastBlockHeight)
	var managerStore store.Store
	if mockStore == nil {
		managerStore = store.New(store.NewDefaultInMemoryKVStore())
	} else {
		managerStore = mockStore
	}
	if _, err := managerStore.UpdateState(state, nil); err != nil {
		return nil, err
	}

	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	err := pubsubServer.Start()
	if err != nil {
		return nil, err
	}

	// Init the settlement layer mock
	if settlementlc == nil {
		settlementlc = slregistry.GetClient(slregistry.Mock)
	}
	//TODO(omritoptix): Change the initialization. a bit dirty.
	proposerKey, proposerPubKey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	pubKeybytes, err := proposerPubKey.Raw()
	if err != nil {
		return nil, err
	}

	err = initSettlementLayerMock(settlementlc, hex.EncodeToString(pubKeybytes), pubsubServer, logger)
	if err != nil {
		return nil, err
	}

	if dalc == nil {
		dalc = &mockda.DataAvailabilityLayerClient{}
	}
	initDALCMock(dalc, pubsubServer, logger)

	var proxyApp proxy.AppConns
	if proxyAppConns == nil {
		proxyApp = testutil.GetABCIProxyAppMock(logger.With("module", "proxy"))
		if err := proxyApp.Start(); err != nil {
			return nil, err
		}
	} else {
		proxyApp = proxyAppConns
	}

	mp := mempoolv1.NewTxMempool(logger, tmcfg.DefaultMempoolConfig(), proxyApp.Mempool(), 0)
	mpIDs := nodemempool.NewMempoolIDs()

	// Init p2p client and validator
	p2pKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	p2pClient, err := p2p.NewClient(config.P2PConfig{}, p2pKey, "TestChain", logger)
	if err != nil {
		return nil, err
	}
	p2pValidator := p2p.NewValidator(logger, pubsubServer)
	p2pClient.SetTxValidator(p2pValidator.TxValidator(mp, mpIDs))
	p2pClient.SetBlockValidator(p2pValidator.BlockValidator())

	if err = p2pClient.Start(context.Background()); err != nil {
		return nil, err
	}

	manager, err := NewManager(proposerKey, conf, genesis, managerStore, mp, proxyApp, dalc, settlementlc, nil,
		pubsubServer, p2pClient, logger)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

// TODO(omritoptix): Possible move out to a generic testutil
func getMockDALC(logger log.Logger) da.DataAvailabilityLayerClient {
	dalc := &mockda.DataAvailabilityLayerClient{}
	initDALCMock(dalc, pubsub.NewServer(), logger)
	return dalc
}

// TODO(omritoptix): Possible move out to a generic testutil
func initDALCMock(dalc da.DataAvailabilityLayerClient, pubsubServer *pubsub.Server, logger log.Logger) {
	_ = dalc.Init(nil, pubsubServer, store.NewDefaultInMemoryKVStore(), logger)
	_ = dalc.Start()
}

// TODO(omritoptix): Possible move out to a generic testutil
func initSettlementLayerMock(settlementlc settlement.LayerI, proposer string, pubsubServer *pubsub.Server, logger log.Logger) error {
	err := settlementlc.Init(settlement.Config{ProposerPubKey: proposer}, pubsubServer, logger)
	if err != nil {
		return err
	}
	err = settlementlc.Start()
	if err != nil {
		return err
	}
	return nil
}

func getManagerConfig() config.BlockManagerConfig {
	return config.BlockManagerConfig{
		BlockTime:              100 * time.Millisecond,
		BlockBatchSize:         defaultBatchSize,
		BlockBatchMaxSizeBytes: 1000,
		BatchSubmitMaxTime:     30 * time.Minute,
		NamespaceID:            "0102030405060708",
	}
}
