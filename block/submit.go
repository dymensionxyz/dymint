package block

import (
	"context"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/gerr"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/types"
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

// SubmitLoop is the main loop for submitting blocks to the DA and SL layers.
// It submits a batch when either
// 1) It accumulates enough block data, so it's necessary to submit a batch to avoid exceeding the max size
// 2) Enough time passed since the last submitted batch, so it's necessary to submit a batch to avoid exceeding the max time
func (m *Manager) SubmitLoop(ctx context.Context) {
	maxTime := time.NewTicker(m.Conf.BatchSubmitMaxTime)
	defer maxTime.Stop()

	// get produced size from the block production loop and signal to submit the batch when batch size reached
	maxSizeC := make(chan struct{}, m.Conf.MaxSupportedBatchSkew)
	go m.AccumulatedDataLoop(ctx, maxSizeC)

	// defer func to clear the channels to release blocked goroutines on shutdown
	defer func() {
		for {
			select {
			case <-m.producedSizeCh:
			case <-maxSizeC:
			default:
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-maxSizeC:
		case <-maxTime.C:
		}

		/*
				Note: since we dont explicitly coordinate changes to the accumulated size with actual batch creation
				we don't have a guarantee that the accumulated size is the same as the actual batch size that will be made.
				See https://github.com/dymensionxyz/dymint/issues/828
				Until that is fixed, it's technically possibly to undercount, by having a some blocks be produced in between
			    setting the counter to 0, and actually producing the batch.
		*/
		m.AccumulatedBatchSize.Store(0)

		// modular submission methods have own retries mechanism.
		// if error returned, we assume it's unrecoverable.
		err := m.HandleSubmissionTrigger()
		if err != nil {
			panic(fmt.Errorf("handle submission trigger: %w", err))
		}
		maxTime.Reset(m.Conf.BatchSubmitMaxTime)
	}
}

// AccumulatedDataLoop is the main loop for accumulating the produced data size.
// It is triggered by the ProducedSizeCh channel, which is populated by the block production loop when a new block is produced.
// It accumulates the size of the produced data and triggers the submission of the batch when the accumulated size is greater than the max size.
// It also emits a health status event when the submission channel is full.
func (m *Manager) AccumulatedDataLoop(ctx context.Context, toSubmit chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case size := <-m.producedSizeCh:
			total := m.AccumulatedBatchSize.Add(size)
			if total < m.Conf.BlockBatchMaxSizeBytes { // TODO: allow some tolerance for block size (e.g support for BlockBatchMaxSize +- 10%)
				continue
			}
		}

		select {
		case <-ctx.Done():
			return
		case toSubmit <- struct{}{}:
			m.logger.Info("New batch accumulated, sent signal to submit the batch.")
		default:
			m.logger.Error("New batch accumulated, but channel is full, stopping block production until the signal is consumed.")

			evt := &events.DataHealthStatus{Error: fmt.Errorf("submission channel is full: %w", gerr.ErrResourceExhausted)}
			uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)

			/*
				Now we stop consuming the produced size channel, so the block production loop will stop producing new blocks.
			*/
			select {
			case <-ctx.Done():
				return
			case toSubmit <- struct{}{}:
			}
			m.logger.Info("Resumed block production.")

			evt = &events.DataHealthStatus{Error: nil}
			uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
		}
	}
}

// HandleSubmissionTrigger processes the sublayer submission trigger event. It checks if there are new blocks produced since the last submission.
// If there are, it attempts to submit a batch of blocks. It then attempts to produce an empty block to ensure IBC messages
// pass through during the batch submission process due to proofs requires for ibc messages only exist on the next block.
// Finally, it submits the next batch of blocks and updates the sync target to the height of the last block in the submitted batch.
func (m *Manager) HandleSubmissionTrigger() error {
	// Load current sync target and height to determine if new blocks are available for submission.

	startHeight := m.SyncTarget.Load() + 1
	endHeightInclusive := m.Store.Height()

	if endHeightInclusive < startHeight {
		return nil // No new blocks have been produced
	}

	nextBatch, err := m.CreateNextBatchToSubmit(startHeight, endHeightInclusive)
	if err != nil {
		return fmt.Errorf("create next batch to submit: %w", err)
	}

	if err := m.ValidateBatch(nextBatch); err != nil {
		return fmt.Errorf("validate batch: %w", err)
	}

	resultSubmitToDA, err := m.submitNextBatchToDA(nextBatch)
	if err != nil {
		return fmt.Errorf("submit next batch to da: %w", err)
	}

	err = m.SLClient.SubmitBatch(nextBatch, m.DAClient.GetClientType(), resultSubmitToDA)
	if err != nil {
		return fmt.Errorf("sl client submit batch: start height: %d: inclusive end height: %d: %w", startHeight, endHeightInclusive, err)
	}

	m.UpdateSyncParams(endHeightInclusive)
	return nil
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

func (m *Manager) CreateNextBatchToSubmit(startHeight uint64, endHeightInclusive uint64) (*types.Batch, error) {
	var height uint64
	// Create the batch
	batchSize := endHeightInclusive - startHeight + 1
	batch := &types.Batch{
		StartHeight: startHeight,
		EndHeight:   endHeightInclusive,
		Blocks:      make([]*types.Block, 0, batchSize),
		Commits:     make([]*types.Commit, 0, batchSize),
	}

	// Populate the batch
	for height = startHeight; height <= endHeightInclusive; height++ {
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
