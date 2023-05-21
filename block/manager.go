package block

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/go-diodes"
	"github.com/avast/retry-go"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	abciconv "github.com/dymensionxyz/dymint/conv/abci"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

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

// defaultDABlockTime is used only if DABlockTime is not configured for manager
var (
	DABatchRetryDelay = 20 * time.Second
	SLBatchRetryDelay = 10 * time.Second
	maxDelay          = 1 * time.Minute
)

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

	batchInProcess   atomic.Value
	batchRetryCtx    context.Context
	batchRetryCancel context.CancelFunc
	batchRetryMu     sync.RWMutex

	syncTarget   uint64
	isSyncedCond sync.Cond

	syncCache map[uint64]*types.Block

	logger log.Logger
}

// getInitialState tries to load lastState from Store, and if it's not available it reads GenesisDoc.
func getInitialState(store store.Store, genesis *tmtypes.GenesisDoc) (types.State, error) {
	s, err := store.LoadState()
	if err == nil {
		return s, nil
	}

	return types.NewFromGenesisDoc(genesis)
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

	// TODO(mtsitrin): Probably should be validated and manage default on config init
	// TODO(omritoptix): Think about the default batchSize and default DABlockTime proper location.
	if conf.DABlockTime == 0 {
		logger.Info("WARNING: using default DA block time", "DABlockTime", config.DefaultNodeConfig.DABlockTime)
		conf.DABlockTime = config.DefaultNodeConfig.DABlockTime
	}

	if conf.BlockBatchSizeBytes == 0 {
		logger.Info("WARNING: using default DA batch size bytes limit", "BlockBatchSizeBytes", config.DefaultNodeConfig.BlockBatchSizeBytes)
		conf.BlockBatchSizeBytes = config.DefaultNodeConfig.BlockBatchSizeBytes
	}

	exec := state.NewBlockExecutor(proposerAddress, conf.NamespaceID, genesis.ChainID, mempool, proxyApp, eventBus, logger)

	s, err := getInitialState(store, genesis)
	if err != nil {
		return nil, err
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
		syncTargetDiode: diodes.NewOneToOne(1, nil),
		syncCache:       make(map[uint64]*types.Block),
		isSyncedCond:    *sync.NewCond(new(sync.Mutex)),
		batchInProcess:  batchInProcess,
		logger:          logger,
	}

	return agg, nil
}

