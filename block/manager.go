package block

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"golang.org/x/sync/errgroup"

	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/dymensionxyz/dymint/indexers/txindex"
	"github.com/dymensionxyz/dymint/store"
	uerrors "github.com/dymensionxyz/dymint/utils/errors"
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

const (
	// RunModeProposer represents a node running as a proposer
	RunModeProposer uint = iota
	// RunModeFullNode represents a node running as a full node
	RunModeFullNode
)

// Manager is responsible for aggregating transactions into blocks.
type Manager struct {
	logger types.Logger

	// Configuration
	Conf            config.BlockManagerConfig
	Genesis         *tmtypes.GenesisDoc
	GenesisChecksum string
	LocalKey        crypto.PrivKey

	// Store and execution
	Store      store.Store
	State      *types.State
	Executor   ExecutorI
	Sequencers *types.SequencerSet // Sequencers is the set of sequencers that are currently active on the rollapp

	// Clients and servers
	Pubsub    *pubsub.Server
	P2PClient *p2p.Client
	DAClient  da.DataAvailabilityLayerClient
	SLClient  settlement.ClientI

	// RunMode represents the mode of the node. Set during initialization and shouldn't change after that.
	RunMode uint

	/*
		Sequencer and full-node
	*/
	// The last height which was submitted to settlement, that we know of. When we produce new batches, we will
	// start at this height + 1.
	// It is ALSO used by the producer, because the producer needs to check if it can prune blocks and it won't
	// prune anything that might be submitted in the future. Therefore, it must be atomic.
	LastSettlementHeight atomic.Uint64

	// channel used to send the retain height to the pruning background loop
	pruningC chan int64

	// indexer
	IndexerService *txindex.IndexerService

	// used to fetch blocks from DA. Sequencer will only fetch batches in case it requires to re-sync (in case of rollback). Full-node will fetch batches for syncing and validation.
	Retriever da.BatchRetriever

	/*
		Full-node only
	*/
	// Protect against processing two blocks at once when there are two routines handling incoming gossiped blocks,
	// and incoming DA blocks, respectively.
	retrieverMu sync.Mutex

	// Cached blocks and commits, coming from P2P, for applying at future heights. The blocks may not be valid, because
	// we can only do full validation in sequential order.
	blockCache *Cache

	// TargetHeight holds the value of the current highest block seen from either p2p (probably higher) or the DA
	TargetHeight atomic.Uint64

	// Fraud handler
	FraudHandler FraudHandler

	// channel used to signal the syncing loop when there is a new state update available
	settlementSyncingC chan struct{}

	// channel used to signal the validation loop when there is a new state update available
	settlementValidationC chan struct{}

	// notifies when the node has completed syncing
	syncedFromSettlement *uchannel.Nudger

	// validates all non-finalized state updates from settlement, checking there is consistency between DA and P2P blocks, and the information in the state update.
	SettlementValidator *SettlementValidator
}

