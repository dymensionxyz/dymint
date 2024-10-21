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

	uchannel "github.com/dymensionxyz/dymint/utils/channel"
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

	Retriever da.BatchRetriever
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

	syncingC chan struct{}

	validateC chan struct{}

	synced *uchannel.Nudger

	validator *StateUpdateValidator

	lastValidatedHeight atomic.Uint64
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
	exec, err := NewExecutor(
		localAddress,
		genesis.ChainID,
		mempool,
		proxyApp,
		eventBus,
		nil, // TODO add ConsensusMessagesStream
		logger,
	)
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
		pruningC:  make(chan int64, 10), // use of buffered channel to avoid blocking applyBlock thread. In case channel is full, pruning will be skipped, but the retain height can be pruned in the next iteration.
		syncingC:  make(chan struct{}, 1),
		validateC: make(chan struct{}, 1),
		synced:    uchannel.NewNudger(),
	}
	m.setFraudHandler(NewFreezeHandler(m))

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

	m.validator = NewStateUpdateValidator(m.logger, m)

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

	// listen to new bonded sequencers events to add them in the sequencer set
	go uevent.MustSubscribe(ctx, m.Pubsub, "newBondedSequencer", settlement.EventQueryNewBondedSequencer, m.UpdateSequencerSet, m.logger)
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SyncLoop(ctx)
	})

	err = m.syncFromSettlement()
	if err != nil {
		return fmt.Errorf("sync block manager from settlement: %w", err)
	}

	/* ----------------------------- full node mode ----------------------------- */
	if !isProposer {

		uerrors.ErrGroupGoLog(eg, m.logger, func() error {
			return m.ValidateLoop(ctx)
		})
		go uevent.MustSubscribe(ctx, m.Pubsub, "syncLoop", settlement.EventQueryNewSettlementBatchAccepted, m.onNewStateUpdate, m.logger)
		go uevent.MustSubscribe(ctx, m.Pubsub, "validateLoop", settlement.EventQueryNewSettlementBatchFinalized, m.onNewStateUpdateFinalized, m.logger)

		// P2P Sync. Subscribe to P2P received blocks events
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyGossipedBlocksLoop", p2p.EventQueryNewGossipedBlock, m.OnReceivedBlock, m.logger)
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyBlockSyncBlocksLoop", p2p.EventQueryNewBlockSyncBlock, m.OnReceivedBlock, m.logger)

		return nil
	}

	/* ----------------------------- sequencer mode ----------------------------- */
	// Subscribe to batch events, to update last submitted height in case batch confirmation was lost. This could happen if the sequencer crash/restarted just after submitting a batch to the settlement and by the time we query the last batch, this batch wasn't accepted yet.
	go uevent.MustSubscribe(ctx, m.Pubsub, "updateSubmittedHeightLoop", settlement.EventQueryNewSettlementBatchAccepted, m.UpdateLastSubmittedHeight, m.logger)

	// Sequencer must wait till DA is synced to start submitting blobs
	m.DAClient.WaitForSyncing()

	// Sequencer must wait till node is synced till last submittedHeight, in case it is not
	m.waitForSyncing()
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
	// Update sequencers list from SL
	err := m.UpdateSequencerSetFromSL()
	if err != nil {
		m.logger.Error("update bonded sequencer set", "error", err)
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
	m.UpdateTargetHeight(res.EndHeight)

	// get the latest finalized height to know from where to start validating
	err = m.UpdateFinalizedHeight()
	if err != nil {
		return err
	}

	// try to sync to last state update submitted on startup
	m.triggerStateUpdateSyncing()
	// try to validate all pending state updates on startup
	m.triggerStateUpdateValidation()

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

// UpdateFinalizedHeight retrieves the latest finalized batch and updates validation height with it
func (m *Manager) UpdateFinalizedHeight() error {
	res, err := m.SLClient.GetLatestFinalizedBatch()
	if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
		// The SL hasn't got any batches for this chain yet.
		return fmt.Errorf("getting finalized height. err: %w", err)
	}
	if errors.Is(err, gerrc.ErrNotFound) {
		// The SL hasn't got any batches for this chain yet.
		m.logger.Info("No finalized batches for chain found in SL.")
	} else {
		// update validation height with latest finalized height (it will be updated only of finalized height is higher)
		m.UpdateLastValidatedHeight(res.EndHeight)
	}
	return nil
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

// setFraudHandler sets the fraud handler for the block manager.
func (m *Manager) setFraudHandler(handler *FreezeHandler) {
	m.FraudHandler = handler
}