func getAddress(key crypto.PrivKey) ([]byte, error) {
	rawKey, err := key.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	return tmcrypto.AddressHash(rawKey), nil
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

// waitForSync enforces the aggregator to be synced before it can produce blocks.
// It requires the retriveBlockLoop to be running.
func (m *Manager) waitForSync(ctx context.Context) error {
	resultRetrieveBatch, err := m.getLatestBatchFromSL(ctx)
	// Set the syncTarget according to the result
	if err == settlement.ErrBatchNotFound {
		// Since we requested the latest batch and got batch not found it means
		// the SL still hasn't got any batches for this chain.
		m.logger.Info("No batches for chain found in SL. Start writing first batch")
		atomic.StoreUint64(&m.syncTarget, uint64(m.genesis.InitialHeight-1))
		return nil
	} else if err != nil {
		m.logger.Error("failed to retrieve batch from SL", "err", err)
		return err
	} else {
		m.updateSyncParams(ctx, resultRetrieveBatch.EndHeight)
	}
	// Wait until isSynced is true and then call the PublishBlockLoop
	m.isSyncedCond.L.Lock()
	// Wait until we're synced and that we have got the latest batch (if we didn't, m.syncTarget == 0)
	// before we start publishing blocks
	for m.store.Height() < atomic.LoadUint64(&m.syncTarget) {
		m.logger.Info("Waiting for sync", "current height", m.store.Height(), "syncTarget", atomic.LoadUint64(&m.syncTarget))
		m.isSyncedCond.Wait()
	}
	m.isSyncedCond.L.Unlock()
	m.logger.Info("Synced, Starting to produce", "current height", m.store.Height(), "syncTarget", atomic.LoadUint64(&m.syncTarget))
	return nil
}

// ProduceBlockLoop is calling publishBlock in a loop as long as wer'e synced.
func (m *Manager) ProduceBlockLoop(ctx context.Context) {
	// We want to wait until we are synced. After that, since there is no leader
	// election yet, and leader are elected manually, we will not be out of sync until
	// we are manually being replaced.
	err := m.waitForSync(ctx)
	if err != nil {
		m.logger.Error("failed to wait for sync", "err", err)
	}
	// If we get blockTime of 0 we'll just run publishBlock in a loop
	// vs waiting for ticks
	produceBlockCh := make(chan bool, 1)
	ticker := &time.Ticker{}
	if m.conf.BlockTime == 0 {
		produceBlockCh <- true
	} else {
		ticker = time.NewTicker(m.conf.BlockTime)
		defer ticker.Stop()
	}
	// The func to invoke upon block publish
	produceBlockLoop := func() {
		err := m.produceBlock(ctx)
		if err != nil {
			m.logger.Error("error while producing block", "error", err)
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			produceBlockLoop()
		case <-produceBlockCh:
			for {
				produceBlockLoop()
			}
		}

	}
}

// SyncTargetLoop is responsible for getting real time updates about batches submission.
// for non aggregator: updating the sync target which will be used by retrieveLoop to sync until this target.
// for aggregator: get notification that batch has been accepted so can send next batch.
func (m *Manager) SyncTargetLoop(ctx context.Context) {
	m.logger.Info("Started sync target loop")
	subscription, err := m.pubsub.Subscribe(ctx, "syncTargetLoop", settlement.EventQueryNewSettlementBatchAccepted)
	if err != nil {
		m.logger.Error("failed to subscribe to state update events")
		panic(err)
	}
	// First time we start we want to get the latest batch from the SL
	resultRetrieveBatch, err := m.getLatestBatchFromSL(ctx)
	if err != nil {
		m.logger.Error("failed to retrieve batch from SL", "err", err)
	} else {
		m.updateSyncParams(ctx, resultRetrieveBatch.EndHeight)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-subscription.Out():
			m.logger.Info("Received state update event", "eventData", event.Data())
			eventData := event.Data().(*settlement.EventDataNewSettlementBatchAccepted)
			m.updateSyncParams(ctx, eventData.EndHeight)
			// In case we are the aggregator and we've got an update, then we can stop blocking from
			// the next batches to be published. For non-aggregators this is not needed.
			// We only want to send the next once the previous has been published successfully.
			// TODO(omritoptix): Once we have leader election, we can add a condition.
			// Update batch accepted is only relevant for the aggregator
			// TODO(omritoptix): Check if we are the aggregator
			m.updateBatchAccepted()
		case <-subscription.Cancelled():
			m.logger.Info("Subscription canceled")
		}
	}
}

// updateSyncParams updates the sync target and state index if necessary
func (m *Manager) updateSyncParams(ctx context.Context, endHeight uint64) {
	m.logger.Info("Received new syncTarget", "syncTarget", endHeight)
	atomic.StoreUint64(&m.syncTarget, endHeight)
	m.syncTargetDiode.Set(diodes.GenericDataType(&endHeight))
}

func (m *Manager) updateBatchAccepted() {
	m.batchRetryMu.Lock()
	if m.batchRetryCtx != nil && m.batchRetryCtx.Err() == nil {
		m.batchRetryCancel()
	}
	m.batchRetryMu.Unlock()
	m.batchInProcess.Store(false)
}

// RetriveLoop listens for new sync messages written to a ring buffer and in turn
// runs syncUntilTarget on the latest message in the ring buffer.
func (m *Manager) RetriveLoop(ctx context.Context) {
	m.logger.Info("Started retrieve loop")
	syncTargetpoller := diodes.NewPoller(m.syncTargetDiode)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get only the latest sync target
			syncTarget := syncTargetpoller.Next()
			m.syncUntilTarget(ctx, *(*uint64)(syncTarget))
			// Check if after we sync we are synced or a new syncTarget was already set.
			// If we are synced then signal all goroutines waiting on isSyncedCond.
			if m.store.Height() >= atomic.LoadUint64(&m.syncTarget) {
				m.logger.Info("Synced at height", "height", m.store.Height())
				m.isSyncedCond.L.Lock()
				m.isSyncedCond.Signal()
				m.isSyncedCond.L.Unlock()
			}
		}
	}
}

