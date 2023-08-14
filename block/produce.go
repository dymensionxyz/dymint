package block

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"cosmossdk.io/errors"
	abciconv "github.com/dymensionxyz/dymint/conv/abci"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

// ProduceBlockLoop is calling publishBlock in a loop as long as wer'e synced.
func (m *Manager) ProduceBlockLoop(ctx context.Context) {
	atomic.StoreInt64(&m.lastSubmissionTime, time.Now().Unix())

	// We want to wait until we are synced. After that, since there is no leader
	// election yet, and leader are elected manually, we will not be out of sync until
	// we are manually being replaced.
	err := m.waitForSync(ctx)
	if err != nil {
		panic(errors.Wrap(err, "failed to wait for sync"))
	}

	ticker := time.NewTicker(m.conf.BlockTime)
	defer ticker.Stop()

	var tickerEmptyBlocksMaxTime *time.Ticker
	var tickerEmptyBlocksMaxTimeCh <-chan time.Time
	// Setup ticker for empty blocks if enabled
	if m.conf.EmptyBlocksMaxTime > 0 {
		tickerEmptyBlocksMaxTime = time.NewTicker(m.conf.EmptyBlocksMaxTime)
		tickerEmptyBlocksMaxTimeCh = tickerEmptyBlocksMaxTime.C
		defer tickerEmptyBlocksMaxTime.Stop()
	}

	//Allow the initial block to be empty
	produceEmptyBlock := true
	for {
		select {
		//Context canceled
		case <-ctx.Done():
			return
		//Empty blocks timeout
		case <-tickerEmptyBlocksMaxTimeCh:
			m.logger.Debug(fmt.Sprintf("No transactions for %.2f seconds, producing empty block", m.conf.EmptyBlocksMaxTime.Seconds()))
			produceEmptyBlock = true
		//Produce block
		case <-ticker.C:
			err := m.produceBlock(ctx, produceEmptyBlock)
			if err == types.ErrSkippedEmptyBlock {
				m.logger.Debug("Skipped empty block")
				continue
			}
			if err != nil {
				m.logger.Error("error while producing block", "error", err)
				continue
			}
			//If empty blocks enabled, after block produced, reset the timeout timer
			if tickerEmptyBlocksMaxTime != nil {
				produceEmptyBlock = false
				tickerEmptyBlocksMaxTime.Reset(m.conf.EmptyBlocksMaxTime)
			}

		//Node's health check channel
		case shouldProduceBlocks := <-m.shouldProduceBlocksCh:
			for !shouldProduceBlocks {
				m.logger.Info("Stopped block production")
				shouldProduceBlocks = <-m.shouldProduceBlocksCh
			}
			m.logger.Info("Resumed Block production")
		}
	}
}

func (m *Manager) produceBlock(ctx context.Context, allowEmpty bool) error {
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
		block = m.executor.CreateBlock(newHeight, lastCommit, lastHeaderHash, m.lastState)
		if !allowEmpty && len(block.Data.Txs) == 0 {
			return types.ErrSkippedEmptyBlock
		}

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

	m.logger.Info("block created", "height", newHeight, "num_tx", len(block.Data.Txs))
	rollappHeightGauge.Set(float64(newHeight))

	//TODO: move to separate function
	lastSubmissionTime := atomic.LoadInt64(&m.lastSubmissionTime)
	requiredByTime := time.Since(time.Unix(0, lastSubmissionTime)) > m.conf.BatchSubmitMaxTime

	// SyncTarget is the height of the last block in the last batch as seen by this node.
	syncTarget := atomic.LoadUint64(&m.syncTarget)
	requiredByNumOfBlocks := (block.Header.Height - syncTarget) > m.conf.BlockBatchSize

	// Submit batch if we've reached the batch size and there isn't another batch currently in submission process.
	if m.batchInProcess.Load() == false && (requiredByTime || requiredByNumOfBlocks) {
		m.batchInProcess.Store(true)
		go m.submitNextBatch(ctx)
	}

	return nil
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
