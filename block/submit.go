package block

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"golang.org/x/sync/errgroup"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
	uchannel "github.com/dymensionxyz/dymint/utils/channel"
)

// SubmitLoop is the main loop for submitting blocks to the DA and SL layers.
// It submits a batch when either
// 1) It accumulates enough block data, so it's necessary to submit a batch to avoid exceeding the max size
// 2) Enough time passed since the last submitted batch, so it's necessary to submit a batch to avoid exceeding the max time
// It will back pressure (pause) block production if it falls too far behind.
func (m *Manager) SubmitLoop(ctx context.Context,
	bytesProduced chan int,
) (err error) {
	return SubmitLoopInner(ctx, bytesProduced, m.Conf.MaxBatchSkew, m.Conf.BatchSubmitMaxTime, m.Conf.BatchMaxSizeBytes, m.CreateAndSubmitBatchGetSizeBlocksCommits)
}

// SubmitLoopInner is a unit testable impl of SubmitLoop
func SubmitLoopInner(
	ctx context.Context,
	bytesProduced chan int,
	maxBatchSkew uint64,
	maxBatchTime time.Duration,
	maxBatchBytes uint64,
	createAndSubmitBatch func(maxSizeBytes uint64) (sizeBlocksCommits uint64, err error),
) error {
	eg, ctx := errgroup.WithContext(ctx)

	pendingBytes := atomic.Uint64{}
	trigger := uchannel.NewNudger()   // used to avoid busy waiting (using cpu) on trigger thread
	submitter := uchannel.NewNudger() // used to avoid busy waiting (using cpu) on submitter thread

	eg.Go(func() error {
		// 'trigger': we need one thread to continuously consume the bytes produced channel, and to monitor timer
		ticker := time.NewTicker(maxBatchTime)
		defer ticker.Stop()
		for {
			if maxBatchSkew*maxBatchBytes < pendingBytes.Load() {
				// too much stuff is pending submission
				// we block here until we get a progress nudge from the submitter thread
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-trigger.C:
				case <-ticker.C:
					// It's theoretically possible for the thread scheduler to pause this thread after entering this if statement
					// for enough time for the submitter thread to submit all the pending bytes and do the nudge, and then for the
					// thread scheduler to wake up this thread after the nudge has been missed, which would be a deadlock.
					// Although this is only a theoretical possibility which should never happen in practice, it may be possible, e.g.
					// in adverse CPU conditions or tests using compressed timeframes. To be sound, we also nudge with the ticker, which
					// has no downside.
				}
			} else {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case n := <-bytesProduced:
					pendingBytes.Add(uint64(n))
				case <-ticker.C:
				}
			}

			types.RollappPendingSubmissionsSkewNumBytes.Set(float64(pendingBytes.Load()))
			types.RollappPendingSubmissionsSkewNumBatches.Set(float64(pendingBytes.Load() / maxBatchBytes))
			submitter.Nudge()
		}
	})

	eg.Go(func() error {
		// 'submitter': this thread actually creates and submits batches
		timeLastSubmission := time.Now()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-submitter.C:
			}
			pending := pendingBytes.Load()
			// while there are accumulated blocks, create and submit batches!!
			for {
				done := ctx.Err() != nil
				nothingToSubmit := pending == 0
				lastSubmissionIsRecent := time.Since(timeLastSubmission) < maxBatchTime
				maxDataNotExceeded := pending <= maxBatchBytes
				if done || nothingToSubmit || (lastSubmissionIsRecent && maxDataNotExceeded) {
					break
				}
				nConsumed, err := createAndSubmitBatch(min(pending, maxBatchBytes))
				if err != nil {
					err = fmt.Errorf("create and submit batch: %w", err)
					if errors.Is(err, gerrc.ErrInternal) {
						panic(err)
					}
					return err
				}
				timeLastSubmission = time.Now()
				pending = pendingBytes.Add(^(nConsumed - 1)) // subtract
			}
			trigger.Nudge()
		}
	})

	return eg.Wait()
}

// CreateAndSubmitBatchGetSizeBlocksCommits creates and submits a batch to the DA and SL.
// Returns size of block and commit bytes
// max size bytes is the maximum size of the serialized batch type
func (m *Manager) CreateAndSubmitBatchGetSizeBlocksCommits(maxSize uint64) (uint64, error) {
	b, err := m.CreateAndSubmitBatch(maxSize)
	if b == nil {
		return 0, err
	}
	return uint64(b.SizeBlockAndCommitBytes()), err
}

