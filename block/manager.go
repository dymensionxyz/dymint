package block

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/go-diodes"
	"github.com/avast/retry-go"
	abciconv "github.com/dymensionxyz/dymint/conv/abci"
	"github.com/libp2p/go-libp2p-core/crypto"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/libs/pubsub"
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

// defaultDABlockTime is used only if DABlockTime is not configured for manager
const defaultDABlockTime = 30 * time.Second

type newBlockEvent struct {
	block    *types.Block
	commit   *types.Commit
	daHeight uint64
}

// Manager is responsible for aggregating transactions into blocks.
type Manager struct {
	pubsub *pubsub.Server

	lastState types.State

	conf    config.BlockManagerConfig
	genesis *tmtypes.GenesisDoc

	proposerKey crypto.PrivKey

	store    store.Store
	executor *state.BlockExecutor

	dalc         da.DataAvailabilityLayerClient
	settlementlc settlement.LayerClient
	retriever    da.BatchRetriever
	// daHeight is the height of the latest processed DA block
	daHeight uint64

	// TODO(omritotpix): Remove the header sync fields
	syncTargetDiode diodes.Diode

	batchInProcess atomic.Value

	syncTarget   uint64
	isSyncedCond sync.Cond

	blockInCh chan newBlockEvent
	syncCache map[uint64]*types.Block

	logger log.Logger
}

