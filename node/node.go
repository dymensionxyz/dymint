package node

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/ipfs/go-datastore"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/config"
	indexer "github.com/dymensionxyz/dymint/indexers/blockindexer"
	blockidxkv "github.com/dymensionxyz/dymint/indexers/blockindexer/kv"
	"github.com/dymensionxyz/dymint/indexers/txindex"
	"github.com/dymensionxyz/dymint/indexers/txindex/kv"
	"github.com/dymensionxyz/dymint/mempool"
	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	nodemempool "github.com/dymensionxyz/dymint/node/mempool"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	slregistry "github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/dymensionxyz/dymint/store"
)

// prefixes used in KV store to separate main node data from DALC data
var (
	mainPrefix    = []byte{0}
	dalcPrefix    = []byte{1}
	indexerPrefix = []byte{2}
)

// Node represents a client node in Dymint network.
// It connects all the components and orchestrates their work.
type Node struct {
	service.BaseService
	eventBus     *tmtypes.EventBus
	PubsubServer *pubsub.Server
	proxyApp     proxy.AppConns

	genesis *tmtypes.GenesisDoc

	conf config.NodeConfig
	P2P  *p2p.Client

	// TODO(tzdybal): consider extracting "mempool reactor"
	Mempool      mempool.Mempool
	MempoolIDs   *nodemempool.MempoolIDs
	incomingTxCh chan *p2p.GossipMessage

	baseKV       store.KV
	Store        store.Store
	BlockManager *block.Manager
	settlementlc settlement.ClientI

	TxIndexer      txindex.TxIndexer
	BlockIndexer   indexer.BlockIndexer
	IndexerService *txindex.IndexerService

	// shared context for all dymint components
	ctx    context.Context
	cancel context.CancelFunc
}

