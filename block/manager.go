package block

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"golang.org/x/sync/errgroup"

	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/indexers/txindex"
	"github.com/dymensionxyz/dymint/store"
	uerrors "github.com/dymensionxyz/dymint/utils/errors"
	uevent "github.com/dymensionxyz/dymint/utils/event"
	"github.com/dymensionxyz/dymint/version"

	"github.com/libp2p/go-libp2p/core/crypto"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

// Manager is responsible for aggregating transactions into blocks.
type Manager struct {
	logger types.Logger

	// Configuration
	Conf     config.BlockManagerConfig
	Genesis  *tmtypes.GenesisDoc
	LocalKey crypto.PrivKey

	// Store and execution
	Store    store.Store
	State    *types.State
	Executor ExecutorI

	// Clients and servers
	Pubsub    *pubsub.Server
	P2PClient *p2p.Client
	DAClient  da.DataAvailabilityLayerClient
	SLClient  settlement.ClientI

	/*
		Submission
	*/
	// The last height which was submitted to both sublayers, that we know of. When we produce new batches, we will
	// start at this height + 1.
	// It is ALSO used by the producer, because the producer needs to check if it can prune blocks and it won't
	// prune anything that might be submitted in the future. Therefore, it must be atomic.
	LastSubmittedHeight atomic.Uint64

	/*
		Retrieval
	*/
	// Protect against processing two blocks at once when there are two routines handling incoming gossiped blocks,
	// and incoming DA blocks, respectively.
	retrieverMu sync.Mutex
	// Protect against syncing twice from DA in case new batch is posted but it did not finish to sync yet.
	syncFromDaMu sync.Mutex
	Retriever    da.BatchRetriever
	// Cached blocks and commits for applying at future heights. The blocks may not be valid, because
	// we can only do full validation in sequential order.
	blockCache *Cache

	// TargetHeight holds the value of the current highest block seen from either p2p (probably higher) or the DA
	TargetHeight atomic.Uint64

	// Fraud handler
	FraudHandler FraudHandler

	// channel used to send the retain height to the pruning background loop
	pruningC chan int64

	// indexer
	indexerService *txindex.IndexerService
}

// NewManager creates new block Manager.
func NewManager(
	localKey crypto.PrivKey,
	conf config.NodeConfig,
	genesis *tmtypes.GenesisDoc,
	store store.Store,
	mempool mempool.Mempool,
	proxyApp proxy.AppConns,
	settlementClient settlement.ClientI,
	eventBus *tmtypes.EventBus,
	pubsub *pubsub.Server,
	p2pClient *p2p.Client,
	dalcKV *store.PrefixKV,
	indexerService *txindex.IndexerService,
	logger log.Logger,
) (*Manager, error) {
	localAddress, err := types.GetAddress(localKey)
	if err != nil {
		return nil, err
	}
	exec, err := NewExecutor(localAddress, genesis.ChainID, mempool, proxyApp, eventBus, logger)
	if err != nil {
		return nil, fmt.Errorf("create block executor: %w", err)
	}

	m := &Manager{
		Pubsub:         pubsub,
		P2PClient:      p2pClient,
		LocalKey:       localKey,
		Conf:           conf.BlockManagerConfig,
		Genesis:        genesis,
		Store:          store,
		Executor:       exec,
		SLClient:       settlementClient,
		indexerService: indexerService,
		logger:         logger.With("module", "block_manager"),
		blockCache: &Cache{
			cache: make(map[uint64]types.CachedBlock),
		},
		FraudHandler: nil,                  // TODO: create a default handler
		pruningC:     make(chan int64, 10), // use of buffered channel to avoid blocking applyBlock thread. In case channel is full, pruning will be skipped, but the retain height can be pruned in the next iteration.
	}

	err = m.LoadStateOnInit(store, genesis, logger)
	if err != nil {
		return nil, fmt.Errorf("get initial state: %w", err)
	}

	err = m.setDA(conf.DAConfig, dalcKV, logger)
	if err != nil {
		return nil, err
	}

	// validate configuration params and rollapp consensus params are in line
	err = m.ValidateConfigWithRollappParams()
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Start starts the block manager.
func (m *Manager) Start(ctx context.Context) error {
	// Check if InitChain flow is needed
	if m.State.IsGenesis() {
		m.logger.Info("Running InitChain")

		err := m.RunInitChain(ctx)
		if err != nil {
			return err
		}
	}

	// Check if the chain is halted
	err := m.isChainHalted()
	if err != nil {
		return err
	}

	isProposer := m.IsProposer()
	m.logger.Info("starting block manager", "proposer", isProposer)

	eg, ctx := errgroup.WithContext(ctx)
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.PruningLoop(ctx)
	})

	/* ----------------------------- full node mode ----------------------------- */
	if !isProposer {
		// Full-nodes can sync from DA but it is not necessary to wait for it, since it can sync from P2P as well in parallel.
		go func() {
			err := m.syncFromSettlement()
			if err != nil {
				m.logger.Error("sync block manager from settlement", "err", err)
			}
			// DA Sync. Subscribe to SL next batch events
			go uevent.MustSubscribe(ctx, m.Pubsub, "syncTargetLoop", settlement.EventQueryNewSettlementBatchAccepted, m.onNewStateUpdate, m.logger)
		}()

		// P2P Sync. Subscribe to P2P received blocks events
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyGossipedBlocksLoop", p2p.EventQueryNewGossipedBlock, m.OnReceivedBlock, m.logger)
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyBlockSyncBlocksLoop", p2p.EventQueryNewBlockSyncBlock, m.OnReceivedBlock, m.logger)
		return nil
	}

	/* ----------------------------- sequencer mode ----------------------------- */
	// Subscribe to batch events, to update last submitted height in case batch confirmation was lost. This could happen if the sequencer crash/restarted just after submitting a batch to the settlement and by the time we query the last batch, this batch wasn't accepted yet.
	go uevent.MustSubscribe(ctx, m.Pubsub, "updateSubmittedHeightLoop", settlement.EventQueryNewSettlementBatchAccepted, m.UpdateLastSubmittedHeight, m.logger)

	// Sequencer must wait till DA is synced to start submitting blobs
	<-m.DAClient.Synced()
	err = m.syncFromSettlement()
	if err != nil {
		return fmt.Errorf("sync block manager from settlement: %w", err)
	}
	// check if sequencer in the middle of rotation
	nextSeqAddr, missing, err := m.MissingLastBatch()
	if err != nil {
		return fmt.Errorf("checking if missing last batch: %w", err)
	}
	// if sequencer is in the middle of rotation, complete rotation instead of running the main loop
	if missing {
		m.handleRotationReq(ctx, nextSeqAddr)
		return nil
	}

	// populate the bytes produced channel
	bytesProducedC := make(chan int)

	// channel to signal sequencer rotation started
	rotateSequencerC := make(chan string, 1)

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SubmitLoop(ctx, bytesProducedC)
	})
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		bytesProducedC <- m.GetUnsubmittedBytes() // load unsubmitted bytes from previous run
		return m.ProduceBlockLoop(ctx, bytesProducedC)
	})
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.MonitorSequencerRotation(ctx, rotateSequencerC)
	})

	go func() {
		_ = eg.Wait()
		// Check if exited due to sequencer rotation signal
		select {
		case nextSeqAddr := <-rotateSequencerC:
			m.handleRotationReq(ctx, nextSeqAddr)
		default:
			m.logger.Info("Block manager err group finished.")
		}
	}()

	return nil
}