// getInitialState tries to load lastState from Store, and if it's not available it reads GenesisDoc.
func getInitialState(store store.Store, genesis *tmtypes.GenesisDoc) (types.State, error) {
	s, err := store.LoadState()
	if err != nil {
		s, err = types.NewFromGenesisDoc(genesis)
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
	proxyApp proxy.AppConnConsensus,
	dalc da.DataAvailabilityLayerClient,
	settlementlc settlement.LayerClient,
	eventBus *tmtypes.EventBus,
	pubsub *pubsub.Server,
	logger log.Logger,
) (*Manager, error) {
	s, err := getInitialState(store, genesis)
	if err != nil {
		return nil, err
	}
	if s.DAHeight < conf.DAStartHeight {
		s.DAHeight = conf.DAStartHeight
	}

	proposerAddress, err := getAddress(proposerKey)
	if err != nil {
		return nil, err
	}
	// TODO(omritoptix): Think about the default batchSize and default DABlockTime proper location.
	if conf.DABlockTime == 0 {
		logger.Info("WARNING: using default DA block time", "DABlockTime", defaultDABlockTime)
		conf.DABlockTime = defaultDABlockTime
	}

	exec := state.NewBlockExecutor(proposerAddress, conf.NamespaceID, genesis.ChainID, mempool, proxyApp, eventBus, logger)
	if s.LastBlockHeight+1 == genesis.InitialHeight {
		res, err := exec.InitChain(genesis)
		if err != nil {
			return nil, err
		}

		updateState(&s, res)
		if err := store.UpdateState(s); err != nil {
			return nil, err
		}
	}

	batchInProcess := atomic.Value{}
	batchInProcess.Store(false)

	agg := &Manager{
		pubsub:       pubsub,
		proposerKey:  proposerKey,
		conf:         conf,
		genesis:      genesis,
		lastState:    s,
		store:        store,
		executor:     exec,
		dalc:         dalc,
		settlementlc: settlementlc,
		retriever:    dalc.(da.BatchRetriever), // TODO(tzdybal): do it in more gentle way (after MVP)
		daHeight:     s.DAHeight,
		// channels are buffered to avoid blocking on input/output operations, buffer sizes are arbitrary
		syncTargetDiode: diodes.NewOneToOne(1, nil),
		blockInCh:       make(chan newBlockEvent, 100),
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
func (m *Manager) getLatestBatchFromSL(ctx context.Context) (settlement.ResultRetrieveBatch, error) {
	var resultRetrieveBatch settlement.ResultRetrieveBatch
	var err error
	// TODO(omritoptix): Move to a separate function.
	// Get latest batch from SL
	err = retry.Do(
		func() error {
			resultRetrieveBatch, err = m.settlementlc.RetrieveBatch()
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

// waitForSync waits until we are synced before it unblocks.
func (m *Manager) waitForSync(ctx context.Context) error {
	resultRetrieveBatch, err := m.getLatestBatchFromSL(ctx)
	// Set the syncTarget according to the result
	if err == settlement.ErrBatchNotFound {
		// Since we requested the latest batch and got batch not found it means
		// the SL still hasn't got any batches for this chain.
		m.logger.Info("No batches for chain found in SL. Start writing first batch")
		atomic.StoreUint64(&m.syncTarget, uint64(m.store.Height()))
		return nil
	} else if err != nil {
		m.logger.Error("failed to retrieve batch from SL", "err", err)
		return err
	} else {
		atomic.StoreUint64(&m.syncTarget, resultRetrieveBatch.EndHeight)
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

// PublishBlockLoop is calling publishBlock in a loop as long as wer'e synced.
func (m *Manager) PublishBlockLoop(ctx context.Context) {
	// We want to wait until we are synced. After that, since there is no leader
	// election yet, and leader are elected manually, we will not be out of sync until
	// we are manually being replaced.
	err := m.waitForSync(ctx)
	if err != nil {
		m.logger.Error("failed to wait for sync", "err", err)
	}
	// If we get blockTime of 0 we'll just run publishBlock in a loop
	// vs waiting for ticks
	publishLoopCh := make(chan bool, 1)
	ticker := &time.Ticker{}
	if m.conf.BlockTime == 0 {
		publishLoopCh <- true
	} else {
		ticker = time.NewTicker(time.Duration(m.conf.BlockTime))
		defer ticker.Stop()
	}
	// The func to invoke upon block publish
	publishLoopFunc := func() {
		err := m.publishBlock(ctx)
		if err != nil {
			m.logger.Error("error while producing block", "error", err)
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			publishLoopFunc()
		case <-publishLoopCh:
			for {
				publishLoopFunc()
			}
		}

	}
}

// SyncTargetLoop is responsible for updating the syncTarget as read from the SL
// to a ring buffer which will later be used by retrieveLoop for actually syncing until this target
func (m *Manager) SyncTargetLoop(ctx context.Context) {
	m.logger.Info("Started sync target loop")
	ticker := time.NewTicker(m.conf.BatchSyncInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Cancel the previous running getLatestBatchFromSL if it still runs
			// during next tick. Currently we timeout after half the batchSyncInterval.
			batchContext, cancel := context.WithTimeout(ctx, m.conf.BatchSyncInterval/2)
			// Cancel the context to avoid a leak
			defer cancel()
			// Get the latest batch from the settlement layer
			resultRetrieveBatch, err := m.getLatestBatchFromSL(batchContext)
			if err != nil {
				m.logger.Error("error while retrieving batch", "error", err)
				continue
			}
			// Check if batch height is higher than current block height and if so, set the syncTarget atomically
			if resultRetrieveBatch.EndHeight > m.store.Height() {
				m.logger.Info("Setting syncTarget", "syncTarget", resultRetrieveBatch.EndHeight)
				atomic.StoreUint64(&m.syncTarget, resultRetrieveBatch.EndHeight)
				m.syncTargetDiode.Set(diodes.GenericDataType(&resultRetrieveBatch.EndHeight))
			}
		}
		// TODO(omritoptix): Listen to events from the settlement layer of new batches and udpate the syncTarget
	}
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
			// If we are synced then signal all processes waiting on isSyncedCond.
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
		resultRetrieveBatch, err := m.settlementlc.RetrieveBatch(currentHeight + 1)
		// TODO(omritoptix): Handle in case we get batchNotfound
		if err != nil {
			m.logger.Error("error while retrieving batch", "error", err)
			continue
		}
		err = m.processNextDABatch(resultRetrieveBatch.MetaData.DA.Height)
		if err != nil {
			m.logger.Error("error while processing next DA batch", "error", err)
			break
		}
		currentHeight = m.store.Height()
	}
}

// ApplyBlockLoop is responsible for applying blocks retrieved from the DA.
func (m *Manager) ApplyBlockLoop(ctx context.Context) {
	for {
		select {
		case blockEvent := <-m.blockInCh:
			block := blockEvent.block
			commit := blockEvent.commit
			daHeight := blockEvent.daHeight
			m.logger.Debug("block body retrieved from DALC",
				"height", block.Header.Height,
				"daHeight", daHeight,
				"hash", block.Hash(),
			)
			if block.Header.Height > m.store.Height() {
				m.logger.Info("Syncing block", "height", block.Header.Height)
				newState, responses, err := m.executor.ApplyBlock(ctx, m.lastState, block)
				if err != nil {
					m.logger.Error("failed to ApplyBlock", "error", err)
					continue
				}
				err = m.store.SaveBlock(block, commit)
				if err != nil {
					m.logger.Error("failed to save block", "error", err)
					continue
				}
				var appHash []byte
				appHash, _, err = m.executor.Commit(ctx, newState, block, responses)
				if err != nil {
					m.logger.Error("failed to Commit", "error", err)
					continue
				}
				m.store.SetHeight(block.Header.Height)

				err = m.store.SaveBlockResponses(block.Header.Height, responses)
				if err != nil {
					m.logger.Error("failed to save block responses", "error", err)
					continue
				}

				copy(newState.AppHash[:], appHash)
				newState.DAHeight = daHeight
				m.lastState = newState
				err = m.store.UpdateState(m.lastState)
				if err != nil {
					m.logger.Error("failed to save updated state", "error", err)
					continue
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) processNextDABatch(daHeight uint64) error {
	m.logger.Debug("trying to retrieve batch from DA", "daHeight", daHeight)
	batchResp, err := m.fetchBatch(daHeight)
	if err != nil {
		m.logger.Error("failed to retrieve batch from DA", "daHeight", daHeight, "error", err)
		return err
	}
	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "daHeight", daHeight)
	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			m.blockInCh <- newBlockEvent{block, batch.Commits[i], daHeight}
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

func (m *Manager) publishBlock(ctx context.Context) error {

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

	// Check if there's an already stored block at a newer height
	// If there is use that instead of creating a new block
	var commit *types.Commit
	pendingBlock, err := m.store.LoadBlock(newHeight)
	if err == nil {
		m.logger.Info("Using pending block", "height", newHeight)
		block = pendingBlock
	} else {
		m.logger.Info("Creating and publishing block", "height", newHeight)
		block = m.executor.CreateBlock(newHeight, lastCommit, lastHeaderHash, m.lastState)
		m.logger.Debug("block info", "num_tx", len(block.Data.Txs))

		abciHeaderPb := abciconv.ToABCIHeaderPB(&block.Header)
		headerBytes, err := abciHeaderPb.Marshal()
		if err != nil {
			return err
		}
		sign, err := m.proposerKey.Sign(headerBytes)
		if err != nil {
			return err
		}
		commit = &types.Commit{
			Height:     block.Header.Height,
			HeaderHash: block.Header.Hash(),
			Signatures: []types.Signature{sign},
		}

		// SaveBlock commits the DB tx
		err = m.store.SaveBlock(block, commit)
		if err != nil {
			return err
		}
	}

	// Apply the block but DONT commit
	newState, responses, err := m.executor.ApplyBlock(ctx, m.lastState, block)
	if err != nil {
		return err
	}

	// Commit the new state and block which writes to disk on the proxy app
	var appHash []byte
	appHash, _, err = m.executor.Commit(ctx, newState, block, responses)
	if err != nil {
		return err
	}

	// SaveBlockResponses commits the DB tx
	err = m.store.SaveBlockResponses(block.Header.Height, responses)
	if err != nil {
		return err
	}

	copy(newState.AppHash[:], appHash)
	newState.DAHeight = atomic.LoadUint64(&m.daHeight)
	// After this call m.lastState is the NEW state returned from ApplyBlock
	m.lastState = newState

	// UpdateState commits the DB tx
	err = m.store.UpdateState(m.lastState)
	if err != nil {
		return err
	}

	// SaveValidators commits the DB tx
	err = m.store.SaveValidators(block.Header.Height, m.lastState.Validators)
	if err != nil {
		return err
	}

	// Only update the stored height after successfully submitting to DA layer and committing to the DB
	m.store.SetHeight(block.Header.Height)

	syncTarget := atomic.LoadUint64(&m.syncTarget)
	currentBlockHeight := block.Header.Height
	if currentBlockHeight > syncTarget && currentBlockHeight-syncTarget >= m.conf.BlockBatchSize && m.batchInProcess.Load() == false {
		go m.submitNextBatch(ctx)
	}

	return nil
}

func (m *Manager) submitNextBatch(ctx context.Context) {
	defer m.batchInProcess.Store(false)
	m.batchInProcess.Store(true)
	// Get the batch start and end height
	startHeight := atomic.LoadUint64(&m.syncTarget) + 1
	endHeight := startHeight + m.conf.BlockBatchSize - 1
	m.logger.Info("Submitting next batch", "startHeight", startHeight, "endHeight", endHeight)
	// Create the batch
	nextBatch, err := m.createNextDABatch(startHeight, endHeight)
	if err != nil {
		m.logger.Error("Failed to create next batch", "startHeight", startHeight, "endHeight", endHeight, "error", err)
		return
	}
	// Submit batch to the DA
	resultSubmitToDA, err := m.submitBatchToDA(ctx, nextBatch)
	if err != nil {
		m.logger.Error("Failed to submit next batch to DA Layer", "startHeight", startHeight, "endHeight", endHeight, "error", err)
		return
	}
	// Submit batch to SL
	// TODO(omritoptix): Handle a case where the SL submission fails due to syncTarget out of sync. In that case
	// we'll want to update the syncTarget before returning.
	err = m.submitBatchToSL(ctx, nextBatch, resultSubmitToDA)
	if err != nil {
		m.logger.Error("Failed to submit next batch to SL Layer", "startHeight", startHeight, "endHeight", endHeight, "error", err)
		return
	}
	// Update the sync target
	atomic.StoreUint64(&m.syncTarget, endHeight)
}

func (m *Manager) createNextDABatch(startHeight uint64, endHeight uint64) (*types.Batch, error) {
	// Create the batch
	batch := &types.Batch{
		StartHeight: startHeight,
		EndHeight:   endHeight,
		Blocks:      make([]*types.Block, 0, m.conf.BlockBatchSize),
		Commits:     make([]*types.Commit, 0, m.conf.BlockBatchSize),
	}
	// Populate the batch
	for height := startHeight; height <= endHeight; height++ {
		m.logger.Debug("Adding element to batch", "height", height)
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
	}
	return batch, nil
}

func (m *Manager) submitBatchToSL(ctx context.Context, batch *types.Batch, resultSubmitToDA *da.ResultSubmitBatch) error {
	// Submit batch to SL
	err := retry.Do(func() error {
		resultSubmitToSL := m.settlementlc.SubmitBatch(batch, resultSubmitToDA)
		if resultSubmitToSL.Code != settlement.StatusSuccess {
			err := fmt.Errorf("failed to submit batch to SL layer: %s", resultSubmitToSL.Message)
			return err
		}
		return nil
	}, retry.Context(ctx), retry.LastErrorOnly(true))
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) submitBatchToDA(ctx context.Context, batch *types.Batch) (*da.ResultSubmitBatch, error) {
	var res da.ResultSubmitBatch
	err := retry.Do(func() error {
		res = m.dalc.SubmitBatch(batch)
		if res.Code != da.StatusSuccess {
			return fmt.Errorf("failed to submit batch to DA layer: %s", res.Message)
		}
		return nil
	}, retry.Context(ctx), retry.LastErrorOnly(true))
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// TODO(omritoptix): possible remove this method from the manager
func updateState(s *types.State, res *abci.ResponseInitChain) {
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

	if len(res.Validators) > 0 {
		vals, err := tmtypes.PB2TM.ValidatorUpdates(res.Validators)
		if err != nil {
			// TODO(tzdybal): handle error properly
			panic(err)
		}
		s.Validators = tmtypes.NewValidatorSet(vals)
		s.NextValidators = tmtypes.NewValidatorSet(vals).CopyIncrementProposerPriority(1)
	}
}
