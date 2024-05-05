package block

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/gerr"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
)

// SubmitLoop submits a batch of blocks to the DA and SL layers on a time interval.
func (m *Manager) SubmitLoop(ctx context.Context) {
	ticker := time.NewTicker(m.Conf.BatchSubmitMaxTime)
	defer ticker.Stop()

	// TODO: add submission trigger by batch size (should be signaled from the the block production)
	for {
		select {
		// Context canceled
		case <-ctx.Done():
			return
		// trigger by time
		case <-ticker.C:
			err := m.HandleSubmissionTrigger(ctx)
			if errors.Is(err, gerr.ErrAborted) {
				continue
			}
			if errors.Is(err, gerr.ErrUnauthenticated) {
				panic(fmt.Errorf("handle submission trigger: %w", err))
			}
			if err != nil {
				m.logger.Error("handle submission trigger", "error", err)
			}
		}
	}
}

// HandleSubmissionTrigger processes the sublayer submission trigger event. It checks if there are new blocks produced since the last submission.
// If there are, it attempts to submit a batch of blocks. It then attempts to produce an empty block to ensure IBC messages
// pass through during the batch submission process due to proofs requires for ibc messages only exist on the next block.
// Finally, it submits the next batch of blocks and updates the sync target to the height of the last block in the submitted batch.
func (m *Manager) HandleSubmissionTrigger(ctx context.Context) error {
	if !m.submitBatchMutex.TryLock() {
		return fmt.Errorf("batch submission already in process, skipping submission: %w", gerr.ErrAborted)
	}
	defer m.submitBatchMutex.Unlock()

	// Load current sync target and height to determine if new blocks are available for submission.
	if m.Store.Height() <= m.SyncTarget.Load() {
		return nil // No new blocks have been produced
	}

	// We try and produce an empty block to make sure relevant ibc messages will pass through during the batch submission: https://github.com/dymensionxyz/research/issues/173.
	err := m.ProduceAndGossipBlock(ctx, true)
	if err != nil {
		m.logger.Error("Produce and gossip empty block.", "error", err)
	}

	if m.pendingBatch == nil {
		nextBatch, err := m.createNextBatch()
		if err != nil {
			return fmt.Errorf("create next batch: %w", err)
		}

		resultSubmitToDA, err := m.submitNextBatchToDA(nextBatch)
		if err != nil {
			return fmt.Errorf("submit next batch to da: %w", err)
		}

		m.pendingBatch = &PendingBatch{
			daResult: resultSubmitToDA,
			batch:    nextBatch,
		}
	} else {
		m.logger.Info("Pending batch already exists.", "startHeight", m.pendingBatch.batch.StartHeight, "endHeight", m.pendingBatch.batch.EndHeight)
	}

	syncHeight, err := m.submitPendingBatchToSL(*m.pendingBatch)
	if err != nil {
		return fmt.Errorf("submit pending batch to sl: %w", err)
	}

	m.pendingBatch = nil

	// Update the syncTarget to the height of the last block in the last batch as seen by this node.
	m.UpdateSyncParams(syncHeight)
	return nil
}

func (m *Manager) createNextBatch() (*types.Batch, error) {
	// Create the batch
	startHeight := m.SyncTarget.Load() + 1
	endHeight := m.Store.Height()
	nextBatch, err := m.CreateNextDABatch(startHeight, endHeight)
	if err != nil {
		m.logger.Error("create next batch", "startHeight", startHeight, "endHeight", endHeight, "error", err)
		return nil, err
	}

	if err := m.ValidateBatch(nextBatch); err != nil {
		return nil, err
	}

	return nextBatch, nil
}

func (m *Manager) submitNextBatchToDA(nextBatch *types.Batch) (*da.ResultSubmitBatch, error) {
	startHeight := nextBatch.StartHeight
	actualEndHeight := nextBatch.EndHeight

	isLastBlockEmpty, err := m.isBlockEmpty(actualEndHeight)
	if err != nil {
		m.logger.Error("validate last block in batch is empty", "startHeight", startHeight, "endHeight", actualEndHeight, "error", err)
		return nil, err
	}
	// Verify the last block in the batch is an empty block and that no ibc messages has accidentally passed through.
	// This block may not be empty if another block has passed it in line. If that's the case our empty block request will
	// be sent to the next batch.
	if !isLastBlockEmpty {
		m.logger.Info("Last block in batch is not an empty block. Requesting for an empty block creation", "endHeight", actualEndHeight)
		m.produceEmptyBlockCh <- true
	}

	// Submit batch to the DA
	m.logger.Info("Submitting next batch", "startHeight", startHeight, "endHeight", actualEndHeight, "size", nextBatch.ToProto().Size())
	resultSubmitToDA := m.DAClient.SubmitBatch(nextBatch)
	if resultSubmitToDA.Code != da.StatusSuccess {
		err = fmt.Errorf("submit next batch to DA Layer: %s", resultSubmitToDA.Message)
		return nil, err
	}
	return &resultSubmitToDA, nil
}

func (m *Manager) submitPendingBatchToSL(p PendingBatch) (uint64, error) {
	startHeight := p.batch.StartHeight
	actualEndHeight := p.batch.EndHeight
	err := m.SLClient.SubmitBatch(p.batch, m.DAClient.GetClientType(), p.daResult)
	if err != nil {
		return 0, fmt.Errorf("sl client submit batch: startheight: %d: actual end height: %d: %w", startHeight, actualEndHeight, err)
	}

	return actualEndHeight, nil
}

func (m *Manager) ValidateBatch(batch *types.Batch) error {
	syncTarget := m.SyncTarget.Load()
	if batch.StartHeight != syncTarget+1 {
		return fmt.Errorf("batch start height != syncTarget + 1. StartHeight %d, m.SyncTarget %d", batch.StartHeight, syncTarget)
	}
	if batch.EndHeight < batch.StartHeight {
		return fmt.Errorf("batch end height must be greater than start height. EndHeight %d, StartHeight %d", batch.EndHeight, batch.StartHeight)
	}
	return nil
}

func (m *Manager) CreateNextDABatch(startHeight uint64, endHeight uint64) (*types.Batch, error) {
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
		block, err := m.Store.LoadBlock(height)
		if err != nil {
			m.logger.Error("load block", "height", height)
			return nil, err
		}
		commit, err := m.Store.LoadCommit(height)
		if err != nil {
			m.logger.Error("load commit", "height", height)
			return nil, err
		}

		batch.Blocks = append(batch.Blocks, block)
		batch.Commits = append(batch.Commits, commit)

		// Check if the batch size is too big
		totalSize := batch.ToProto().Size()
		if totalSize > int(m.Conf.BlockBatchMaxSizeBytes) {
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
	lastBlock, err := m.Store.LoadBlock(endHeight)
	if err != nil {
		m.logger.Error("load block", "height", endHeight, "error", err)
		return false, err
	}

	return len(lastBlock.Data.Txs) == 0, nil
}
