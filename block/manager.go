package block

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"
	"go.uber.org/multierr"

	"github.com/celestiaorg/optimint/config"
	"github.com/celestiaorg/optimint/da"
	"github.com/celestiaorg/optimint/log"
	"github.com/celestiaorg/optimint/mempool"
	"github.com/celestiaorg/optimint/state"
	"github.com/celestiaorg/optimint/store"
	"github.com/celestiaorg/optimint/types"
)

// defaultDABlockTime is used only if DABlockTime is not configured for manager
const defaultDABlockTime = 30 * time.Second

// maxSubmitAttempts defines how many times Optimint will re-try to publish block to DA layer.
// This is temporary solution. It will be removed in future versions.
const maxSubmitAttempts = 30

// initialBackoff defines initial value for block submission backoff
var initialBackoff = 100 * time.Millisecond

type newBlockEvent struct {
	block    *types.Block
	daHeight uint64
}

// Manager is responsible for aggregating transactions into blocks.
type Manager struct {
	lastState types.State

	conf    config.BlockManagerConfig
	genesis *tmtypes.GenesisDoc

	proposerKey crypto.PrivKey

	store    store.Store
	executor *state.BlockExecutor

	dalc      da.DataAvailabilityLayerClient
	retriever da.BatchRetriever
	// daHeight is the height of the latest processed DA block
	daHeight uint64

	HeaderOutCh chan *types.Header
	HeaderInCh  chan *types.Header

	syncTarget uint64
	blockInCh  chan newBlockEvent
	syncCache  map[uint64]*types.Block

	// retrieveMtx is used by retrieveCond
	retrieveMtx *sync.Mutex
	// retrieveCond is used to notify sync goroutine (SyncLoop) that it needs to retrieve data
	retrieveCond *sync.Cond

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
	eventBus *tmtypes.EventBus,
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

	agg := &Manager{
		proposerKey: proposerKey,
		conf:        conf,
		genesis:     genesis,
		lastState:   s,
		store:       store,
		executor:    exec,
		dalc:        dalc,
		retriever:   dalc.(da.BatchRetriever), // TODO(tzdybal): do it in more gentle way (after MVP)
		daHeight:    s.DAHeight,
		// channels are buffered to avoid blocking on input/output operations, buffer sizes are arbitrary
		HeaderOutCh: make(chan *types.Header, 100),
		HeaderInCh:  make(chan *types.Header, 100),
		blockInCh:   make(chan newBlockEvent, 100),
		retrieveMtx: new(sync.Mutex),
		syncCache:   make(map[uint64]*types.Block),
		logger:      logger,
	}
	agg.retrieveCond = sync.NewCond(agg.retrieveMtx)

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
func (m *Manager) SetDALC(dalc da.DataAvailabilityLayerClient) {
	m.dalc = dalc
	m.retriever = dalc.(da.BatchRetriever)
}

// AggregationLoop is responsible for aggregating transactions into rollup-blocks.
func (m *Manager) AggregationLoop(ctx context.Context) {
	timer := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			start := time.Now()
			err := m.publishBlock(ctx)
			if err != nil {
				m.logger.Error("error while publishing block", "error", err)
			}
			timer.Reset(m.getRemainingSleep(start))
		}
	}
}

// SyncLoop is responsible for syncing blocks.
//
// SyncLoop processes headers gossiped in P2p network to know what's the latest block height,
// block data is retrieved from DA layer.
func (m *Manager) SyncLoop(ctx context.Context) {
	daTicker := time.NewTicker(m.conf.DABlockTime)
	for {
		select {
		case <-daTicker.C:
			m.retrieveCond.Signal()
		case header := <-m.HeaderInCh:
			m.logger.Debug("block header received", "height", header.Height, "hash", header.Hash())
			newHeight := header.Height
			currentHeight := m.store.Height()
			// in case of client reconnecting after being offline
			// newHeight may be significantly larger than currentHeight
			// it's handled gently in RetrieveLoop
			if newHeight > currentHeight {
				atomic.StoreUint64(&m.syncTarget, newHeight)
				m.retrieveCond.Signal()
			}
		case blockEvent := <-m.blockInCh:
			block := blockEvent.block
			daHeight := blockEvent.daHeight
			m.logger.Debug("block body retrieved from DALC",
				"height", block.Header.Height,
				"daHeight", daHeight,
				"hash", block.Hash(),
			)
			m.syncCache[block.Header.Height] = block
			m.retrieveCond.Signal()
			currentHeight := m.store.Height() // TODO(tzdybal): maybe store a copy in memory
			b1, ok1 := m.syncCache[currentHeight+1]
			b2, ok2 := m.syncCache[currentHeight+2]
			if ok1 && ok2 {
				m.logger.Info("Syncing block", "height", b1.Header.Height)
				newState, responses, err := m.executor.ApplyBlock(ctx, m.lastState, b1)
				if err != nil {
					m.logger.Error("failed to ApplyBlock", "error", err)
					continue
				}
				err = m.store.SaveBlock(b1, &b2.LastCommit)
				if err != nil {
					m.logger.Error("failed to save block", "error", err)
					continue
				}
				_, _, err = m.executor.Commit(ctx, newState, b1, responses)
				if err != nil {
					m.logger.Error("failed to Commit", "error", err)
					continue
				}
				m.store.SetHeight(b1.Header.Height)

				err = m.store.SaveBlockResponses(b1.Header.Height, responses)
				if err != nil {
					m.logger.Error("failed to save block responses", "error", err)
					continue
				}

				newState.DAHeight = daHeight
				m.lastState = newState
				err = m.store.UpdateState(m.lastState)
				if err != nil {
					m.logger.Error("failed to save updated state", "error", err)
					continue
				}
				delete(m.syncCache, currentHeight+1)
			}
		case <-ctx.Done():
			return
		}
	}
}

