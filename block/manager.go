package block

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	uevent "github.com/dymensionxyz/dymint/utils/event"

	"code.cloudfoundry.org/go-diodes"

	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/libp2p/go-libp2p/core/crypto"

	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/tendermint/tendermint/proxy"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

// Manager is responsible for aggregating transactions into blocks.
type Manager struct {
	// Configuration
	Conf        config.BlockManagerConfig
	Genesis     *tmtypes.GenesisDoc
	ProposerKey crypto.PrivKey

	// Store and execution
	Store     store.Store
	LastState types.State
	Executor  *Executor

	// Clients and servers
	Pubsub    *pubsub.Server
	p2pClient *p2p.Client
	DAClient  da.DataAvailabilityLayerClient
	SLClient  settlement.LayerI

	// Data retrieval
	Retriever da.BatchRetriever

	SyncTargetDiode diodes.Diode
	SyncTarget      atomic.Uint64

	// Block production
	shouldProduceBlocksCh chan bool
	produceEmptyBlockCh   chan bool
	lastSubmissionTime    atomic.Int64

	/*
		Guard against triggering a new batch submission when the old one is still going on (taking a while)
	*/
	submitBatchMutex sync.Mutex

	/*
		Protect against producing two blocks at once if the first one is taking a while
		Also, used to protect against the block production that occurs when batch submission thread
		creates its empty block.
	*/
	produceBlockMutex sync.Mutex

	/*
		Protect against processing two blocks at once when there are two routines handling incoming gossiped blocks,
		and incoming DA blocks, respectively.
	*/
	retrieverMutex sync.Mutex

	// pendingBatch is the result of the last DA submission
	// that is pending settlement layer submission.
	// It is used to avoid double submission of the same batch.
	// It's protected by submitBatchMutex.
	pendingBatch *PendingBatch

	logger types.Logger

	// Cached blocks and commits for applying at future heights. Invariant: the block and commit are .Valid() (validated sigs etc)
	blockCache map[uint64]CachedBlock
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

	exec, err := NewExecutor(proposerAddress, conf.NamespaceID, genesis.ChainID, mempool, proxyApp, eventBus, logger)
	if err != nil {
		return nil, fmt.Errorf("create block executor: %w", err)
	}
	s, err := getInitialState(store, genesis, logger)
	if err != nil {
		return nil, fmt.Errorf("get initial state: %w", err)
	}

	agg := &Manager{
		Pubsub:      pubsub,
		p2pClient:   p2pClient,
		ProposerKey: proposerKey,
		Conf:        conf,
		Genesis:     genesis,
		LastState:   s,
		Store:       store,
		Executor:    exec,
		DAClient:    dalc,
		SLClient:    settlementClient,
		Retriever:   dalc.(da.BatchRetriever),
		// channels are buffered to avoid blocking on input/output operations, buffer sizes are arbitrary
		SyncTargetDiode:       diodes.NewOneToOne(1, nil),
		shouldProduceBlocksCh: make(chan bool, 1),
		produceEmptyBlockCh:   make(chan bool, 1),
		logger:                logger,
		blockCache:            make(map[uint64]CachedBlock),
	}

	return agg, nil
}

// Start starts the block manager.
func (m *Manager) Start(ctx context.Context, isAggregator bool) error {
	m.logger.Debug("SANITY CHECK ~~~~~~~~~~")

	m.logger.Info("Starting the block manager")

	// TODO (#283): set aggregator mode by proposer addr on the hub
	if isAggregator {
		// make sure local signing key is the registered on the hub
		slProposerKey := m.SLClient.GetProposer().PublicKey.Bytes()
		localProposerKey, _ := m.ProposerKey.GetPublic().Raw()
		if !bytes.Equal(slProposerKey, localProposerKey) {
			return fmt.Errorf("proposer key mismatch: settlement proposer key: %s, block manager proposer key: %s", slProposerKey, m.ProposerKey.GetPublic())
		}
		m.logger.Info("Starting in aggregator mode")
	}

	// Check if InitChain flow is needed
	if m.LastState.IsGenesis() {
		m.logger.Info("Running InitChain")

		err := m.RunInitChain(ctx)
		if err != nil {
			return err
		}
	}

	err := m.syncBlockManager()
	if err != nil {
		err = fmt.Errorf("sync block manager: %w", err)
		return err
	}

	if isAggregator {
		go uevent.MustSubscribe(ctx, m.Pubsub, "nodeHealth", events.QueryHealthStatus, m.onNodeHealthStatus, m.logger)
		go m.ProduceBlockLoop(ctx)
		go m.SubmitLoop(ctx)
	} else {
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyBlockLoop", p2p.EventQueryNewNewGossipedBlock, m.onNewGossipedBlock, m.logger, 100)
		go m.RetrieveLoop(ctx)
		go m.SyncTargetLoop(ctx)
	}

	return nil
}

