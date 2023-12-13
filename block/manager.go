package block

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"code.cloudfoundry.org/go-diodes"

	"github.com/avast/retry-go/v4"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/utils"
	"github.com/libp2p/go-libp2p/core/crypto"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/tendermint/tendermint/proxy"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/state"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

type blockSource string

const (
	producedBlock blockSource = "produced"
	gossipedBlock blockSource = "gossip"
	daBlock       blockSource = "da"
)

type blockMetaData struct {
	source   blockSource
	daHeight uint64
}

// Manager is responsible for aggregating transactions into blocks.
type Manager struct {
	pubsub *pubsub.Server

	p2pClient *p2p.Client

	lastState types.State

	conf    config.BlockManagerConfig
	genesis *tmtypes.GenesisDoc

	proposerKey crypto.PrivKey

	store    store.Store
	executor *state.BlockExecutor

	dalc             da.DataAvailabilityLayerClient
	settlementClient settlement.LayerI
	retriever        da.BatchRetriever

	syncTargetDiode diodes.Diode

	shouldProduceBlocksCh chan bool
	produceEmptyBlockCh   chan bool

	syncTarget         uint64
	lastSubmissionTime int64
	batchInProcess     atomic.Value
	isSyncedCond       sync.Cond
	produceBlockMutex  sync.Mutex

	syncCache map[uint64]*types.Block

	logger log.Logger

	prevBlock    map[uint64]*types.Block
	prevCommit   map[uint64]*types.Commit
	prevMetaData map[uint64]blockMetaData
}

// getInitialState tries to load lastState from Store, and if it's not available it reads GenesisDoc.
func getInitialState(store store.Store, genesis *tmtypes.GenesisDoc, logger log.Logger) (types.State, error) {
	s, err := store.LoadState()
	if err == types.ErrNoStateFound {
		logger.Info("failed to find state in the store, creating new state from genesis")
		return types.NewFromGenesisDoc(genesis)
	}

	return s, err
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
	logger log.Logger,
) (*Manager, error) {

	proposerAddress, err := getAddress(proposerKey)
	if err != nil {
		return nil, err
	}

	exec, err := state.NewBlockExecutor(proposerAddress, conf.NamespaceID, genesis.ChainID, mempool, proxyApp, eventBus, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create block executor: %w", err)
	}
	s, err := getInitialState(store, genesis, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial state: %w", err)
	}

	validators := []*tmtypes.Validator{}

	if s.LastBlockHeight+1 == genesis.InitialHeight {
		sequencersList := settlementClient.GetSequencersList()
		for _, sequencer := range sequencersList {
			tmPubKey, err := cryptocodec.ToTmPubKeyInterface(sequencer.PublicKey)
			if err != nil {
				return nil, err
			}
			validators = append(validators, tmtypes.NewValidator(tmPubKey, 1))
		}

		res, err := exec.InitChain(genesis, validators)
		if err != nil {
			return nil, err
		}

		updateInitChainState(&s, res, validators)
		if _, err := store.UpdateState(s, nil); err != nil {
			return nil, err
		}
	}

	batchInProcess := atomic.Value{}
	batchInProcess.Store(false)

	agg := &Manager{
		pubsub:           pubsub,
		p2pClient:        p2pClient,
		proposerKey:      proposerKey,
		conf:             conf,
		genesis:          genesis,
		lastState:        s,
		store:            store,
		executor:         exec,
		dalc:             dalc,
		settlementClient: settlementClient,
		retriever:        dalc.(da.BatchRetriever),
		// channels are buffered to avoid blocking on input/output operations, buffer sizes are arbitrary
		syncTargetDiode:       diodes.NewOneToOne(1, nil),
		syncCache:             make(map[uint64]*types.Block),
		isSyncedCond:          *sync.NewCond(new(sync.Mutex)),
		batchInProcess:        batchInProcess,
		shouldProduceBlocksCh: make(chan bool, 1),
		produceEmptyBlockCh:   make(chan bool, 1),
		logger:                logger,
		prevBlock:             make(map[uint64]*types.Block),
		prevCommit:            make(map[uint64]*types.Commit),
		prevMetaData:          make(map[uint64]blockMetaData),
	}

	return agg, nil
}

// Start starts the block manager.
func (m *Manager) Start(ctx context.Context, isAggregator bool) error {
	m.logger.Info("Starting the block manager")
	if isAggregator {
		m.logger.Info("Starting in aggregator mode")
		// TODO(omritoptix): change to private methods
		go m.ProduceBlockLoop(ctx)
		go m.SubmitLoop(ctx)
	}
	// TODO(omritoptix): change to private methods
	go m.RetrieveLoop(ctx)
	go m.SyncTargetLoop(ctx)
	m.EventListener(ctx)

	return nil
}