// syncUntilTarget syncs the block until the syncTarget is reached.
// It fetches the batches from the settlement, gets the DA height and gets
// the actual blocks from the DA.
func (m *Manager) syncUntilTarget(ctx context.Context, syncTarget uint64) {
	currentHeight := m.store.Height()
	for currentHeight < syncTarget {
		m.logger.Info("Syncing until target", "current height", currentHeight, "syncTarget", syncTarget)
		resultRetrieveBatch, err := m.settlementClient.RetrieveBatch(atomic.LoadUint64(&m.lastState.SLStateIndex) + 1)
		if err != nil {
			m.logger.Error("Failed to sync until target. error while retrieving batch", "error", err)
			continue
		}
		err = m.processNextDABatch(ctx, resultRetrieveBatch.MetaData.DA.Height)
		if err != nil {
			m.logger.Error("Failed to sync until target. error while processing next DA batch", "error", err)
			break
		}
		err = m.updateStateIndex(resultRetrieveBatch.StateIndex)
		if err != nil {
			return
		}
		currentHeight = m.store.Height()
	}
}

// ApplyBlockLoop is responsible for applying blocks retrieved from pubsub server.
func (m *Manager) ApplyBlockLoop(ctx context.Context) {
	subscription, err := m.pubsub.Subscribe(ctx, "ApplyBlockLoop", p2p.EventQueryNewNewGossipedBlock, 100)
	if err != nil {
		m.logger.Error("failed to subscribe to gossiped blocked events")
		panic(err)
	}
	for {
		select {
		case blockEvent := <-subscription.Out():
			m.logger.Debug("Received new block event", "eventData", blockEvent.Data())
			eventData := blockEvent.Data().(p2p.GossipedBlock)
			block := eventData.Block
			commit := eventData.Commit
			err := m.applyBlock(ctx, &block, &commit, blockMetaData{source: gossipedBlock})
			if err != nil {
				continue
			}
		case <-ctx.Done():
			return
		case <-subscription.Cancelled():
			m.logger.Info("Subscription for gossied blocked events canceled")
		}
	}
}

// applyBlock applies the block to the store and the abci app.
// steps: save block -> execute block with app -> update state -> commit block to app -> update store height and state hash.
// As the entire process can't be atomic we need to make sure the following condition apply before
// we're applying the block in the happy path: block height - 1 == abci app last block height.
// In case the following doesn't hold true, it means we crashed after the commit and before updating the store height.
// In that case we'll want to align the store with the app state and continue to the next block.
func (m *Manager) applyBlock(ctx context.Context, block *types.Block, commit *types.Commit, blockMetaData blockMetaData) error {
	if block.Header.Height == m.store.Height()+1 {
		m.logger.Info("Applying block", "height", block.Header.Height, "source", blockMetaData.source)

		// Check if alignment is needed due to incosistencies between the store and the app.
		isAlignRequired, err := m.alignStoreWithApp(ctx, block)
		if err != nil {
			return err
		}
		if isAlignRequired {
			m.logger.Info("Aligned with app state required. Skipping to next block", "height", block.Header.Height)
			return nil
		}
		// Start applying the block assuming no inconsistency was found.
		_, err = m.store.SaveBlock(block, commit, nil)
		if err != nil {
			m.logger.Error("Failed to save block", "error", err)
			return err
		}

		responses, err := m.executeBlock(ctx, block, commit)
		if err != nil {
			m.logger.Error("Failed to execute block", "error", err)
			return err
		}

		newState, err := m.executor.UpdateStateFromResponses(responses, m.lastState, block)
		if err != nil {
			return err
		}

		batch := m.store.NewBatch()

		batch, err = m.store.SaveBlockResponses(block.Header.Height, responses, batch)
		if err != nil {
			batch.Discard()
			return err
		}

		m.lastState = newState
		batch, err = m.store.UpdateState(m.lastState, batch)
		if err != nil {
			batch.Discard()
			return err
		}
		batch, err = m.store.SaveValidators(block.Header.Height, m.lastState.Validators, batch)
		if err != nil {
			batch.Discard()
			return err
		}

		err = batch.Commit()
		if err != nil {
			m.logger.Error("Failed to persist batch to disk", "error", err)
			return err
		}

		// Commit block to app
		err = m.executor.Commit(ctx, &newState, block, responses)
		if err != nil {
			m.logger.Error("Failed to commit to the block", "error", err)
			return err
		}

		// Update the state with the new app hash, last validators and store height from the commit.
		// Every one of those, if happens before commit, prevents us from re-executing the block in case failed during commit.
		newState.LastValidators = m.lastState.Validators.Copy()
		newState.LastStoreHeight = block.Header.Height
		_, err = m.store.UpdateState(newState, nil)
		if err != nil {
			m.logger.Error("Failed to update state", "error", err)
			return err
		}
		m.lastState = newState

		m.store.SetHeight(block.Header.Height)

	}
	return nil
}