// NewNode creates new Dymint node.
func NewNode(
	ctx context.Context,
	conf config.NodeConfig,
	p2pKey crypto.PrivKey,
	signingKey crypto.PrivKey,
	clientCreator proxy.ClientCreator,
	genesis *tmtypes.GenesisDoc,
	genesisChecksum string,
	logger log.Logger,
	metrics *mempool.Metrics,
) (*Node, error) {
	proxyApp := proxy.NewAppConns(clientCreator)
	proxyApp.SetLogger(logger.With("module", "proxy"))
	if err := proxyApp.Start(); err != nil {
		return nil, fmt.Errorf("starting proxy app connections: %w", err)
	}

	eventBus := tmtypes.NewEventBus()
	eventBus.SetLogger(logger.With("module", "events"))
	if err := eventBus.Start(); err != nil {
		return nil, err
	}

	pubsubServer := pubsub.NewServer()

	var baseKV store.KV
	var dstore datastore.Datastore

	if conf.DBConfig.InMemory || (conf.RootDir == "" && conf.DBPath == "") { // this is used for testing
		logger.Info("WARNING: working in in-memory mode")
		baseKV = store.NewDefaultInMemoryKVStore()
		dstore = datastore.NewMapDatastore()
	} else {
		// TODO(omritoptx): Move dymint to const
		baseKV = store.NewKVStore(conf.RootDir, conf.DBPath, "dymint",
			store.BadgerOpts{SyncWrites: conf.DBConfig.SyncWrites, NumCompactors: conf.DBConfig.BadgerCompactors}, logger)
		path := filepath.Join(store.Rootify(conf.RootDir, conf.DBPath), "blocksync")
		var err error
		dstore, err = leveldb.NewDatastore(path, &leveldb.Options{})
		if err != nil {
			return nil, fmt.Errorf("initialize datastore at %s: %w", path, err)
		}
	}

	s := store.New(store.NewPrefixKV(baseKV, mainPrefix))
	indexerKV := store.NewPrefixKV(baseKV, indexerPrefix)

	/* ------------------------------ init DA layer ----------------------------- */

	var dalcs []da.DataAvailabilityLayerClient
	for i, daLayer := range conf.DALayer {
		dalc := registry.GetClient(daLayer)
		if dalc == nil {
			return nil, fmt.Errorf("get data availability client named '%s'", daLayer)
		}

		err := dalc.Init([]byte(conf.DAConfig[i]), pubsubServer, store.NewPrefixKV(baseKV, dalcPrefix), logger.With("module", string(dalc.GetClientType())))
		if err != nil {
			return nil, fmt.Errorf("data availability layer client initialization:  %w", err)
		}
		dalcs = append(dalcs, dalc)
	}

	/* -------------------- Init the settlement layer client -------------------- */
	settlementlc := slregistry.GetClient(slregistry.Client(conf.SettlementLayer))
	if settlementlc == nil {
		return nil, fmt.Errorf("get settlement client: named: %s", conf.SettlementLayer)
	}
	if conf.SettlementLayer == "mock" {
		conf.SettlementConfig.KeyringHomeDir = conf.RootDir
	}
	err := settlementlc.Init(conf.SettlementConfig, genesis.ChainID, pubsubServer, logger.With("module", "settlement_client"))
	if err != nil {
		return nil, fmt.Errorf("settlement layer client initialization: %w", err)
	}

	indexerService, txIndexer, blockIndexer, err := createAndStartIndexerService(conf, indexerKV, eventBus, logger)
	if err != nil {
		return nil, err
	}

	info, err := proxyApp.Query().InfoSync(proxy.RequestInfo)
	if err != nil {
		return nil, fmt.Errorf("querying info: %w", err)
	}

	height := max(genesis.InitialHeight, info.LastBlockHeight)

	mp := mempoolv1.NewTxMempool(logger, &conf.MempoolConfig, proxyApp.Mempool(), height, mempoolv1.WithMetrics(metrics))
	mpIDs := nodemempool.NewMempoolIDs()

	blockManager, err := block.NewManager(
		signingKey,
		conf,
		genesis,
		genesisChecksum,
		s,
		mp,
		proxyApp,
		settlementlc,
		dalcs,
		eventBus,
		pubsubServer,
		nil, // p2p client is set later
		indexerService,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("BlockManager initialization: %w", err)
	}

	// Set p2p client and it's validators
	p2pValidator := p2p.NewValidator(logger.With("module", "p2p_validator"), blockManager)
	p2pClient, err := p2p.NewClient(conf.P2PConfig, p2pKey, genesis.ChainID, s, pubsubServer, dstore, logger.With("module", "p2p"))
	if err != nil {
		return nil, err
	}
	p2pClient.SetTxValidator(p2pValidator.TxValidator(mp, mpIDs))
	p2pClient.SetBlockValidator(p2pValidator.BlockValidator())

	// Set p2p client in block manager
	blockManager.P2PClient = p2pClient

	ctx, cancel := context.WithCancel(ctx)
	node := &Node{
		proxyApp:       proxyApp,
		eventBus:       eventBus,
		PubsubServer:   pubsubServer,
		genesis:        genesis,
		conf:           conf,
		P2P:            p2pClient,
		BlockManager:   blockManager,
		settlementlc:   settlementlc,
		Mempool:        mp,
		MempoolIDs:     mpIDs,
		incomingTxCh:   make(chan *p2p.GossipMessage),
		baseKV:         baseKV,
		Store:          s,
		TxIndexer:      txIndexer,
		IndexerService: indexerService,
		BlockIndexer:   blockIndexer,
		ctx:            ctx,
		cancel:         cancel,
	}

	node.BaseService = *service.NewBaseService(logger, "Node", node)

	return node, nil
}

