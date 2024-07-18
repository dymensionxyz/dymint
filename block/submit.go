package block

import (
	"context"
	"fmt"
	"time"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/types"
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

// SubmitLoop is the main loop for submitting blocks to the DA and SL layers.
// It submits a batch when either
// 1) It accumulates enough block data, so it's necessary to submit a batch to avoid exceeding the max size
// 2) Enough time passed since the last submitted batch, so it's necessary to submit a batch to avoid exceeding the max time
func (m *Manager) SubmitLoop(ctx context.Context) (err error) {
	maxTime := time.NewTicker(m.Conf.BatchSubmitMaxTime)
	defer maxTime.Stop()

	// get produced size from the block production loop and signal to submit the batch when batch size reached
	maxSizeC := make(chan struct{}, m.Conf.MaxSupportedBatchSkew)
	go m.AccumulatedDataLoop(ctx, maxSizeC)

	// defer func to clear the channels to release blocked goroutines on shutdown
	defer func() {
		m.logger.Info("Stopped submit loop.")

		for {
			select {
			case <-m.producedSizeC:
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

		// modular submission methods have own retries mechanism.
		// if error returned, we assume it's unrecoverable.
		err = m.HandleSubmissionTrigger()
		if err != nil {
			m.logger.Error("Error submitting batch", "error", err)
			uevent.MustPublish(ctx, m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
			return
		}
		maxTime.Reset(m.Conf.BatchSubmitMaxTime)
	}
}

// AccumulatedDataLoop is the main loop for accumulating the produced data size.
// It is triggered by the ProducedSizeCh channel, which is populated by the block production loop when a new block is produced.
// It accumulates the size of the produced data and triggers the submission of the batch when the accumulated size is greater than the max size.
// It also emits a health status event when the submission channel is full.
func (m *Manager) AccumulatedDataLoop(ctx context.Context, toSubmit chan struct{}) {
	total := uint64(0)
	// Get size unsubmitted blocks and commits
	currH := m.State.Height()
	for h := m.LastSubmittedHeight.Load() + 1; h <= currH; h++ {
		block := m.MustLoadBlock(h)
		commit := m.MustLoadCommit(h)
		total += uint64(block.ToProto().Size()) + uint64(commit.ToProto().Size())
	}

	for {
		select {
		case <-ctx.Done():
			return
		case size := <-m.producedSizeC:
			total += size
			if total < m.Conf.BlockBatchMaxSizeBytes { // TODO: allow some tolerance for block size (e.g support for BlockBatchMaxSize +- 10%)
				// batch size limit not yet reached so we don't want to submit a batch yet
				// note: the actual batch size is slightly more than the sum of block sizes, but this is a rough estimate
				continue
			}
		}

		total -= m.Conf.BlockBatchMaxSizeBytes

		select {
		case <-ctx.Done():
			return
		case toSubmit <- struct{}{}:
			m.logger.Info("Enough bytes to build a batch have been accumulated. Sent signal to submit the batch.")
		default:
			m.logger.Error("Enough bytes to build a batch have been accumulated. Sent signal to submit the batch. " +
				"Pausing block production until the signal is consumed.")

			evt := &events.DataHealthStatus{Error: fmt.Errorf("submission channel is full: %w", gerrc.ErrResourceExhausted)}
			uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)

			/*
				Now we block until earlier batches have been submitted. This has the effect of not consuming the producedSizeC,
				which will stop new block production.
			*/
			select {
			case <-ctx.Done():
				return
			case toSubmit <- struct{}{}:
			}

			evt = &events.DataHealthStatus{Error: nil}
			uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)

			m.logger.Info("Resumed block production.")
		}
	}
}

// HandleSubmissionTrigger processes the sublayer submission trigger event. It checks if there are new blocks produced since the last submission.
// If there are, it attempts to submit a batch of blocks. It then attempts to produce an empty block to ensure IBC messages
// pass through during the batch submission process due to proofs requires for ibc messages only exist on the next block.
// Finally, it submits the next batch of blocks and updates the sync target to the height of the last block in the submitted batch.
func (m *Manager) HandleSubmissionTrigger() error {
	// Load current sync target and height to determine if new blocks are available for submission.

	startHeight := m.NextHeightToSubmit()
	endHeightInclusive := m.State.Height()

	if endHeightInclusive < startHeight {
		return nil // No new blocks have been produced
	}

	nextBatch, err := m.CreateNextBatchToSubmit(startHeight, endHeightInclusive)
	if err != nil {
		return fmt.Errorf("create next batch to submit: %w", err)
	}

	resultSubmitToDA := m.DAClient.SubmitBatch(nextBatch)
	m.logger.Info("Submitted batch to DA", "start height", nextBatch.StartHeight, "end height", nextBatch.EndHeight)
	if resultSubmitToDA.Code != da.StatusSuccess {
		return fmt.Errorf("submit next batch to da: %s", resultSubmitToDA.Message)
	}

	actualEndHeight := nextBatch.EndHeight

	err = m.SLClient.SubmitBatch(nextBatch, m.DAClient.GetClientType(), &resultSubmitToDA)
	if err != nil {
		return fmt.Errorf("sl client submit batch: start height: %d: inclusive end height: %d: %w", startHeight, actualEndHeight, err)
	}
	m.logger.Info("Submitted batch to SL.", "start height", resultSubmitToDA, "end height", nextBatch.EndHeight)

	types.RollappHubHeightGauge.Set(float64(actualEndHeight))
	m.LastSubmittedHeight.Store(actualEndHeight)
	return nil
}

func (m *Manager) CreateNextBatchToSubmit(startHeight uint64, endHeightInclusive uint64) (*types.Batch, error) {
	batchSize := endHeightInclusive - startHeight + 1
	batch := &types.Batch{
		StartHeight: startHeight,
		Blocks:      make([]*types.Block, 0, batchSize),
		Commits:     make([]*types.Commit, 0, batchSize),
	}

	// Populate the batch
	for height := startHeight; height <= endHeightInclusive; height++ {
		block, err := m.Store.LoadBlock(height)
		if err != nil {
			return nil, fmt.Errorf("load block: height: %d: %w", height, err)
		}
		commit, err := m.Store.LoadCommit(height)
		if err != nil {
			return nil, fmt.Errorf("load commit: height: %d: %w", height, err)
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

			if height == startHeight {
				return nil, fmt.Errorf("block size exceeds max batch size: height %d: size: %d", height, totalSize)
			}
			break
		}

		batch.EndHeight = height
	}

	return batch, nil
}
