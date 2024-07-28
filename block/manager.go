package block

import (
	"bytes"
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

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
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
	blockCache map[uint64]CachedBlock
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
	localAddress, err := getAddress(localKey)
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
		blockCache:       make(map[uint64]CachedBlock),
	}

	err = m.LoadStateOnInit(store, genesis, logger)
	if err != nil {
		return nil, fmt.Errorf("get initial state: %w", err)
	}

	return m, nil
}

// Start starts the block manager.
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("Starting block manager")

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

	if !isSequencer {
		// Fullnode loop can start before syncing from DA
		go uevent.MustSubscribe(ctx, m.Pubsub, "applyGossipedBlocksLoop", p2p.EventQueryNewNewGossipedBlock, m.onNewGossipedBlock, m.logger)
	}

	err := m.syncBlockManager()
	if err != nil {
		return fmt.Errorf("sync block manager: %w", err)
	}

	eg, ctx := errgroup.WithContext(ctx)
	if isSequencer {
		rotateSequencerC := make(chan []byte, 1)
		bytesProducedC := make(chan int)
		// Sequencer must wait till DA is synced to start submitting blobs
		<-m.DAClient.Synced()
		nBytes := m.GetUnsubmittedBytes()
		go func() {
			bytesProducedC <- nBytes
		}()

		// check if sequencer in the middle of rotation
		proposer := m.SLClient.GetProposer()
		next := m.SLClient.IsRotating()
		// if next defined and it's not the same as the current proposer, start rotation
		if next != nil && proposer.SequencerAddress != next.SequencerAddress {
			val, err := next.TMValidator()
			if err != nil {
				return err
			}

			go func() {
				err = m.CreateAndPostLastBatch(ctx, val.Address)
				if err != nil {
					m.logger.Error("Create and post last batch failed.", "error", err)
				}

				// FIXME: fallback to full node
			}()

			return nil

		}

		eg.Go(func() error {
			return m.SubmitLoop(ctx, bytesProducedC)
		})
		eg.Go(func() error {
			return m.ProduceBlockLoop(ctx, bytesProducedC)
		})
		eg.Go(func() error {
			rotateSequencerC <- m.MonitorSequencerRotation(ctx)
			return fmt.Errorf("sequencer rotation started")
		})

		go func() {
			err := eg.Wait()

			// Check if sequencer needs to complete rotation
			var nextSeqAddr []byte
			select {
			case nextSeqAddr = <-rotateSequencerC:
			default:
			}
			// Check if sequencer needs to complete rotation
			if len(nextSeqAddr) > 0 {
				m.logger.Info("Sequencer rotation started. Production stopped on this sequencer", "nextSeqAddr", nextSeqAddr)
				err := m.CreateAndPostLastBatch(ctx, nextSeqAddr)
				if err != nil {
					m.logger.Error("Create and post last batch failed.", "error", err)
				}
				// FIXME: fallback to full node
				return
			}
			m.logger.Info("Block manager err group finished.", "err", err)
		}()
	} else {
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
	}

	return nil
}

func (m *Manager) IsSequencer() bool {
	expectedProposer := m.GetProposerPubKey().Bytes()
	localProposerKey, _ := m.LocalKey.GetPublic().Raw() //already validated on manager creation
	return bytes.Equal(expectedProposer, localProposerKey)
}

func (m *Manager) NextHeightToSubmit() uint64 {
	return m.LastSubmittedHeight.Load() + 1
}

// add bonded sequencers to the seqSet without changing the proposer
func (m *Manager) UpdateBondedSequencerSetFromSL() error {
	seqs, err := m.SLClient.GetSequencers()
	if err != nil {
		return err
	}
	newSet := m.State.ActiveSequencer.BondedSet.Copy()
	for _, seq := range seqs {
		tmPubKey, err := cryptocodec.ToTmPubKeyInterface(seq.PublicKey)
		if err != nil {
			return err
		}
		val := tmtypes.NewValidator(tmPubKey, 1)

		// check if not exists already
		if newSet.HasAddress(val.Address) {
			continue
		}

		newSet.Validators = append(newSet.Validators, val)
	}
	// update state on changes
	if len(newSet.Validators) != len(m.State.ActiveSequencer.BondedSet.Validators) {
		m.State.ActiveSequencer.SetBondedSet(newSet)
	}

	m.logger.Debug("Updated bonded sequencer set", "newSet", newSet)
	return nil
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
