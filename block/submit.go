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
	uatomic "github.com/dymensionxyz/dymint/utils/atomic"
	uchannel "github.com/dymensionxyz/dymint/utils/channel"
)

// SubmitLoop is the main loop for submitting blocks to the DA and SL layers.
// It submits a batch when either
// 1) It accumulates enough block data, so it's necessary to submit a batch to avoid exceeding the max size
// 2) Enough time passed since the last submitted batch, so it's necessary to submit a batch to avoid exceeding the max time
// It will back pressure (pause) block production if it falls too far behind.
func (m *Manager) SubmitLoop(ctx context.Context,
	blockProduced chan struct{},
) (err error) {
	return SubmitLoopInner(
		ctx,
		m.logger,
		blockProduced,
		m.Conf.MaxBatchSkew,
		m.Conf.BatchSubmitMaxTime,
		m.Conf.BatchMaxSizeBytes,
		m.CreateAndSubmitBatchGetSizeBlocksCommits,
	)
}

// SubmitLoopInner is a unit testable impl of SubmitLoop
func SubmitLoopInner(
	ctx context.Context,
	logger types.Logger,
	blockProduced chan struct{}, // a channel indicating a block and commit have been produced
	maxBatchSkew uint64, // max number of batches that submitter is allowed to have pending
	maxBatchTime time.Duration, // max time to allow between batches
	maxBatchBytes uint64, // max size of serialised batch in bytes
	createAndSubmitBatch func(maxSizeBytes uint64) (sizeBlocksCommits uint64, err error),
) error {
	eg, ctx := errgroup.WithContext(ctx)

	pendingBytes := atomic.Uint64{}
	trigger := uchannel.NewNudger()   // used to avoid busy waiting (using cpu) on trigger thread
	submitter := uchannel.NewNudger() // used to avoid busy waiting (using cpu) on submitter thread

	eg.Go(func() error {
		// 'trigger': the job of this thread is to (un)block the block producer by (not)consuming from the channel, and to nudge
		// the submission thread whenever something is produced.

		for {
			chanToWait := blockProduced
			if maxBatchSkew*maxBatchBytes < pendingBytes.Load() {
				// too much stuff has accumulated, wait for submission thread to wake us up after it has made progress
				chanToWait = trigger.C
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-chanToWait:
			}
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
				// TODO: do something with these lines
				types.RollappPendingSubmissionsSkewNumBytes.Set(float64(pendingBytes.Load()))
				types.RollappPendingSubmissionsSkewNumBatches.Set(float64(pendingBytes.Load() / maxBatchBytes))
				done := ctx.Err() != nil
				somethingToSubmit := 0 < pending
				timeExceeded := maxBatchTime <= time.Since(timeLastSubmission)
				maxDataExceeded := maxBatchBytes < pending
				if !done && somethingToSubmit && (timeExceeded || maxDataExceeded) {

					logger.Info("Creating and submitting batch.", "pending", pending, "time exceeded", timeExceeded, "maxDataExceeded", maxDataExceeded) // TODO: debug level

					// Note that actually submitting the batch can take a while (10 seconds plus), so the actual gap between submission times
					// could exceed the maxBatchTime by this amount. To mitigate, reduce max batch time in config. TODO: handle in code automatically
					nConsumed, err := createAndSubmitBatch(min(pending, maxBatchBytes))
					if err != nil {
						err = fmt.Errorf("create and submit batch: %w", err)
						if errors.Is(err, gerrc.ErrInternal) {
							logger.Error("Create and submit batch", "err", err, "pending", pending)
							panic(err)
						}
						return err
					}
					timeLastSubmission = time.Now()
					pending = uatomic.Uint64Sub(&pendingBytes, nConsumed)
					logger.Info("Submitted a batch to both sub-layers.", "n bytes consumed from pending", nConsumed, "pending after", pending) // TODO: debug level
				} else {
					// done for now, wait for another nudge from trigger thread
					break
				}
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
		return nil, fmt.Errorf(
			"next height to submit is greater than last block height, create and submit batch should not have been called: start height: %d: end height inclusive: %d: %w",
			startHeight,
			endHeightInclusive,
			gerrc.ErrInternal,
		)
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

	for h := startHeight; h <= endHeightInclusive; h++ {
		block, err := m.Store.LoadBlock(h)
		if err != nil {
			return nil, fmt.Errorf("load block: h: %d: %w", h, err)
		}
		commit, err := m.Store.LoadCommit(h)
		if err != nil {
			return nil, fmt.Errorf("load commit: h: %d: %w", h, err)
		}

		batch.Blocks = append(batch.Blocks, block)
		batch.Commits = append(batch.Commits, commit)

		totalSize := batch.SizeBytes()
		if int(maxBatchSize) < totalSize {

			// Remove the last block and commit from the batch
			batch.Blocks = batch.Blocks[:len(batch.Blocks)-1]
			batch.Commits = batch.Commits[:len(batch.Commits)-1]

			if h == startHeight {
				return nil, fmt.Errorf("block size exceeds max batch size: h %d: size: %d: %w", h, totalSize, gerrc.ErrOutOfRange)
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