// syncBlockManager enforces the node to be synced on initial run.
func (m *Manager) syncBlockManager() error {
	resultRetrieveBatch, err := m.getLatestBatchFromSL()
	// Set the syncTarget according to the result
	if err != nil {
		// TODO: separate between fresh rollapp and non-registered rollapp
		if errors.Is(err, settlement.ErrBatchNotFound) {
			// Since we requested the latest batch and got batch not found it means
			// the SL still hasn't got any batches for this chain.
			m.logger.Info("No batches for chain found in SL. Start writing first batch")
			m.SyncTarget.Store(uint64(m.Genesis.InitialHeight - 1))
			return nil
		}
		return err
	}
	m.SyncTarget.Store(resultRetrieveBatch.EndHeight)
	err = m.syncUntilTarget(resultRetrieveBatch.EndHeight)
	if err != nil {
		return err
	}

	m.logger.Info("Synced", "current height", m.Store.Height(), "syncTarget", m.SyncTarget.Load())
	return nil
}

// UpdateSyncParams updates the sync target and state index if necessary
func (m *Manager) UpdateSyncParams(endHeight uint64) {
	types.RollappHubHeightGauge.Set(float64(endHeight))
	m.logger.Info("Received new syncTarget", "syncTarget", endHeight)
	m.SyncTarget.Store(endHeight)
	m.lastSubmissionTime.Store(time.Now().UnixNano())
}

func getAddress(key crypto.PrivKey) ([]byte, error) {
	rawKey, err := key.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	return tmcrypto.AddressHash(rawKey), nil
}

func (m *Manager) onNodeHealthStatus(event pubsub.Message) {
	eventData := event.Data().(*events.DataHealthStatus)
	m.logger.Info("Received node health status event.", "eventData", eventData)
	m.shouldProduceBlocksCh <- eventData.Error == nil
}

// TODO: move to gossip.go
// onNewGossippedBlock will take a block and apply it
func (m *Manager) onNewGossipedBlock(event pubsub.Message) {
	m.logger.Debug("Received new block event", "eventData", event.Data(), "cachedBlocks", len(m.blockCache))
	eventData := event.Data().(p2p.GossipedBlock)
	block := eventData.Block
	commit := eventData.Commit

	nextHeight := m.Store.NextHeight()
	if block.Header.Height >= nextHeight {
		m.blockCache[block.Header.Height] = CachedBlock{
			Block:  &block,
			Commit: &commit,
		}
		m.logger.Debug("caching block", "block height", block.Header.Height, "store height", m.Store.Height())
	}

	if block.Header.Height == nextHeight {
		err := m.attemptApplyCachedBlocks()
		if err != nil {
			m.logger.Error("applying cached blocks", "err", err)
		}
	}
}

// getLatestBatchFromSL gets the latest batch from the SL
func (m *Manager) getLatestBatchFromSL() (*settlement.ResultRetrieveBatch, error) {
	return m.SLClient.RetrieveBatch()
}

// getInitialState tries to load lastState from Store, and if it's not available it reads GenesisDoc.
func getInitialState(store store.Store, genesis *tmtypes.GenesisDoc, logger types.Logger) (types.State, error) {
	s, err := store.LoadState()
	if errors.Is(err, types.ErrNoStateFound) {
		logger.Info("failed to find state in the store, creating new state from genesis")
		return types.NewFromGenesisDoc(genesis)
	}

	return s, err
}
