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
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/store"
	uerrors "github.com/dymensionxyz/dymint/utils/errors"
	uevent "github.com/dymensionxyz/dymint/utils/event"

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
	RunModeProposer uint = iota

	RunModeFullNode
)

type Manager struct {
	logger types.Logger

	Conf            config.BlockManagerConfig
	Genesis         *tmtypes.GenesisDoc
	GenesisChecksum string
	LocalKey        crypto.PrivKey
	RootDir         string

	Store      store.Store
	State      *types.State
	Executor   ExecutorI
	Sequencers *types.SequencerSet

	Pubsub    *pubsub.Server
	P2PClient *p2p.Client
	DAClient  da.DataAvailabilityLayerClient
	SLClient  settlement.ClientI

	RunMode uint

	Cancel context.CancelFunc
	Ctx    context.Context

	LastBlockTimeInSettlement atomic.Int64

	LastBlockTime atomic.Int64

	forkMu sync.Mutex

	LastSettlementHeight atomic.Uint64

	pruningC chan int64

	IndexerService *txindex.IndexerService

	Retriever da.BatchRetriever

	retrieverMu sync.Mutex

	blockCache *Cache

	TargetHeight atomic.Uint64

	FraudHandler FraudHandler

	settlementSyncingC chan struct{}

	settlementValidationC chan struct{}

	syncedFromSettlement *uchannel.Nudger

	SettlementValidator *SettlementValidator
}

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
		NewConsensusMsgQueue(),
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("create block executor: %w", err)
	}

	m := &Manager{
		Pubsub:          pubsub,
		P2PClient:       p2pClient,
		LocalKey:        localKey,
		RootDir:         conf.RootDir,
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
		pruningC:              make(chan int64, 10),
		settlementSyncingC:    make(chan struct{}, 1),
		settlementValidationC: make(chan struct{}, 1),
		syncedFromSettlement:  uchannel.NewNudger(),
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

	err = m.updateStateForNextRevision()
	if err != nil {
		return nil, err
	}

	err = m.ValidateConfigWithRollappParams()
	if err != nil {
		return nil, err
	}

	m.SettlementValidator = NewSettlementValidator(m.logger, m)

	return m, nil
}

func (m *Manager) Start(ctx context.Context) error {
	m.Ctx, m.Cancel = context.WithCancel(ctx)

	if m.State.IsGenesis() {
		m.logger.Info("Running InitChain")

		err := m.RunInitChain()
		if err != nil {
			return err
		}
	}

	if m.State.GetProposer() == nil {
		m.logger.Info("No proposer on the rollapp, fallback to the hub proposer, if available")
		err := m.UpdateProposerFromSL()
		if err != nil {
			return err
		}
		_, err = m.Store.SaveState(m.State, nil)
		if err != nil {
			return err
		}
	}

	amIProposerOnSL, err := m.AmIProposerOnSL()
	if err != nil {
		return fmt.Errorf("am i proposer on SL: %w", err)
	}

	amIProposer := amIProposerOnSL || m.AmIProposerOnRollapp()

	m.logger.Info("starting block manager", "mode", map[bool]string{true: "proposer", false: "full node"}[amIProposer])

	err = m.updateFromLastSettlementState()
	if err != nil {
		return fmt.Errorf("sync block manager from settlement: %w", err)
	}

	m.triggerSettlementSyncing()

	m.triggerSettlementValidation()

	eg, ctx := errgroup.WithContext(m.Ctx)

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.PruningLoop(ctx)
	})

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SettlementSyncLoop(ctx)
	})

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.MonitorSequencerSetUpdates(ctx)
	})

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.MonitorForkUpdateLoop(ctx)
	})

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.MonitorBalances(ctx)
	})

	if !amIProposer {
		return m.runAsFullNode(ctx, eg)
	}

	return m.runAsProposer(ctx, eg)
}

func (m *Manager) NextHeightToSubmit() uint64 {
	return m.LastSettlementHeight.Load() + 1
}

func (m *Manager) updateFromLastSettlementState() error {
	err := m.UpdateSequencerSetFromSL()
	if err != nil {
		m.logger.Error("Cannot fetch sequencer set from the Hub", "error", err)
	}

	latestHeight, err := m.SLClient.GetLatestHeight()
	if errors.Is(err, gerrc.ErrNotFound) {

		m.logger.Info("No batches for chain found in SL.")
		m.LastSettlementHeight.Store(uint64(m.Genesis.InitialHeight - 1))
		m.LastBlockTimeInSettlement.Store(m.Genesis.GenesisTime.UTC().UnixNano())
		return nil
	}
	if err != nil {
		return err
	}

	m.P2PClient.UpdateLatestSeenHeight(latestHeight)
	if latestHeight >= m.State.NextHeight() {
		m.UpdateTargetHeight(latestHeight)
	}

	m.LastSettlementHeight.Store(latestHeight)

	m.SetLastBlockTimeInSettlementFromHeight(latestHeight)

	block, err := m.Store.LoadBlock(m.State.Height())
	if err == nil {
		m.LastBlockTime.Store(block.Header.GetTimestamp().UTC().UnixNano())
	}
	return nil
}

func (m *Manager) updateLastFinalizedHeightFromSettlement() error {
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

func (m *Manager) GetRevision() uint64 {
	return m.State.GetRevision()
}

func (m *Manager) UpdateTargetHeight(h uint64) {
	for {
		currentHeight := m.TargetHeight.Load()
		if m.TargetHeight.CompareAndSwap(currentHeight, max(currentHeight, h)) {
			break
		}
	}
}

func (m *Manager) ValidateConfigWithRollappParams() error {
	if da.Client(m.State.RollappParams.Da) != m.DAClient.GetClientType() {
		return fmt.Errorf("da client mismatch. rollapp param: %s da configured: %s", m.State.RollappParams.Da, m.DAClient.GetClientType())
	}

	if m.Conf.BatchSubmitBytes > uint64(m.DAClient.GetMaxBlobSizeBytes()) {
		return fmt.Errorf("batch size above limit: batch size: %d limit: %d: DA %s", m.Conf.BatchSubmitBytes, m.DAClient.GetMaxBlobSizeBytes(), m.DAClient.GetClientType())
	}

	return nil
}

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

func (m *Manager) setFraudHandler(handler *FreezeHandler) {
	m.FraudHandler = handler
}

func (m *Manager) freezeNode(err error) {
	m.logger.Info("Freezing node", "err", err)
	if m.Ctx.Err() != nil {
		return
	}
	uevent.MustPublish(m.Ctx, m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
	m.Cancel()
}

func (m *Manager) SetLastBlockTimeInSettlementFromHeight(lastSettlementHeight uint64) {
	block, err := m.Store.LoadBlock(lastSettlementHeight)
	if err != nil {
		return
	}
	m.LastBlockTimeInSettlement.Store(block.Header.GetTimestamp().UTC().UnixNano())
}
