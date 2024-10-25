package block

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/tendermint/tendermint/libs/pubsub"
	"golang.org/x/sync/errgroup"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
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
	bytesProduced chan int,
) (err error) {
	return SubmitLoopInner(
		ctx,
		m.logger,
		bytesProduced,
		m.Conf.BatchSkew,
		m.GetUnsubmittedBlocks,
		m.Conf.BatchSubmitTime,
		m.Conf.BatchSubmitBytes,
		m.CreateAndSubmitBatchGetSizeBlocksCommits,
	)
}

// SubmitLoopInner is a unit testable impl of SubmitLoop
func SubmitLoopInner(
	ctx context.Context,
	logger types.Logger,
	bytesProduced chan int, // a channel of block and commit bytes produced
	maxBatchSkew uint64, // max number of blocks that submitter is allowed to have pending
	unsubmittedBlocks func() uint64,
	maxBatchTime time.Duration, // max time to allow between batches
	maxBatchBytes uint64, // max size of serialised batch in bytes
	createAndSubmitBatch func(maxSizeBytes uint64) (sizeBlocksCommits uint64, err error),
) error {
	eg, ctx := errgroup.WithContext(ctx)

	pendingBytes := atomic.Uint64{}

	trigger := uchannel.NewNudger()   // used to avoid busy waiting (using cpu) on trigger thread
	submitter := uchannel.NewNudger() // used to avoid busy waiting (using cpu) on submitter thread

	eg.Go(func() error {
		// 'trigger': this thread is responsible for waking up the submitter when a new block arrives, and back-pressures the block production loop
		// if it gets too far ahead.
		for {
			if maxBatchSkew*maxBatchBytes < pendingBytes.Load() {
				// too much stuff is pending submission
				// we block here until we get a progress nudge from the submitter thread
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-trigger.C:
				}
			} else {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case n := <-bytesProduced:
					pendingBytes.Add(uint64(n))
					logger.Debug("Added bytes produced to bytes pending submission counter.", "bytes added", n, "pending", pendingBytes.Load())
				}
			}

			types.RollappPendingSubmissionsSkewBytes.Set(float64(pendingBytes.Load()))
			types.RollappPendingSubmissionsSkewBlocks.Set(float64(unsubmittedBlocks()))
			submitter.Nudge()
		}
	})

	eg.Go(func() error {
		// 'submitter': this thread actually creates and submits batches, and will do it on a timer if he isn't nudged by block production
		timeLastSubmission := time.Now()
		ticker := time.NewTicker(maxBatchTime / 10) // interval does not need to match max batch time since we keep track anyway, it's just to wakeup
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			case <-submitter.C:
			}
			pending := pendingBytes.Load()
			types.RollappPendingSubmissionsSkewBytes.Set(float64(pendingBytes.Load()))
			types.RollappPendingSubmissionsSkewBlocks.Set(float64(unsubmittedBlocks()))
			types.RollappPendingSubmissionsSkewBatches.Set(float64(pendingBytes.Load() / maxBatchBytes))

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
						logger.Error("Create and submit batch", "err", err, "pending", pending)
						panic(err)
					}
					// this could happen if we timed-out waiting for acceptance in the previous iteration, but the batch was indeed submitted.
					// we panic here cause restarting may reset the last batch submitted counter and the sequencer can potentially resume submitting batches.
					if errors.Is(err, gerrc.ErrAlreadyExists) {
						logger.Debug("Batch already accepted", "err", err, "pending", pending)
						panic(err)
					}
					return err
				}
				timeLastSubmission = time.Now()
				ticker.Reset(maxBatchTime)
				pending = uatomic.Uint64Sub(&pendingBytes, nConsumed)
				logger.Info("Submitted a batch to both sub-layers.", "n bytes consumed from pending", nConsumed, "pending after", pending) // TODO: debug level
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
	b, err := m.CreateAndSubmitBatch(maxSize, false)
	if b == nil {
		return 0, err
	}
	return uint64(b.SizeBlockAndCommitBytes()), err
}

// CreateAndSubmitBatch creates and submits a batch to the DA and SL.
// max size bytes is the maximum size of the serialized batch type
func (m *Manager) CreateAndSubmitBatch(maxSizeBytes uint64, lastBatch bool) (*types.Batch, error) {
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
	// This is the last batch, so we need to mark it as such
	if lastBatch && b.EndHeight() == endHeightInclusive {
		b.LastBatch = true
	}

	m.logger.Info("Created batch.", "start height", startHeight, "end height", endHeightInclusive, "size", b.SizeBytes(), "last batch", b.LastBatch)
	types.LastBatchSubmittedBytes.Set(float64(b.SizeBytes()))

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

		drsVersion, err := m.DRSVersionHistory.GetDRSVersion(block.Header.Height)
		if err != nil {
			return nil, fmt.Errorf("load drs version: h: %d: %w", h, err)
		}
		batch.Blocks = append(batch.Blocks, block)
		batch.Commits = append(batch.Commits, commit)
		batch.DRSVersion = append(batch.DRSVersion, drsVersion)

		totalSize := batch.SizeBytes()
		if int(maxBatchSize) < totalSize {

			// Remove the last block and commit from the batch
			batch.Blocks = batch.Blocks[:len(batch.Blocks)-1]
			batch.Commits = batch.Commits[:len(batch.Commits)-1]
			batch.DRSVersion = batch.DRSVersion[:len(batch.DRSVersion)-1]

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
	m.LastSettlementHeight.Store(batch.EndHeight())

	// clear drs history for submitted heights
	m.DRSVersionHistory.ClearDRSVersionHeights(batch.EndHeight())
	_, err = m.Store.SaveDRSVersionHistory(m.DRSVersionHistory, nil)
	if err != nil {
		m.logger.Error("save drs history", "error", err)
	}

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
		commit, err := m.Store.LoadCommit(h)
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

func (m *Manager) GetUnsubmittedBlocks() uint64 {
	return m.State.Height() - m.LastSettlementHeight.Load()
}

// UpdateLastSubmittedHeight will update last height submitted height upon events.
// This may be necessary in case we crashed/restarted before getting response for our submission to the settlement layer.
func (m *Manager) UpdateLastSubmittedHeight(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatch)
	if !ok {
		m.logger.Error("onReceivedBatch", "err", "wrong event data received")
		return
	}
	h := eventData.EndHeight

	for {
		curr := m.LastSettlementHeight.Load()
		if m.LastSettlementHeight.CompareAndSwap(curr, max(curr, h)) {
			break
		}
	}
}
