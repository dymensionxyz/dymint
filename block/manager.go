package block

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"code.cloudfoundry.org/go-diodes"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"golang.org/x/sync/errgroup"

	"github.com/dymensionxyz/dymint/store"
	uerrors "github.com/dymensionxyz/dymint/utils/errors"
	uevent "github.com/dymensionxyz/dymint/utils/event"

	"github.com/libp2p/go-libp2p/core/crypto"
	tmcrypto "github.com/tendermint/tendermint/crypto"
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
	Retriever   da.BatchRetriever
	// get the next target height to sync local state to
	targetSyncHeight diodes.Diode
	// TargetHeight holds the value of the current highest block seen from either p2p (probably higher) or the DA
	TargetHeight atomic.Uint64

	// Cached blocks and commits for applying at future heights. The blocks may not be valid, because
	// we can only do full validation in sequential order.
	blockCache *Cache
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
		Pubsub:           pubsub,
		P2PClient:        p2pClient,
		LocalKey:         localKey,
		Conf:             conf,
		Genesis:          genesis,
		Store:            store,
		Executor:         exec,
		DAClient:         dalc,
		SLClient:         settlementClient,
		Retriever:        dalc.(da.BatchRetriever),
		targetSyncHeight: diodes.NewOneToOne(1, nil),
		logger:           logger,
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
	m.logger.Debug("Starting block manager")

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

	isSequencer := m.IsSequencer()
	m.logger.Info("sequencer mode", "isSequencer", isSequencer)

	eg, ctx := errgroup.WithContext(ctx)

	/* ----------------------------- full node mode ----------------------------- */
	if !isSequencer {
		// Fullnode loop can start before syncing from DA
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyGossipedBlocksLoop", p2p.EventQueryNewNewGossipedBlock, m.onNewGossipedBlock, m.logger)

		err := m.syncBlockManager()
		if err != nil {
			return fmt.Errorf("sync block manager: %w", err)
		}

		uerrors.ErrGroupGoLog(eg, m.logger, func() error {
			return m.RetrieveLoop(ctx)
		})
		uerrors.ErrGroupGoLog(eg, m.logger, func() error {
			return m.SyncToTargetHeightLoop(ctx)
		})
		go func() {
			_ = eg.Wait()
			m.logger.Info("Block manager err group finished.")
		}()
		return nil
	}

	/* ----------------------------- sequencer mode ----------------------------- */
	err = m.syncBlockManager()
	if err != nil {
		return fmt.Errorf("sync block manager: %w", err)
	}

	// Sequencer must wait till DA is synced to start submitting blobs
	<-m.DAClient.Synced()

	// check if sequencer in the middle of rotation
	next, err := m.SLClient.IsRotationInProgress()
	if err != nil {
		return fmt.Errorf("check rotation in progress: %w", err)
	}
	if next != nil {
		go func() {
			m.handleRotationReq(ctx, next.SequencerAddress)
		}()
		return nil
	}

	// populate the bytes produced channel
	bytesProducedC := make(chan int)
	nBytes := m.GetUnsubmittedBytes()
	go func() {
		bytesProducedC <- nBytes
	}()

	// channel to signal sequencer rotation started
	rotateSequencerC := make(chan string, 1)

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SubmitLoop(ctx, bytesProducedC)
	})
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.ProduceBlockLoop(ctx, bytesProducedC)
	})
	eg.Go(func() error {
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

// syncBlockManager enforces the node to be synced on initial run.
func (m *Manager) syncBlockManager() error {
	err := m.UpdateBondedSequencerSetFromSL()
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