// alignStoreWithApp is responsible for aligning the state of the store and the abci app if necessary.
func (m *Manager) alignStoreWithApp(ctx context.Context, block *types.Block) (bool, error) {
	isRequired := false
	// Validate incosistency in height wasn't caused by a crash and if so handle it.
	proxyAppInfo, err := m.executor.GetAppInfo()
	if err != nil {
		m.logger.Error("Failed to get app info", "error", err)
		return isRequired, err
	}
	if uint64(proxyAppInfo.LastBlockHeight) == block.Header.Height {
		isRequired = true
		m.logger.Info("Skipping block application and only updating store height and state hash", "height", block.Header.Height)
		// update the state with the hash, last store height and last validators.
		m.lastState.AppHash = *(*[32]byte)(proxyAppInfo.LastBlockAppHash)
		m.lastState.LastStoreHeight = block.Header.Height
		m.lastState.LastValidators = m.lastState.Validators.Copy()
		_, err := m.store.UpdateState(m.lastState, nil)
		if err != nil {
			m.logger.Error("Failed to update state", "error", err)
			return isRequired, err
		}
		m.store.SetHeight(block.Header.Height)
		return isRequired, nil
	}
	return isRequired, nil
}

func (m *Manager) executeBlock(ctx context.Context, block *types.Block, commit *types.Commit) (*tmstate.ABCIResponses, error) {
	// Currently we're assuming proposer is never nil as it's a pre-condition for
	// dymint to start
	proposer := m.settlementClient.GetProposer()

	if err := m.executor.Validate(m.lastState, block, commit, proposer); err != nil {
		return &tmstate.ABCIResponses{}, err
	}

	responses, err := m.executor.Execute(ctx, m.lastState, block)
	if err != nil {
		return &tmstate.ABCIResponses{}, err
	}

	return responses, nil
}

func (m *Manager) gossipBlock(ctx context.Context, block types.Block, commit types.Commit) error {
	gossipedBlock := p2p.GossipedBlock{Block: block, Commit: commit}
	gossipedBlockBytes, err := gossipedBlock.MarshalBinary()
	if err != nil {
		m.logger.Error("Failed to marshal block", "error", err)
		return err
	}
	if err := m.p2pClient.GossipBlock(ctx, gossipedBlockBytes); err != nil {
		m.logger.Error("Failed to gossip block", "error", err)
		return err
	}
	return nil

}

func (m *Manager) processNextDABatch(ctx context.Context, daHeight uint64) error {
	m.logger.Debug("trying to retrieve batch from DA", "daHeight", daHeight)
	batchResp, err := m.fetchBatch(daHeight)
	if err != nil {
		m.logger.Error("failed to retrieve batch from DA", "daHeight", daHeight, "error", err)
		return err
	}
	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "daHeight", daHeight)
	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			err := m.applyBlock(ctx, block, batch.Commits[i], blockMetaData{source: daBlock, daHeight: daHeight})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) fetchBatch(daHeight uint64) (da.ResultRetrieveBatch, error) {
	var err error
	batchRes := m.retriever.RetrieveBatches(daHeight)
	switch batchRes.Code {
	case da.StatusError:
		err = fmt.Errorf("failed to retrieve batch: %s", batchRes.Message)
	case da.StatusTimeout:
		err = fmt.Errorf("timeout during retrieve batch: %s", batchRes.Message)
	}
	return batchRes, err
}

