package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/indexers/txindex"
	"github.com/dymensionxyz/dymint/indexers/txindex/kv"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"

	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"

	tmcfg "github.com/tendermint/tendermint/config"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	localda "github.com/dymensionxyz/dymint/da/local"
	blockidxkv "github.com/dymensionxyz/dymint/indexers/blockindexer/kv"
	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	nodemempool "github.com/dymensionxyz/dymint/node/mempool"
	slregistry "github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/dymensionxyz/dymint/store"
)

const (
	DefaultTestBatchSize = 5
)

/* -------------------------------------------------------------------------- */
/*                                    utils                                   */
/* -------------------------------------------------------------------------- */

func GetManagerWithProposerKey(conf config.BlockManagerConfig, proposerKey crypto.PrivKey, settlementlc settlement.ClientI, genesisHeight, storeInitialHeight, storeLastBlockHeight int64, proxyAppConns proxy.AppConns, mockStore store.Store) (*block.Manager, error) {
	genesis := GenerateGenesis(genesisHeight)
	// Change the LastBlockHeight to avoid calling InitChainSync within the manager
	// And updating the state according to the genesis.
	raw, _ := proposerKey.GetPublic().Raw()
	pubkey := ed25519.PubKey(raw)

	state := GenerateStateWithSequencer(storeInitialHeight, storeLastBlockHeight, pubkey)
	state.ChainID = genesis.ChainID
	var managerStore store.Store
	if mockStore == nil {
		managerStore = store.New(store.NewDefaultInMemoryKVStore())
	} else {
		managerStore = mockStore
	}
	if _, err := managerStore.SaveState(state, nil); err != nil {
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
		settlementlc = slregistry.GetClient(slregistry.Local)
	}

	proposerPubKey := proposerKey.GetPublic()
	pubKeybytes, err := proposerPubKey.Raw()
	if err != nil {
		return nil, err
	}

	err = initSettlementLayerMock(genesis.ChainID, settlementlc, hex.EncodeToString(pubKeybytes), pubsubServer, logger)
	if err != nil {
		return nil, err
	}

	dacl := GetMockDALC(logger)

	var proxyApp proxy.AppConns
	if proxyAppConns == nil {
		proxyApp = GetABCIProxyAppMock(logger.With("module", "proxy"))
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
	p2pClient, err := p2p.NewClient(config.P2PConfig{
		GossipSubCacheSize:           50,
		BootstrapRetryTime:           30 * time.Second,
		BlockSyncRequestIntervalTime: 30 * time.Second,
		BlockSyncEnabled:             true,
	}, p2pKey, "TestChain", managerStore, pubsubServer, datastore.NewMapDatastore(), logger)
	if err != nil {
		return nil, err
	}

	config := config.NodeConfig{
		BlockManagerConfig: conf,
		DAConfig:           "",
	}

	indexer, err := createIndexerService()
	if err != nil {
		return nil, err
	}

	manager, err := block.NewManager(proposerKey, config, genesis, "", managerStore, mp, proxyApp, settlementlc, dacl, nil,
		pubsubServer, p2pClient, indexer, logger)
	if err != nil {
		return nil, err
	}

	p2pValidator := p2p.NewValidator(logger, manager)
	p2pClient.SetTxValidator(p2pValidator.TxValidator(mp, mpIDs))
	p2pClient.SetBlockValidator(p2pValidator.BlockValidator())

	manager.P2PClient = p2pClient
	if err = p2pClient.Start(context.Background()); err != nil {
		return nil, err
	}

	manager.Sequencers.Set([]types.Sequencer{*state.Proposer.Load()})

	return manager, nil
}

func GetManager(conf config.BlockManagerConfig, settlementlc settlement.ClientI, genesisHeight, storeInitialHeight, storeLastBlockHeight int64, proxyAppConns proxy.AppConns, mockStore store.Store) (*block.Manager, error) {
	proposerKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	return GetManagerWithProposerKey(conf, proposerKey, settlementlc, genesisHeight, storeInitialHeight, storeLastBlockHeight, proxyAppConns, mockStore)
}

func GetMockDALC(logger log.Logger) da.DataAvailabilityLayerClient {
	dalc := &localda.DataAvailabilityLayerClient{}
	initDALCMock(dalc, pubsub.NewServer(), logger)
	return dalc
}

func initDALCMock(dalc da.DataAvailabilityLayerClient, pubsubServer *pubsub.Server, logger log.Logger) {
	_ = dalc.Init(nil, pubsubServer, store.NewDefaultInMemoryKVStore(), logger)
	_ = dalc.Start()
}

func initSettlementLayerMock(rollappId string, settlementlc settlement.ClientI, proposer string, pubsubServer *pubsub.Server, logger log.Logger) error {
	err := settlementlc.Init(settlement.Config{ProposerPubKey: proposer}, rollappId, pubsubServer, logger)
	if err != nil {
		return err
	}
	err = settlementlc.Start()
	if err != nil {
		return err
	}
	return nil
}

func GetManagerConfig() config.BlockManagerConfig {
	return config.BlockManagerConfig{
		BlockTime:                  100 * time.Millisecond,
		BatchSubmitBytes:           1000000,
		BatchSubmitTime:            30 * time.Minute,
		MaxSkewTime:                24 * time.Hour,
		SequencerSetUpdateInterval: config.DefaultSequencerSetUpdateInterval,
	}
}

func createIndexerService() (*txindex.IndexerService, error) {
	kvStore := store.NewDefaultInMemoryKVStore()
	txIndexer := kv.NewTxIndex(kvStore)
	blockIndexer := blockidxkv.New(store.NewPrefixKV(kvStore, []byte("block_events")))

	indexerService := txindex.NewIndexerService(txIndexer, blockIndexer, nil)

	return indexerService, nil
}
