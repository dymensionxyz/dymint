package block

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/go-diodes"

	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/utils"
	"github.com/libp2p/go-libp2p/core/crypto"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/tendermint/tendermint/proxy"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/state"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

// Manager is responsible for aggregating transactions into blocks.
type Manager struct {
	// Configuration
	conf        config.BlockManagerConfig
	genesis     *tmtypes.GenesisDoc
	proposerKey crypto.PrivKey

	// Store and execution
	store     store.Store
	lastState types.State
	executor  *state.BlockExecutor

	// Clients and servers
	pubsub           *pubsub.Server
	p2pClient        *p2p.Client
	dalc             da.DataAvailabilityLayerClient
	settlementClient settlement.LayerI

	// Data retrieval
	retriever da.BatchRetriever

	// Synchronization
	syncTargetDiode diodes.Diode
	syncTarget      uint64
	isSyncedCond    sync.Cond

	// Block production
	shouldProduceBlocksCh chan bool
	produceEmptyBlockCh   chan bool
	lastSubmissionTime    int64
	batchInProcess        atomic.Value
	produceBlockMutex     sync.Mutex
	applyCachedBlockMutex sync.Mutex

	// Logging
	logger types.Logger

	// Previous data
	prevBlock  map[uint64]*types.Block
	prevCommit map[uint64]*types.Commit
}

// NewManager creates new block Manager.
func NewManager(
	proposerKey crypto.PrivKey,
	conf config.BlockManagerConfig,
	genesis *tmtypes.GenesisDoc,
	store store.Store,
	mempool mempool.Mempool,
	proxyApp proxy.AppConns,
	dalc da.DataAvailabilityLayerClient,
	settlementClient settlement.LayerI,
	eventBus *tmtypes.EventBus,
	pubsub *pubsub.Server,
	p2pClient *p2p.Client,
	logger types.Logger,
) (*Manager, error) {
	proposerAddress, err := getAddress(proposerKey)
	if err != nil {
		return nil, err
	}

	exec, err := state.NewBlockExecutor(proposerAddress, conf.NamespaceID, genesis.ChainID, mempool, proxyApp, eventBus, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create block executor: %w", err)
	}
	s, err := getInitialState(store, genesis, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial state: %w", err)
	}

	batchInProcess := atomic.Value{}
	batchInProcess.Store(false)

	agg := &Manager{
		pubsub:           pubsub,
		p2pClient:        p2pClient,
		proposerKey:      proposerKey,
		conf:             conf,
		genesis:          genesis,
		lastState:        s,
		store:            store,
		executor:         exec,
		dalc:             dalc,
		settlementClient: settlementClient,
		retriever:        dalc.(da.BatchRetriever),
		// channels are buffered to avoid blocking on input/output operations, buffer sizes are arbitrary
		syncTargetDiode:       diodes.NewOneToOne(1, nil),
		isSyncedCond:          *sync.NewCond(new(sync.Mutex)),
		batchInProcess:        batchInProcess,
		shouldProduceBlocksCh: make(chan bool, 1),
		produceEmptyBlockCh:   make(chan bool, 1),
		logger:                logger,
		prevBlock:             make(map[uint64]*types.Block),
		prevCommit:            make(map[uint64]*types.Commit),
	}

	return agg, nil
}

// Start starts the block manager.
func (m *Manager) Start(ctx context.Context, isAggregator bool) error {
	m.logger.Info("Starting the block manager")

	if isAggregator {
		// make sure local signing key is the registered on the hub
		slProposerKey := m.settlementClient.GetProposer().PublicKey.Bytes()
		localProposerKey, _ := m.proposerKey.GetPublic().Raw()
		if !bytes.Equal(slProposerKey, localProposerKey) {
			return fmt.Errorf("proposer key mismatch: settlement proposer key: %s, block manager proposer key: %s", slProposerKey, m.proposerKey.GetPublic())
		}
		m.logger.Info("Starting in aggregator mode")
	}

	// Check if InitChain flow is needed
	if m.lastState.LastBlockHeight+1 == m.genesis.InitialHeight {
		err := m.RunInitChain(ctx)
		if err != nil {
			return err
		}
	}

	err := m.syncBlockManager(ctx)
	if err != nil {
		err = fmt.Errorf("failed to sync block manager: %w", err)
		return err
	}

	if isAggregator {
		go m.ProduceBlockLoop(ctx)
		go m.SubmitLoop(ctx)
	} else {
		go m.RetriveLoop(ctx)
		go m.SyncTargetLoop(ctx)
	}

	m.EventListener(ctx, isAggregator)

	return nil
}