func (m *Manager) produceBlock(ctx context.Context) error {
	var lastCommit *types.Commit
	var lastHeaderHash [32]byte
	var err error
	height := m.store.Height()
	newHeight := height + 1

	// this is a special case, when first block is produced - there is no previous commit
	if newHeight == uint64(m.genesis.InitialHeight) {
		lastCommit = &types.Commit{Height: height, HeaderHash: [32]byte{}}
	} else {
		lastCommit, err = m.store.LoadCommit(height)
		if err != nil {
			return fmt.Errorf("error while loading last commit: %w", err)
		}
		lastBlock, err := m.store.LoadBlock(height)
		if err != nil {
			return fmt.Errorf("error while loading last block: %w", err)
		}
		lastHeaderHash = lastBlock.Header.Hash()
	}

	var block *types.Block
	// Check if there's an already stored block and commit at a newer height
	// If there is use that instead of creating a new block
	var commit *types.Commit
	pendingBlock, err := m.store.LoadBlock(newHeight)
	if err == nil {
		m.logger.Info("Using pending block", "height", newHeight)
		block = pendingBlock
		commit, err = m.store.LoadCommit(newHeight)
		if err != nil {
			m.logger.Error("Loaded block but failed to load commit", "height", newHeight, "error", err)
			return err
		}
	} else {
		m.logger.Info("Creating block", "height", newHeight)
		block = m.executor.CreateBlock(newHeight, lastCommit, lastHeaderHash, m.lastState)
		m.logger.Debug("block info", "num_tx", len(block.Data.Txs))

		abciHeaderPb := abciconv.ToABCIHeaderPB(&block.Header)
		abciHeaderBytes, err := abciHeaderPb.Marshal()
		if err != nil {
			return err
		}
		sign, err := m.proposerKey.Sign(abciHeaderBytes)
		if err != nil {
			return err
		}
		commit = &types.Commit{
			Height:     block.Header.Height,
			HeaderHash: block.Header.Hash(),
			Signatures: []types.Signature{sign},
		}

	}

	// Gossip the block as soon as it is produced
	if err := m.gossipBlock(ctx, *block, *commit); err != nil {
		return err
	}

	if err := m.applyBlock(ctx, block, commit, blockMetaData{source: producedBlock}); err != nil {
		return err
	}

	// Submit batch if we've reached the batch size and there isn't another batch currently in submission process.
	// SyncTarget is the height of the last block in the last batch as seen by this node.
	syncTarget := atomic.LoadUint64(&m.syncTarget)
	if block.Header.Height-syncTarget >= m.conf.BlockBatchSize && m.batchInProcess.Load() == false {
		m.batchInProcess.Store(true)
		go m.submitNextBatch(ctx)
	}

	return nil
}

func (m *Manager) submitNextBatch(ctx context.Context) {
	// Get the batch start and end height
	startHeight := atomic.LoadUint64(&m.syncTarget) + 1
	endHeight := startHeight + m.conf.BlockBatchSize - 1
	// Create the batch
	nextBatch, err := m.createNextDABatch(startHeight, endHeight)
	if err != nil {
		m.logger.Error("Failed to create next batch", "startHeight", startHeight, "endHeight", endHeight, "error", err)
		return
	}

	actualEndHeight := nextBatch.EndHeight

	// Submit batch to the DA
	m.logger.Info("Submitting next batch", "startHeight", startHeight, "endHeight", actualEndHeight, "size", nextBatch.ToProto().Size())
	resultSubmitToDA, err := m.submitBatchToDA(ctx, nextBatch)
	if err != nil {
		m.logger.Error("Failed to submit next batch to DA Layer", "startHeight", startHeight, "endHeight", actualEndHeight, "error", err)
		panic("Failed to submit next batch to DA Layer")
	}

	// Submit batch to SL
	// TODO(omritoptix): Handle a case where the SL submission fails due to syncTarget out of sync with the latestHeight in the SL.
	// In that case we'll want to update the syncTarget before returning.
	m.submitBatchToSL(nextBatch, resultSubmitToDA)
}

func (m *Manager) updateStateIndex(stateIndex uint64) error {
	atomic.StoreUint64(&m.lastState.SLStateIndex, stateIndex)
	_, err := m.store.UpdateState(m.lastState, nil)
	if err != nil {
		m.logger.Error("Failed to update state", "error", err)
		return err
	}
	return nil
}