// RetrieveLoop is responsible for interacting with DA layer.
func (m *Manager) RetrieveLoop(ctx context.Context) {
	// waitCh is used to signal the retrieve loop, that it should process next blocks
	// retrieveCond can be signalled in completely async manner, and goroutine below
	// works as some kind of "buffer" for those signals
	waitCh := make(chan interface{})
	go func() {
		for {
			m.retrieveMtx.Lock()
			m.retrieveCond.Wait()
			waitCh <- nil
			m.retrieveMtx.Unlock()
			if ctx.Err() != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-waitCh:
			for {
				daHeight := atomic.LoadUint64(&m.daHeight)
				m.logger.Debug("retrieve", "daHeight", daHeight)
				err := m.processNextDABatch()
				if err != nil {
					m.logger.Error("failed to retrieve block from DALC", "daHeight", daHeight, "errors", err.Error())
					break
				}
				atomic.AddUint64(&m.daHeight, 1)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) processNextDABatch() error {
	// TODO(tzdybal): extract configuration option
	maxRetries := 10
	daHeight := atomic.LoadUint64(&m.daHeight)

	var err error
	m.logger.Debug("trying to retrieve batch from DA", "daHeight", daHeight)
	for r := 0; r < maxRetries; r++ {
		batchResp, fetchErr := m.fetchBatch(daHeight)
		if fetchErr != nil {
			err = multierr.Append(err, fetchErr)
			time.Sleep(100 * time.Millisecond)
		} else {
			m.logger.Debug("retrieved potential batches", "n", len(batchResp.Batches), "daHeight", daHeight)
			for _, batch := range batchResp.Batches {
				for _, block := range batch.Blocks {
					m.blockInCh <- newBlockEvent{block, daHeight}
				}
			}
			return nil
		}
	}
	return err
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

func (m *Manager) getRemainingSleep(start time.Time) time.Duration {
	publishingDuration := time.Since(start)
	sleepDuration := m.conf.BlockTime - publishingDuration
	if sleepDuration < 0 {
		sleepDuration = 0
	}
	return sleepDuration
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
	pendingBlock, err := m.store.LoadBlock(newHeight)
	if err == nil {
		m.logger.Info("Using pending block", "height", newHeight)
		block = pendingBlock
	} else {
		m.logger.Info("Creating and publishing block", "height", newHeight)
		block = m.executor.CreateBlock(newHeight, lastCommit, lastHeaderHash, m.lastState)
		m.logger.Debug("block info", "num_tx", len(block.Data.Txs))

		headerBytes, err := block.Header.MarshalBinary()
		if err != nil {
			return err
		}
		sign, err := m.proposerKey.Sign(headerBytes)
		if err != nil {
			return err
		}
		commit := &types.Commit{
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

	err = m.submitBatchToDA(ctx, block)
	if err != nil {
		m.logger.Error("Failed to submit block to DA Layer")
		return err
	}

	// Only update the stored height after successfully submitting to DA layer
	m.store.SetHeight(block.Header.Height)

	// Commit the new state and block which writes to disk on the proxy app
	_, _, err = m.executor.Commit(ctx, newState, block, responses)
	if err != nil {
		return err
	}

	// SaveBlockResponses commits the DB tx
	err = m.store.SaveBlockResponses(block.Header.Height, responses)
	if err != nil {
		return err
	}

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

	m.publishHeader(block)

	return nil
}

func (m *Manager) submitBatchToDA(ctx context.Context, block *types.Block) error {
	m.logger.Info("submitting batch to DA layer", "height", block.Header.Height)

	submitted := false
	backoff := initialBackoff
	for attempt := 1; ctx.Err() == nil && !submitted && attempt <= maxSubmitAttempts; attempt++ {
		batch := &types.Batch{
			StartHeight: block.Header.Height,
			EndHeight:   block.Header.Height,
			Blocks:      []*types.Block{block},
		}
		res := m.dalc.SubmitBatch(batch)
		if res.Code == da.StatusSuccess {
			m.logger.Info("successfully submitted dymint batch to DA layer", "dymintHeight", block.Header.Height, "daHeight", res.DAHeight)
			submitted = true
		} else {
			m.logger.Error("DA layer submission failed", "error", res.Message, "attempt", attempt)
			time.Sleep(backoff)
			backoff = m.exponentialBackoff(backoff)
		}
	}

	if !submitted {
		return fmt.Errorf("failed to submit batch to DA layer after %d attempts", maxSubmitAttempts)
	}

	m.HeaderOutCh <- &block.Header

	return nil
}

func (m *Manager) exponentialBackoff(backoff time.Duration) time.Duration {
	backoff *= 2
	if backoff > m.conf.DABlockTime {
		backoff = m.conf.DABlockTime
	}
	return backoff
}

func (m *Manager) publishHeader(block *types.Block) {
	m.HeaderOutCh <- &block.Header
}

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
