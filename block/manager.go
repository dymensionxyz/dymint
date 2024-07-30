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
	// It is ALSO used by the producer, because the producer needs to check if it can prune blocks and it wont'
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

	isSequencer := m.IsSequencer()
	m.logger.Info("sequencer mode", "isSequencer", isSequencer)

	eg, ctx := errgroup.WithContext(ctx)
	if !isSequencer {
		// Fullnode loop can start before syncing from DA
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyGossipedBlocksLoop", p2p.EventQueryNewNewGossipedBlock, m.onNewGossipedBlock, m.logger)

		err := m.syncBlockManager()
		if err != nil {
			return fmt.Errorf("sync block manager: %w", err)
		}

		eg.Go(func() error {
			return m.RetrieveLoop(ctx)
		})
		eg.Go(func() error {
			return m.SyncToTargetHeightLoop(ctx)
		})
		go func() {
			err := eg.Wait()
			m.logger.Info("Block manager err group finished.", "err", err)
		}()
		return nil
	}

	/* ----------------------------- sequencer mode ----------------------------- */
	err := m.syncBlockManager()
	if err != nil {
		return fmt.Errorf("sync block manager: %w", err)
	}

	// Sequencer must wait till DA is synced to start submitting blobs
	<-m.DAClient.Synced()

	// check if sequencer in the middle of rotation
	if m.MissingLastBatch() {
		next := m.SLClient.GetNextProposer()
		val, err := next.TMValidator()
		if err != nil {
			return err
		}

		go func() {
			err = m.CompleteRotation(ctx, val.Address)
			if err != nil {
				panic(err)
			}
			// TODO: graceful fallback to full node
			panic("sequencer is no longer the proposer")
		}()
		return nil
	}

	// populate the bytes produced channel
	bytesProducedC := make(chan int)
	nBytes := m.GetUnsubmittedBytes()
	go func() {
		bytesProducedC <- nBytes
	}()

	rotateSequencerC := make(chan []byte, 1)

	eg.Go(func() error {
		return m.SubmitLoop(ctx, bytesProducedC)
	})
	eg.Go(func() error {
		return m.ProduceBlockLoop(ctx, bytesProducedC)
	})
	eg.Go(func() error {
		rotateSequencerC <- m.MonitorSequencerRotation(ctx)
		return fmt.Errorf("sequencer rotation started. signal to stop production")
	})

	go func() {
		err := eg.Wait()

		// Check if exited due to sequencer rotation signal
		select {
		case nextSeqAddr := <-rotateSequencerC:
			m.logger.Info("Sequencer rotation started. Production stopped on this sequencer", "nextSeqAddr", nextSeqAddr)
			err := m.CompleteRotation(ctx, nextSeqAddr)
			if err != nil {
				panic(err)
			}
			// TODO: graceful fallback to full node
			panic("sequencer is no longer the proposer")
		default:
			m.logger.Info("Block manager err group finished.", "err", err)
		}
	}()

	return nil
}

// check if last batch needed

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

// get proposer pubkey
func (m *Manager) GetProposerPubKey() tmcrypto.PubKey {
	return m.State.ActiveSequencer.GetProposerPubKey()
}