// NewManager creates new block Manager.
func NewManager(
	localKey crypto.PrivKey,
	conf config.NodeConfig,
	genesis *tmtypes.GenesisDoc,
	genesisChecksum string,
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
		NewConsensusMsgQueue(), // TODO properly specify ConsensusMsgStream: https://github.com/dymensionxyz/dymint/issues/1125
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("create block executor: %w", err)
	}

	m := &Manager{
		Pubsub:          pubsub,
		P2PClient:       p2pClient,
		LocalKey:        localKey,
		Conf:            conf.BlockManagerConfig,
		Genesis:         genesis,
		GenesisChecksum: genesisChecksum,
		Store:           store,
		Executor:        exec,
		Sequencers:      types.NewSequencerSet(),
		SLClient:        settlementClient,
		IndexerService:  indexerService,
		logger:          logger.With("module", "block_manager"),
		blockCache: &Cache{
			cache: make(map[uint64]types.CachedBlock),
		},
		pruningC:              make(chan int64, 10),   // use of buffered channel to avoid blocking applyBlock thread. In case channel is full, pruning will be skipped, but the retain height can be pruned in the next iteration.
		settlementSyncingC:    make(chan struct{}, 1), // use of buffered channel to avoid blocking. In case channel is full, its skipped because there is an ongoing syncing process, but syncing height is updated, which means the ongoing syncing will sync to the new height.
		settlementValidationC: make(chan struct{}, 1), // use of buffered channel to avoid blocking. In case channel is full, its skipped because there is an ongoing validation process, but validation height is updated, which means the ongoing validation will validate to the new height.
		syncedFromSettlement:  uchannel.NewNudger(),   // used by the sequencer to wait  till the node completes the syncing from settlement.
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

	m.SettlementValidator = NewSettlementValidator(m.logger, m)

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

	// Check if a proposer on the rollapp is set. In case no proposer is set on the Rollapp, fallback to the hub proposer (If such exists).
	// No proposer on the rollapp means that at some point there was no available proposer.
	// In case there is also no proposer on the hub to our current height, it means that the chain is halted.
	if m.State.GetProposer() == nil {
		m.logger.Info("No proposer on the rollapp, fallback to the hub proposer, if available")
		SLProposer, err := m.SLClient.GetProposerAtHeight(int64(m.State.NextHeight()))
		if err != nil {
			return fmt.Errorf("get proposer at height: %w", err)
		}
		m.State.SetProposer(SLProposer)
	}

	// checks if the the current node is the proposer either on rollapp or on the hub.
	// In case of sequencer rotation, there's a phase where proposer rotated on Rollapp but hasn't yet rotated on hub.
	// for this case, 2 nodes will get `true` for `AmIProposer` so the l2 proposer can produce blocks and the hub proposer can submit his last batch.
	// The hub proposer, after sending the last state update, will panic and restart as full node.
	amIProposerOnSL, err := m.AmIProposerOnSL()
	if err != nil {
		return fmt.Errorf("am i proposer on SL: %w", err)
	}
	amIProposer := amIProposerOnSL || m.AmIProposerOnRollapp()

	// update local state from latest state in settlement
	err = m.updateFromLastSettlementState()
	if err != nil {
		return fmt.Errorf("sync block manager from settlement: %w", err)
	}

	// send signal to syncing loop with last settlement state update
	m.triggerSettlementSyncing()
	// send signal to validation loop with last settlement state update
	m.triggerSettlementValidation()

	eg, ctx := errgroup.WithContext(ctx)

	// Start the pruning loop in the background
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.PruningLoop(ctx)
	})

	// Start the settlement sync loop in the background
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SettlementSyncLoop(ctx)
	})

	// Monitor sequencer set updates
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.MonitorSequencerSetUpdates(ctx)
	})

	// run based on the node role
	if !amIProposer {
		return m.runAsFullNode(ctx, eg)
	}

	return m.runAsProposer(ctx, eg)
}

func (m *Manager) NextHeightToSubmit() uint64 {
	return m.LastSettlementHeight.Load() + 1
}

// updateFromLastSettlementState retrieves last sequencers and state update from the Hub and updates local state with it
func (m *Manager) updateFromLastSettlementState() error {
	// Update sequencers list from SL
	err := m.UpdateSequencerSetFromSL()
	if err != nil {
		// this error is not critical
		m.logger.Error("Cannot fetch sequencer set from the Hub", "error", err)
	}

	// update latest height from SL
	latestHeight, err := m.SLClient.GetLatestHeight()
	if errors.Is(err, gerrc.ErrNotFound) {
		// The SL hasn't got any batches for this chain yet.
		m.logger.Info("No batches for chain found in SL.")
		m.LastSettlementHeight.Store(uint64(m.Genesis.InitialHeight - 1))
		return nil
	}

	if err != nil {
		// TODO: separate between fresh rollapp and non-registered rollapp
		return err
	}

	m.LastSettlementHeight.Store(latestHeight)

	if latestHeight >= m.State.NextHeight() {
		m.UpdateTargetHeight(latestHeight)
	}

	return nil
}

func (m *Manager) updateLastFinalizedHeightFromSettlement() error {
	// update latest finalized height from SL
	height, err := m.SLClient.GetLatestFinalizedHeight()
	if errors.Is(err, gerrc.ErrNotFound) {
		m.logger.Info("No finalized batches for chain found in SL.")
	} else if err != nil {
		return fmt.Errorf("getting finalized height. err: %w", err)
	}
	m.SettlementValidator.UpdateLastValidatedHeight(height)

	return nil
}

func (m *Manager) GetProposerPubKey() tmcrypto.PubKey {
	return m.State.GetProposerPubKey()
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
	drsVersion, err := strconv.ParseUint(version.DrsVersion, 10, 32)
	if err != nil {
		return fmt.Errorf("unable to parse drs version")
	}
	if uint32(drsVersion) != m.State.RollappParams.DrsVersion {
		return fmt.Errorf("DRS version mismatch. rollapp param: %d binary used:%d", m.State.RollappParams.DrsVersion, drsVersion)
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