// CreateAndSubmitBatch creates and submits a batch to the DA and SL.
// max size bytes is the maximum size of the serialized batch type
func (m *Manager) CreateAndSubmitBatch(maxSizeBytes uint64) (*types.Batch, error) {
	startHeight := m.NextHeightToSubmit()
	endHeightInclusive := m.State.Height()

	if endHeightInclusive < startHeight {
		// TODO: https://github.com/dymensionxyz/dymint/issues/999
		return nil, fmt.Errorf("next height to submit is greater than last block height, create and submit batch should not have been called: %w", gerrc.ErrInternal)
	}

	b, err := m.CreateBatch(maxSizeBytes, startHeight, endHeightInclusive)
	if err != nil {
		return nil, fmt.Errorf("create batch: %w", err)
	}

	m.logger.Info("Created batch.", "start height", startHeight, "end height", endHeightInclusive)

	if err := m.SubmitBatch(b); err != nil {
		return nil, fmt.Errorf("submit batch: %w", err)
	}
	return b, nil
}

// CreateBatch looks through the store for any unsubmitted blocks and commits and bundles them into a batch
// max size bytes is the maximum size of the serialized batch type
func (m *Manager) CreateBatch(maxBatchSize uint64, startHeight uint64, endHeightInclusive uint64) (*types.Batch, error) {
	batchSize := endHeightInclusive - startHeight + 1
	batch := &types.Batch{
		Blocks:  make([]*types.Block, 0, batchSize),
		Commits: make([]*types.Commit, 0, batchSize),
	}

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

		totalSize := batch.SizeBytes()
		if int(maxBatchSize) < totalSize {

			// Remove the last block and commit from the batch
			batch.Blocks = batch.Blocks[:len(batch.Blocks)-1]
			batch.Commits = batch.Commits[:len(batch.Commits)-1]

			if height == startHeight {
				return nil, fmt.Errorf("block size exceeds max batch size: height %d: size: %d: %w", height, totalSize, gerrc.ErrOutOfRange)
			}
			break
		}
	}

	return batch, nil
}

func (m *Manager) SubmitBatch(batch *types.Batch) error {
	resultSubmitToDA := m.DAClient.SubmitBatch(batch)
	if resultSubmitToDA.Code != da.StatusSuccess {
		return fmt.Errorf("da client submit batch: %s: %w", resultSubmitToDA.Message, resultSubmitToDA.Error)
	}
	m.logger.Info("Submitted batch to DA.", "start height", batch.StartHeight(), "end height", batch.EndHeight())

	err := m.SLClient.SubmitBatch(batch, m.DAClient.GetClientType(), &resultSubmitToDA)
	if err != nil {
		return fmt.Errorf("sl client submit batch: start height: %d: end height: %d: %w", batch.StartHeight(), batch.EndHeight(), err)
	}
	m.logger.Info("Submitted batch to SL.", "start height", batch.StartHeight(), "end height", batch.EndHeight())

	types.RollappHubHeightGauge.Set(float64(batch.EndHeight()))
	m.LastSubmittedHeight.Store(batch.EndHeight())
	return nil
}

// GetUnsubmittedBytes returns the total number of unsubmitted bytes produced an element on a channel
// Intended only to be used at startup, before block production and submission loops start
func (m *Manager) GetUnsubmittedBytes() int {
	total := 0
	/*
		On node start we want to include the count of any blocks which were produced and not submitted in a previous instance
	*/
	currH := m.State.Height()
	for h := m.NextHeightToSubmit(); h <= currH; h++ {
		block, err := m.Store.LoadBlock(h)
		if err != nil {
			if !errors.Is(err, gerrc.ErrNotFound) {
				m.logger.Error("Get unsubmitted bytes load block.", "err", err)
			}
			break
		}
		commit, err := m.Store.LoadBlock(h)
		if err != nil {
			if !errors.Is(err, gerrc.ErrNotFound) {
				m.logger.Error("Get unsubmitted bytes load commit.", "err", err)
			}
			break
		}
		total += block.SizeBytes() + commit.SizeBytes()
	}
	return total
}
