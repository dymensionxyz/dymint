package block

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/dymensionxyz/dymint/gerr"
	"github.com/dymensionxyz/dymint/store"

	uevent "github.com/dymensionxyz/dymint/utils/event"

	"code.cloudfoundry.org/go-diodes"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/tendermint/tendermint/proxy"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

// Manager is responsible for aggregating transactions into blocks.
type Manager struct {
	// Configuration
	Conf        config.BlockManagerConfig
	Genesis     *tmtypes.GenesisDoc
	ProposerKey crypto.PrivKey

	// Store and execution
	Store    store.Store
	State    *types.State
	Executor *Executor

	// Clients and servers
	Pubsub    *pubsub.Server
	p2pClient *p2p.Client
	DAClient  da.DataAvailabilityLayerClient
	SLClient  settlement.LayerI

	// Data retrieval
	Retriever       da.BatchRetriever
	SyncTargetDiode diodes.Diode
	SyncTarget      atomic.Uint64

	// Block production
	producedSizeCh chan uint64 // channel for the producer to report the size of the block it produced

	// Submitter
	AccumulatedBatchSize atomic.Uint64

	/*
		Protect against processing two blocks at once when there are two routines handling incoming gossiped blocks,
		and incoming DA blocks, respectively.
	*/
	retrieverMutex sync.Mutex

	logger types.Logger

	// Cached blocks and commits for applying at future heights. The blocks may not be valid, because
	// we can only do full validation in sequential order.
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
		Pubsub:          pubsub,
		p2pClient:       p2pClient,
		ProposerKey:     proposerKey,
		Conf:            conf,
		Genesis:         genesis,
		State:           s,
		Store:           store,
		Executor:        exec,
		DAClient:        dalc,
		SLClient:        settlementClient,
		Retriever:       dalc.(da.BatchRetriever),
		SyncTargetDiode: diodes.NewOneToOne(1, nil),
		producedSizeCh:  make(chan uint64),
		logger:          logger,
		blockCache:      make(map[uint64]CachedBlock),
	}

	return agg, nil
}

// Start starts the block manager.
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("Starting the block manager")

	slProposerKey := m.SLClient.GetProposer().PublicKey.Bytes()
	localProposerKey, err := m.ProposerKey.GetPublic().Raw()
	if err != nil {
		return fmt.Errorf("get local node public key: %w", err)
	}

	isSequencer := bytes.Equal(slProposerKey, localProposerKey)
	m.logger.Info("Starting block manager", "isSequencer", isSequencer)

	// Check if InitChain flow is needed
	if m.State.IsGenesis() {
		m.logger.Info("Running InitChain")

		err := m.RunInitChain(ctx)
		if err != nil {
			return err
		}
	}

	if !isSequencer {
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyGossipedBlocksLoop", p2p.EventQueryNewNewGossipedBlock, m.onNewGossipedBlock, m.logger)
	}

	err = m.syncBlockManager()
	if err != nil {
		return fmt.Errorf("sync block manager: %w", err)
	}

	if isSequencer {
		// TODO: populate the accumulatedSize on startup
		go m.ProduceBlockLoop(ctx)
		go m.SubmitLoop(ctx)
	} else {
		go m.RetrieveLoop(ctx)
		go m.SyncTargetLoop(ctx)
	}

	return nil
}

// syncBlockManager enforces the node to be synced on initial run.
func (m *Manager) syncBlockManager() error {
	res, err := m.SLClient.RetrieveBatch()
	if errors.Is(err, gerr.ErrNotFound) {
		// The SL hasn't got any batches for this chain yet.
		m.logger.Info("No batches for chain found in SL.")
		m.SyncTarget.Store(uint64(m.Genesis.InitialHeight - 1))
		return nil
	}
	if err != nil {
		// TODO: separate between fresh rollapp and non-registered rollapp
		return err
	}
	// Set the syncTarget according to the result
	m.SyncTarget.Store(res.EndHeight)
	err = m.syncUntilTarget(res.EndHeight)
	if err != nil {
		return err
	}

	m.logger.Info("Synced.", "current height", m.State.Height(), "syncTarget", m.SyncTarget.Load())
	return nil
}

// UpdateSyncParams updates the sync target and state index if necessary
func (m *Manager) UpdateSyncParams(endHeight uint64) {
	types.RollappHubHeightGauge.Set(float64(endHeight))
	m.logger.Info("SyncTarget updated", "syncTarget", endHeight)
	m.SyncTarget.Store(endHeight)
}
