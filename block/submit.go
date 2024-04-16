package block

import (
	"context"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
)

func (m *Manager) SubmitLoop(ctx context.Context) {
	ticker := time.NewTicker(m.conf.BatchSubmitMaxTime)
	defer ticker.Stop()

	// TODO: add submission trigger by batch size (should be signaled from the the block production)
	for {
		select {
		// Context canceled
		case <-ctx.Done():
			return
		// trigger by time
		case <-ticker.C:
			m.handleSubmissionTrigger(ctx)
		}
	}
}

func (m *Manager) handleSubmissionTrigger(ctx context.Context) {
	// SyncTarget is the height of the last block in the last batch as seen by this node.
	syncTarget := m.syncTarget.Load()
	height := m.store.Height()
	// no new blocks produced yet
	if height <= syncTarget {
		return
	}

	// Submit batch if we've reached the batch size and there isn't another batch currently in submission process.

	if !m.submitBatchMutex.TryLock() {
		m.logger.Debug("Batch submission already in process, skipping submission")
		return
	}

	defer m.submitBatchMutex.Unlock()

	// We try and produce an empty block to make sure releavnt ibc messages will pass through during the batch submission: https://github.com/dymensionxyz/research/issues/173.
	err := m.produceBlock(ctx, true)
	if err != nil {
		m.logger.Error("while producing empty block", "error", err)
	}

	syncHeight, err := m.submitNextBatch()
	if err != nil {
		m.logger.Error("while submitting next batch", "error", err)
		return
	}

	// Update the syncTarget to the height of the last block in the last batch as seen by this node.
	m.updateSyncParams(syncHeight)
}

func (m *Manager) submitNextBatch() (uint64, error) {
	// Get the batch start and end height
	startHeight := m.syncTarget.Load() + 1
	endHeight := uint64(m.store.Height())

	// Create the batch
	nextBatch, err := m.createNextDABatch(startHeight, endHeight)
	if err != nil {
		m.logger.Error("create next batch", "startHeight", startHeight, "endHeight", endHeight, "error", err)
		return 0, err
	}

	if err := m.validateBatch(nextBatch); err != nil {
		return 0, err
	}
	actualEndHeight := nextBatch.EndHeight

	isLastBlockEmpty, err := m.isBlockEmpty(actualEndHeight)
	if err != nil {
		m.logger.Error("validate last block in batch is empty", "startHeight", startHeight, "endHeight", actualEndHeight, "error", err)
		return 0, err
	}
	// Verify the last block in the batch is an empty block and that no ibc messages has accidentially passed through.
	// This block may not be empty if another block has passed it in line. If that's the case our empty block request will
	// be sent to the next batch.
	if !isLastBlockEmpty {
		m.logger.Info("Last block in batch is not an empty block. Requesting for an empty block creation", "endHeight", actualEndHeight)
		m.produceEmptyBlockCh <- true
	}

	// Submit batch to the DA
	m.logger.Info("Submitting next batch", "startHeight", startHeight, "endHeight", actualEndHeight, "size", nextBatch.ToProto().Size())
	resultSubmitToDA := m.dalc.SubmitBatch(nextBatch)
	if resultSubmitToDA.Code != da.StatusSuccess {
		err = fmt.Errorf("submit next batch to DA Layer: %s", resultSubmitToDA.Message)
		return 0, err
	}

	// Submit batch to SL
	err = m.settlementClient.SubmitBatch(nextBatch, m.dalc.GetClientType(), &resultSubmitToDA)
	if err != nil {
		m.logger.Error("submit batch to SL", "startHeight", startHeight, "endHeight", actualEndHeight, "error", err)
		return 0, err
	}

	return actualEndHeight, nil
}

func (m *Manager) validateBatch(batch *types.Batch) error {
	syncTarget := m.syncTarget.Load()
	if batch.StartHeight != syncTarget+1 {
		return fmt.Errorf("batch start height != syncTarget + 1. StartHeight %d, m.syncTarget %d", batch.StartHeight, syncTarget)
	}
	if batch.EndHeight < batch.StartHeight {
		return fmt.Errorf("batch end height must be greater than start height. EndHeight %d, StartHeight %d", batch.EndHeight, batch.StartHeight)
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
			m.logger.Error("load block", "height", height)
			return nil, err
		}
		commit, err := m.store.LoadCommit(height)
		if err != nil {
			m.logger.Error("load commit", "height", height)
			return nil, err
		}

		batch.Blocks = append(batch.Blocks, block)
		batch.Commits = append(batch.Commits, commit)

		// Check if the batch size is too big
		totalSize := batch.ToProto().Size()
		if totalSize > int(m.conf.BlockBatchMaxSizeBytes) {
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

func (m *Manager) isBlockEmpty(endHeight uint64) (isEmpty bool, err error) {
	m.logger.Debug("Verifying last block in batch is an empty block", "endHeight", endHeight, "height")
	lastBlock, err := m.store.LoadBlock(endHeight)
	if err != nil {
		m.logger.Error("load block", "height", endHeight, "error", err)
		return false, err
	}

	return len(lastBlock.Data.Txs) == 0, nil
}
