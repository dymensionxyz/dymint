package block

import (
	"context"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/types"
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

// SubmitLoop is the main loop for submitting blocks to the DA and SL layers.
// It is triggered by the shouldSubmitBatchCh channel, which is triggered by the block production loop when accumulated produced size is enogh to submit.
// It is also triggered by a BatchSubmitMaxTime timer to limit the time between submissions.
func (m *Manager) SubmitLoop(ctx context.Context) {
	// ticker to limit the time between submissions
	ticker := time.NewTicker(m.Conf.BatchSubmitMaxTime)
	defer ticker.Stop()

	// get produced size from the block production loop and signal to submit the batch when batch size reached
	submitByAccumulatedSizeCh := make(chan bool, m.Conf.MaxSupportedBatchSkew)
	go m.AccumulatedDataLoop(ctx, submitByAccumulatedSizeCh)

	// defer func to clear the channels to release blocked goroutines on shutdown
	defer func() {
		for {
			select {
			case <-m.producedSizeCh:
			case <-submitByAccumulatedSizeCh:
			default:
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-submitByAccumulatedSizeCh: // block production
		case <-ticker.C: // max time
			// reset the accumulated size when triggered by time,
			// as we gonna submit all our data anyway
			m.AccumulatedBatchSize.Store(0)
		}

		// modular submission methods have own retries mechanism.
		// if error returned, we assume it's unrecoverable.
		err := m.HandleSubmissionTrigger(ctx)
		if err != nil {
			panic(fmt.Errorf("handle submission trigger: %w", err))
		}
		ticker.Reset(m.Conf.BatchSubmitMaxTime)
	}
}

// AccumulatedDataLoop is the main loop for accumulating the produced data size.
// It is triggered by the ProducedSizeCh channel, which is triggered by the block production loop when a new block is produced.
// It accumulates the size of the produced data and triggers the submission of the batch when the accumulated size is greater than the max size.
// It also emits a health status event when the submission channel is full.
func (m *Manager) AccumulatedDataLoop(ctx context.Context, toSubmit chan bool) {
	for {
		select {
		case <-ctx.Done():
			return
		case size := <-m.producedSizeCh:
			total := m.AccumulatedBatchSize.Add(size)

			// Check if accumulated size is greater than the max size
			// TODO: allow some tolerance for block size (e.g support for BlockBatchMaxSize +- 10%)
			if total > m.Conf.BlockBatchMaxSizeBytes {
				select {
				case toSubmit <- true:
					m.logger.Info("new batch accumulated, signal sent to submit the batch")
				default:
					m.logger.Error("new batch accumulated, but channel is full, stopping block production until the signal is consumed")
					// emit unhealthy event for the node
					evt := &events.DataHealthStatus{Error: fmt.Errorf("submission channel is full")}
					uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
					// wait for the signal to be consumed
					select {
					case <-ctx.Done():
						return
					case toSubmit <- true:
					}
					m.logger.Info("resumed block production")
					// emit healthy event for the node
					evt = &events.DataHealthStatus{Error: nil}
					uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
				}
				m.AccumulatedBatchSize.Store(0)
			}
		}
	}
}

// HandleSubmissionTrigger processes the sublayer submission trigger event. It checks if there are new blocks produced since the last submission.
// If there are, it attempts to submit a batch of blocks. It then attempts to produce an empty block to ensure IBC messages
// pass through during the batch submission process due to proofs requires for ibc messages only exist on the next block.
// Finally, it submits the next batch of blocks and updates the sync target to the height of the last block in the submitted batch.
func (m *Manager) HandleSubmissionTrigger(ctx context.Context) error {
	// Load current sync target and height to determine if new blocks are available for submission.
	if m.Store.Height() <= m.SyncTarget.Load() {
		return nil // No new blocks have been produced
	}

	nextBatch, err := m.createNextBatch()
	if err != nil {
		return fmt.Errorf("create next batch: %w", err)
	}

	resultSubmitToDA, err := m.submitNextBatchToDA(nextBatch)
	if err != nil {
		return fmt.Errorf("submit next batch to da: %w", err)
	}

	syncHeight, err := m.submitNextBatchToSL(nextBatch, resultSubmitToDA)
	if err != nil {
		return fmt.Errorf("submit pending batch to sl: %w", err)
	}

	// Update the syncTarget to the height of the last block in the last batch as seen by this node.
	m.UpdateSyncParams(syncHeight)
	return nil
}

func (m *Manager) createNextBatch() (*types.Batch, error) {
	// Create the batch
	startHeight := m.SyncTarget.Load() + 1
	endHeight := m.Store.Height()
	nextBatch, err := m.CreateNextBatchToSubmit(startHeight, endHeight)
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

	// Submit batch to the DA
	m.logger.Info("Submitting next batch", "startHeight", startHeight, "endHeight", actualEndHeight, "size", nextBatch.ToProto().Size())
	resultSubmitToDA := m.DAClient.SubmitBatch(nextBatch)
	if resultSubmitToDA.Code != da.StatusSuccess {
		return nil, fmt.Errorf("submit next batch to DA Layer: %s", resultSubmitToDA.Message)
	}
	return &resultSubmitToDA, nil
}

func (m *Manager) submitNextBatchToSL(batch *types.Batch, daResult *da.ResultSubmitBatch) (uint64, error) {
	startHeight := batch.StartHeight
	actualEndHeight := batch.EndHeight
	err := m.SLClient.SubmitBatch(batch, m.DAClient.GetClientType(), daResult)
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

func (m *Manager) CreateNextBatchToSubmit(startHeight uint64, endHeight uint64) (*types.Batch, error) {
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
