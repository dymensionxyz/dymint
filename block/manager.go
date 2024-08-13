package block

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"golang.org/x/sync/errgroup"

	"github.com/dymensionxyz/dymint/store"
	uerrors "github.com/dymensionxyz/dymint/utils/errors"
	uevent "github.com/dymensionxyz/dymint/utils/event"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/p2p"

	"github.com/tendermint/tendermint/proxy"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/mempool"
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
	Executor *Executor

	// Clients and servers
	Pubsub    *pubsub.Server
	p2pClient *p2p.Client
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
}

// NewManager creates new block Manager.
func NewManager(
	localKey crypto.PrivKey,
	conf config.BlockManagerConfig,
	genesis *tmtypes.GenesisDoc,
	store store.Store,
	mempool mempool.Mempool,
	proxyApp proxy.AppConns,
	dalc da.DataAvailabilityLayerClient,
	settlementClient settlement.ClientI,
	eventBus *tmtypes.EventBus,
	pubsub *pubsub.Server,
	p2pClient *p2p.Client,
	logger types.Logger,
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
		Pubsub:    pubsub,
		p2pClient: p2pClient,
		LocalKey:  localKey,
		Conf:      conf,
		Genesis:   genesis,
		Store:     store,
		Executor:  exec,
		DAClient:  dalc,
		SLClient:  settlementClient,
		Retriever: dalc.(da.BatchRetriever),
		logger:    logger,
		blockCache: &Cache{
			cache: make(map[uint64]types.CachedBlock),
		},
	}

	err = m.LoadStateOnInit(store, genesis, logger)
	if err != nil {
		return nil, fmt.Errorf("get initial state: %w", err)
	}

	return m, nil
}

// Start starts the block manager.
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("Starting the block manager")

	isSequencer, err := m.IsSequencerVerify()
	if err != nil {
		return err
	}

	m.logger.Info("Starting block manager", "isSequencer", isSequencer)

	// Check if InitChain flow is needed
	if m.State.IsGenesis() {
		m.logger.Info("Running InitChain")

		err := m.RunInitChain(ctx)
		if err != nil {
			return err
		}
	}

	if isSequencer {

		eg, ctx := errgroup.WithContext(ctx)

		// Sequencer must wait till DA is synced to start submitting blobs
		<-m.DAClient.Synced()

		err = m.syncFromSettlement()
		if err != nil {
			return fmt.Errorf("sync block manager from settlement: %w", err)
		}

		nBytes := m.GetUnsubmittedBytes()
		bytesProducedC := make(chan int)

		uerrors.ErrGroupGoLog(eg, m.logger, func() error {
			return m.SubmitLoop(ctx, bytesProducedC)
		})
		uerrors.ErrGroupGoLog(eg, m.logger, func() error {
			bytesProducedC <- nBytes
			return m.ProduceBlockLoop(ctx, bytesProducedC)
		})
		go func() {
			_ = eg.Wait() // errors are already logged
			m.logger.Info("Block manager err group finished.")
		}()

	} else {
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
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyGossipedBlocksLoop", p2p.EventQueryNewGossipedBlock, m.onReceivedBlock, m.logger)
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyBlockSyncBlocksLoop", p2p.EventQueryNewBlockSyncBlock, m.onReceivedBlock, m.logger)
	}

	return nil
}

func (m *Manager) IsSequencerVerify() (bool, error) {
	slProposerKey := m.SLClient.GetProposer().PublicKey.Bytes()
	localProposerKey, err := m.LocalKey.GetPublic().Raw()
	if err != nil {
		return false, fmt.Errorf("get local node public key: %w", err)
	}
	return bytes.Equal(slProposerKey, localProposerKey), nil
}

func (m *Manager) IsSequencer() bool {
	ret, _ := m.IsSequencerVerify()
	return ret
}

func (m *Manager) NextHeightToSubmit() uint64 {
	return m.LastSubmittedHeight.Load() + 1
}

// syncFromSettlement enforces the node to be synced on initial run from SL and DA.
func (m *Manager) syncFromSettlement() error {
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

func (m *Manager) UpdateTargetHeight(h uint64) {
	for {
		currentHeight := m.TargetHeight.Load()
		if m.TargetHeight.CompareAndSwap(currentHeight, max(currentHeight, h)) {
			break
		}
	}
}