// syncBlockManager enforces the node to be synced on initial run.
func (m *Manager) syncBlockManager(ctx context.Context) error {
	resultRetrieveBatch, err := m.getLatestBatchFromSL(ctx)
	// Set the syncTarget according to the result
	if err != nil {
		// TODO: separate between fresh rollapp and non-registred rollapp
		if err == settlement.ErrBatchNotFound {
			// Since we requested the latest batch and got batch not found it means
			// the SL still hasn't got any batches for this chain.
			m.logger.Info("No batches for chain found in SL. Start writing first batch")
			atomic.StoreUint64(&m.syncTarget, uint64(m.genesis.InitialHeight-1))
			return nil
		}
		return err
	}
	atomic.StoreUint64(&m.syncTarget, resultRetrieveBatch.EndHeight)
	err = m.syncUntilTarget(ctx, resultRetrieveBatch.EndHeight)
	if err != nil {
		return err
	}

	m.logger.Info("Synced", "current height", m.store.Height(), "syncTarget", atomic.LoadUint64(&m.syncTarget))
	return nil
}

// updateSyncParams updates the sync target and state index if necessary
func (m *Manager) updateSyncParams(endHeight uint64) {
	types.RollappHubHeightGauge.Set(float64(endHeight))
	m.logger.Info("Received new syncTarget", "syncTarget", endHeight)
	atomic.StoreUint64(&m.syncTarget, endHeight)
	atomic.StoreInt64(&m.lastSubmissionTime, time.Now().UnixNano())
}

func getAddress(key crypto.PrivKey) ([]byte, error) {
	rawKey, err := key.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	return tmcrypto.AddressHash(rawKey), nil
}

// EventListener registers events to callbacks.
func (m *Manager) EventListener(ctx context.Context, isAggregator bool) {
	go utils.SubscribeAndHandleEvents(ctx, m.pubsub, "nodeHealthStatusHandler", events.EventQueryHealthStatus, m.healthStatusEventCallback, m.logger)
	if !isAggregator {
		go utils.SubscribeAndHandleEvents(ctx, m.pubsub, "ApplyBlockLoop", p2p.EventQueryNewNewGossipedBlock, m.applyBlockCallback, m.logger, 100)
	}
}

func (m *Manager) healthStatusEventCallback(event pubsub.Message) {
	eventData := event.Data().(*events.EventDataHealthStatus)
	m.logger.Info("Received health status event", "eventData", eventData)
	m.shouldProduceBlocksCh <- eventData.Healthy
}

func (m *Manager) applyBlockCallback(event pubsub.Message) {
	m.logger.Debug("Received new block event", "eventData", event.Data(), "cachedBlocks", len(m.prevBlock))
	eventData := event.Data().(p2p.GossipedBlock)
	block := eventData.Block
	commit := eventData.Commit

	if block.Header.Height != m.store.Height()+1 {
		if block.Header.Height > m.store.Height() {
			m.prevBlock[block.Header.Height] = &block
			m.prevCommit[block.Header.Height] = &commit
			m.logger.Debug("Caching block", "block height", block.Header.Height, "store height", m.store.Height())
		}
	} else {
		err := m.applyBlock(context.Background(), &block, &commit, blockMetaData{source: gossipedBlock})
		if err != nil {
			m.logger.Debug("Failed to apply block", "err", err)
		}
	}
	err := m.attemptApplyCachedBlocks(context.Background())
	if err != nil {
		m.logger.Debug("Failed to apply previous cached blocks", "err", err)
	}
}

// getLatestBatchFromSL gets the latest batch from the SL
func (m *Manager) getLatestBatchFromSL(ctx context.Context) (*settlement.ResultRetrieveBatch, error) {
	return m.settlementClient.RetrieveBatch()
}

// getInitialState tries to load lastState from Store, and if it's not available it reads GenesisDoc.
func getInitialState(store store.Store, genesis *tmtypes.GenesisDoc, logger types.Logger) (types.State, error) {
	s, err := store.LoadState()
	if err == types.ErrNoStateFound {
		logger.Info("failed to find state in the store, creating new state from genesis")
		return types.NewFromGenesisDoc(genesis)
	}

	return s, err
}
