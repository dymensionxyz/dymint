package block

import (
	"context"
	"sync/atomic"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
)

func (m *Manager) submitNextBatch(ctx context.Context) {
	// Get the batch start and end height
	startHeight := atomic.LoadUint64(&m.syncTarget) + 1
	endHeight := uint64(m.lastState.LastBlockHeight)

	// Create the batch
	nextBatch, err := m.createNextDABatch(startHeight, endHeight)
	if err != nil {
		m.logger.Error("Failed to create next batch", "startHeight", startHeight, "endHeight", endHeight, "error", err)
		return
	}

	actualEndHeight := nextBatch.EndHeight

	// Submit batch to the DA
	m.logger.Info("Submitting next batch", "startHeight", startHeight, "endHeight", actualEndHeight, "size", nextBatch.ToProto().Size())
	resultSubmitToDA := m.dalc.SubmitBatch(nextBatch)
	if resultSubmitToDA.Code != da.StatusSuccess {
		panic("Failed to submit next batch to DA Layer")
	}

	// Submit batch to SL
	// TODO(omritoptix): Handle a case where the SL submission fails due to syncTarget out of sync with the latestHeight in the SL.
	// In that case we'll want to update the syncTarget before returning.
	m.settlementClient.SubmitBatch(nextBatch, m.dalc.GetClientType(), &resultSubmitToDA)
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