func (m *Manager) isChainHalted() error {
	if m.GetProposerPubKey() == nil {
		// if no proposer set in state, try to update it from the hub
		err := m.UpdateProposer()
		if err != nil {
			return fmt.Errorf("update proposer: %w", err)
		}
		if m.GetProposerPubKey() == nil {
			return fmt.Errorf("no proposer pubkey found. chain is halted")
		}
	}
	return nil
}

func (m *Manager) NextHeightToSubmit() uint64 {
	return m.LastSubmittedHeight.Load() + 1
}

// syncFromSettlement enforces the node to be synced on initial run from SL and DA.
func (m *Manager) syncFromSettlement() error {
	err := m.UpdateSequencerSetFromSL()
	if err != nil {
		return fmt.Errorf("update bonded sequencer set: %w", err)
	}

	res, err := m.SLClient.GetLatestBatch()
	if errors.Is(err, gerrc.ErrNotFound) {
		// The SL hasn't got any batches for this chain yet.
		m.logger.Info("No batches for chain found in SL.")
		m.LastSubmittedHeight.Store(uint64(m.Genesis.InitialHeight - 1))
		return nil
	}

	if err != nil {
		// TODO: separate between fresh rollapp and non-registered rollapp
		return err
	}
	m.LastSubmittedHeight.Store(res.EndHeight)
	err = m.syncToTargetHeight(res.EndHeight)
	m.UpdateTargetHeight(res.EndHeight)
	if err != nil {
		return err
	}

	m.logger.Info("Synced.", "current height", m.State.Height(), "last submitted height", m.LastSubmittedHeight.Load())
	return nil
}

func (m *Manager) GetProposerPubKey() tmcrypto.PubKey {
	return m.State.Sequencers.GetProposerPubKey()
}

func (m *Manager) UpdateTargetHeight(h uint64) {
	for {
		currentHeight := m.TargetHeight.Load()
		if m.TargetHeight.CompareAndSwap(currentHeight, max(currentHeight, h)) {
			break
		}
	}
}

// ValidateConfigWithRollappParams checks the configuration params are consistent with the params in the dymint state (e.g. DA and version)
func (m *Manager) ValidateConfigWithRollappParams() error {
	if version.Commit != m.State.RollappParams.Version {
		return fmt.Errorf("binary version mismatch. rollapp param: %s binary used:%s", m.State.RollappParams.Version, version.Commit)
	}

	if da.Client(m.State.RollappParams.Da) != m.DAClient.GetClientType() {
		return fmt.Errorf("da client mismatch. rollapp param: %s da configured: %s", m.State.RollappParams.Da, m.DAClient.GetClientType())
	}

	if m.Conf.BatchSubmitBytes > uint64(m.DAClient.GetMaxBlobSizeBytes()) {
		return fmt.Errorf("batch size above limit: batch size: %d limit: %d: DA %s", m.Conf.BatchSubmitBytes, m.DAClient.GetMaxBlobSizeBytes(), m.DAClient.GetClientType())
	}

	return nil
}

// setDA initializes DA client in blockmanager according to DA type set in genesis or stored in state
func (m *Manager) setDA(daconfig string, dalcKV store.KV, logger log.Logger) error {
	daLayer := m.State.RollappParams.Da
	dalc := registry.GetClient(daLayer)
	if dalc == nil {
		return fmt.Errorf("get data availability client named '%s'", daLayer)
	}

	err := dalc.Init([]byte(daconfig), m.Pubsub, dalcKV, logger.With("module", string(dalc.GetClientType())))
	if err != nil {
		return fmt.Errorf("data availability layer client initialization:  %w", err)
	}
	m.DAClient = dalc
	retriever, ok := dalc.(da.BatchRetriever)
	if !ok {
		return fmt.Errorf("data availability layer client is not of type BatchRetriever")
	}
	m.Retriever = retriever
	return nil
}