func (m *Manager) createNextDABatch(startHeight uint64, endHeight uint64) (*types.Batch, error) {
	var height uint64
	// Create the batch
	batchSize := endHeight - startHeight + 1
	batch := &types.Batch{
		StartHeight: startHeight,
		EndHeight:   endHeight,
		Blocks:      make([]*types.Block, 0, batchSize),
		Commits:     make([]*types.Commit, 0, batchSize),
	}

	// Populate the batch
	for height = startHeight; height <= endHeight; height++ {
		block, err := m.store.LoadBlock(height)
		if err != nil {
			m.logger.Error("Failed to load block", "height", height)
			return nil, err
		}
		commit, err := m.store.LoadCommit(height)
		if err != nil {
			m.logger.Error("Failed to load commit", "height", height)
			return nil, err
		}

		batch.Blocks = append(batch.Blocks, block)
		batch.Commits = append(batch.Commits, commit)

		//Check if the batch size is too big
		totalSize := batch.ToProto().Size()
		if totalSize > int(m.conf.BlockBatchSizeBytes) {
			// Nil out the last block and commit
			batch.Blocks[len(batch.Blocks)-1] = nil
			batch.Commits[len(batch.Commits)-1] = nil

			// Remove the last block and commit from the batch
			batch.Blocks = batch.Blocks[:len(batch.Blocks)-1]
			batch.Commits = batch.Commits[:len(batch.Commits)-1]
			break
		}
	}

	batch.EndHeight = height - 1
	return batch, nil
}

func (m *Manager) submitBatchToSL(batch *types.Batch, resultSubmitToDA *da.ResultSubmitBatch) {
	var resultSubmitToSL *settlement.ResultSubmitBatch
	m.batchRetryMu.Lock()
	m.batchRetryCtx, m.batchRetryCancel = context.WithCancel(context.Background())
	m.batchRetryMu.Unlock()
	defer m.batchRetryCancel()
	// Submit batch to SL
	err := retry.Do(func() error {
		resultSubmitToSL = m.settlementClient.SubmitBatch(batch, m.dalc.GetClientType(), resultSubmitToDA)
		if resultSubmitToSL.Code != settlement.StatusSuccess {
			m.logger.Error("failed to submit batch to SL layer", "startHeight", batch.StartHeight, "endHeight", batch.EndHeight, "error", resultSubmitToSL.Message)
			err := fmt.Errorf("failed to submit batch to SL layer: %s", resultSubmitToSL.Message)
			return err
		}
		return nil
	}, retry.Context(m.batchRetryCtx), retry.LastErrorOnly(true), retry.Delay(SLBatchRetryDelay), retry.MaxDelay(maxDelay))
	// Panic if we failed not due to context cancellation
	m.batchRetryMu.Lock()
	if err != nil && m.batchRetryCtx.Err() == nil {
		m.logger.Error("Failed to submit batch to SL Layer", "startHeight", batch.StartHeight, "endHeight", batch.EndHeight, "error", err)
		panic(err)
	}
	m.batchRetryMu.Unlock()
}

func (m *Manager) submitBatchToDA(ctx context.Context, batch *types.Batch) (*da.ResultSubmitBatch, error) {
	var res da.ResultSubmitBatch
	err := retry.Do(func() error {
		res = m.dalc.SubmitBatch(batch)
		if res.Code != da.StatusSuccess {
			m.logger.Error("failed to submit batch to DA layer", "startHeight", batch.StartHeight, "endHeight", batch.EndHeight, "error", res.Message)
			return fmt.Errorf("failed to submit batch to DA layer: %s", res.Message)
		}
		return nil
	}, retry.Context(ctx), retry.LastErrorOnly(true), retry.Delay(DABatchRetryDelay), retry.MaxDelay(maxDelay))
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// TODO(omritoptix): possible remove this method from the manager
func updateInitChainState(s *types.State, res *abci.ResponseInitChain, validators []*tmtypes.Validator) {
	// If the app did not return an app hash, we keep the one set from the genesis doc in
	// the state. We don't set appHash since we don't want the genesis doc app hash
	// recorded in the genesis block. We should probably just remove GenesisDoc.AppHash.
	if len(res.AppHash) > 0 {
		copy(s.AppHash[:], res.AppHash)
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

	if len(validators) > 0 {
		s.Validators = tmtypes.NewValidatorSet(validators)
		s.NextValidators = tmtypes.NewValidatorSet(validators).CopyIncrementProposerPriority(1)
	}
}