func getAddress(key crypto.PrivKey) ([]byte, error) {
	rawKey, err := key.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	return tmcrypto.AddressHash(rawKey), nil
}

// EventListener registers events to callbacks.
func (m *Manager) EventListener(ctx context.Context) {
	go utils.SubscribeAndHandleEvents(ctx, m.pubsub, "nodeHealthStatusHandler", events.EventQueryHealthStatus, m.healthStatusEventCallback, m.logger)
	go utils.SubscribeAndHandleEvents(ctx, m.pubsub, "ApplyBlockLoop", p2p.EventQueryNewNewGossipedBlock, m.applyBlockCallback, m.logger, 100)

}

func (m *Manager) healthStatusEventCallback(event pubsub.Message) {
	eventData := event.Data().(*events.EventDataHealthStatus)
	m.logger.Info("Received health status event", "eventData", eventData)
	m.shouldProduceBlocksCh <- eventData.Healthy
}

func (m *Manager) applyBlockCallback(event pubsub.Message) {
	m.logger.Debug("Received new block event", "eventData", event.Data())
	eventData := event.Data().(p2p.GossipedBlock)

	block := eventData.Block
	commit := eventData.Commit

	err := m.applyBlock(context.Background(), &block, &commit, blockMetaData{source: gossipedBlock})

	if err != nil {
		m.logger.Debug("Failed to apply block", "err", err)
	}
}

// SetDALC is used to set DataAvailabilityLayerClient used by Manager.
// TODO(omritoptix): Remove this from here as it's only being used for tests.
func (m *Manager) SetDALC(dalc da.DataAvailabilityLayerClient) {
	m.dalc = dalc
	m.retriever = dalc.(da.BatchRetriever)
}

// getLatestBatchFromSL gets the latest batch from the SL
func (m *Manager) getLatestBatchFromSL(ctx context.Context) (*settlement.ResultRetrieveBatch, error) {
	var resultRetrieveBatch *settlement.ResultRetrieveBatch
	var err error
	// Get latest batch from SL
	m.logger.Info("getLatestBatchFromSL")
	err = retry.Do(
		func() error {
			resultRetrieveBatch, err = m.settlementClient.RetrieveBatch()
			if err != nil {
				return err
			}
			return nil
		},
		retry.LastErrorOnly(true),
		retry.Context(ctx),
		retry.Attempts(1),
	)
	if err != nil {
		return resultRetrieveBatch, err
	}
	return resultRetrieveBatch, nil

}

// TODO(omritoptix): possible remove this method from the manager
func updateInitChainState(s *types.State, res *abci.ResponseInitChain, validators []*tmtypes.Validator) {
	// If the app did not return an app hash, we keep the one set from the genesis doc in
	// the state. We don't set appHash since we don't want the genesis doc app hash
	// recorded in the genesis block. We should probably just remove GenesisDoc.AppHash.
	if len(res.AppHash) > 0 {
		copy(s.AppHash[:], res.AppHash)
	}

	//The validators after initChain must be greater than zero, otherwise this state is not loadable
	if len(validators) <= 0 {
		panic("Validators must be greater than zero")
	}

	if res.ConsensusParams != nil {
		params := res.ConsensusParams
		if params.Block != nil {
			s.ConsensusParams.Block.MaxBytes = params.Block.MaxBytes
			s.ConsensusParams.Block.MaxGas = params.Block.MaxGas
		}
		if params.Evidence != nil {
			s.ConsensusParams.Evidence.MaxAgeNumBlocks = params.Evidence.MaxAgeNumBlocks
			s.ConsensusParams.Evidence.MaxAgeDuration = params.Evidence.MaxAgeDuration
			s.ConsensusParams.Evidence.MaxBytes = params.Evidence.MaxBytes
		}
		if params.Validator != nil {
			// Copy params.Validator.PubkeyTypes, and set result's value to the copy.
			// This avoids having to initialize the slice to 0 values, and then write to it again.
			s.ConsensusParams.Validator.PubKeyTypes = append([]string{}, params.Validator.PubKeyTypes...)
		}
		if params.Version != nil {
			s.ConsensusParams.Version.AppVersion = params.Version.AppVersion
		}
		s.Version.Consensus.App = s.ConsensusParams.Version.AppVersion
	}
	// We update the last results hash with the empty hash, to conform with RFC-6962.
	copy(s.LastResultsHash[:], merkle.HashFromByteSlices(nil))

	// Set the validators in the state
	s.Validators = tmtypes.NewValidatorSet(validators).CopyIncrementProposerPriority(1)
	s.NextValidators = s.Validators.Copy()
	s.LastValidators = s.Validators.Copy()
}