// OnStart is a part of Service interface.
func (n *Node) OnStart() error {
	n.Logger.Info("starting P2P client")
	err := n.P2P.Start(n.ctx)
	if err != nil {
		return fmt.Errorf("start P2P client: %w", err)
	}
	err = n.PubsubServer.Start()
	if err != nil {
		return fmt.Errorf("start pubsub server: %w", err)
	}
	for _, daClient := range n.BlockManager.DAClients {
		err = daClient.Start()
		if err != nil {
			return fmt.Errorf("start data availability layer client: %w", err)
		}
	}
	err = n.settlementlc.Start()
	if err != nil {
		return fmt.Errorf("start settlement layer client: %w", err)
	}
	go func() {
		if err := n.startPrometheusServer(); err != nil {
			panic(err)
		}
	}()

	// start the block manager
	err = n.BlockManager.Start(n.ctx)
	if err != nil {
		return fmt.Errorf("while starting block manager: %w", err)
	}

	return nil
}

// GetGenesis returns entire genesis doc.
func (n *Node) GetGenesis() *tmtypes.GenesisDoc {
	return n.genesis
}

// OnStop is a part of Service interface.
func (n *Node) OnStop() {
	for _, daClient := range n.BlockManager.DAClients {
		err := daClient.Stop()
		if err != nil {
			n.Logger.Error("stop data availability layer client", "error", err)
		}
	}

	err := n.settlementlc.Stop()
	if err != nil {
		n.Logger.Error("stop settlement layer client", "error", err)
	}

	err = n.P2P.Close()
	if err != nil {
		n.Logger.Error("stop P2P client", "error", err)
	}

	err = n.Store.Close()
	if err != nil {
		n.Logger.Error("close store", "error", err)
	}

	n.cancel()
}

// OnReset is a part of Service interface.
func (n *Node) OnReset() error {
	panic("OnReset - not implemented!")
}

// SetLogger sets the logger used by node.
func (n *Node) SetLogger(logger log.Logger) {
	n.Logger = logger
}

// GetLogger returns logger.
func (n *Node) GetLogger() log.Logger {
	return n.Logger
}

// EventBus gives access to Node's event bus.
func (n *Node) EventBus() *tmtypes.EventBus {
	return n.eventBus
}

// PubSubServer gives access to the Node's pubsub server
func (n *Node) PubSubServer() *pubsub.Server {
	return n.PubsubServer
}

// ProxyApp returns ABCI proxy connections to communicate with application.
func (n *Node) ProxyApp() proxy.AppConns {
	return n.proxyApp
}

func createAndStartIndexerService(
	conf config.NodeConfig,
	kvStore store.KV,
	eventBus *tmtypes.EventBus,
	logger log.Logger,
) (*txindex.IndexerService, txindex.TxIndexer, indexer.BlockIndexer, error) {
	var (
		txIndexer    txindex.TxIndexer
		blockIndexer indexer.BlockIndexer
	)

	txIndexer = kv.NewTxIndex(kvStore)
	blockIndexer = blockidxkv.New(store.NewPrefixKV(kvStore, []byte("block_events")))

	indexerService := txindex.NewIndexerService(txIndexer, blockIndexer, eventBus)
	indexerService.SetLogger(logger.With("module", "txindex"))

	if err := indexerService.Start(); err != nil {
		return nil, nil, nil, err
	}

	return indexerService, txIndexer, blockIndexer, nil
}

func (n *Node) startPrometheusServer() error {
	if n.conf.Instrumentation != nil && n.conf.Instrumentation.Prometheus {
		http.Handle("/metrics", promhttp.Handler())
		srv := &http.Server{
			Addr:         n.conf.Instrumentation.PrometheusListenAddr,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			Handler:      http.DefaultServeMux,
		}
		go func() {
			if err := srv.ListenAndServe(); err != nil {
				n.Logger.Error("Serving prometheus server.", "error", err)
			}
		}()
		n.Logger.Info("Prometheus server started", "address", n.conf.Instrumentation.PrometheusListenAddr)

		<-n.ctx.Done()
		return srv.Close()
	}
	return nil
}

func (n *Node) GetBlockManagerHeight() uint64 {
	return n.BlockManager.State.Height()
}
