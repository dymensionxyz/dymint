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
	"github.com/dymensionxyz/dymint/types/metrics"
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
		m.Conf.MaxSkewTime,
		m.GetUnsubmittedBlocks,
		m.GetUnsubmittedBytes,
		m.GetBatchSkewTime,
		m.isLastBatchRecent,
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
	maxSkewTime time.Duration, // max time between last submitted block and last produced block allowed. if this threshold is reached block production is stopped.
	unsubmittedBlocksNum func() uint64, // func that returns the amount of non-submitted blocks
	unsubmittedBlocksBytes func() int, // func that returns bytes from non-submitted blocks
	batchSkewTime func() time.Duration, // func that returns measured time between last submitted block and last produced block
	isLastBatchRecent func(time.Duration) bool, // func that returns true if the last batch submission time is more recent than batch submission time
	maxBatchSubmitTime time.Duration, // max time to allow between batches
	maxBatchSubmitBytes uint64, // max size of serialised batch in bytes
	createAndSubmitBatch func(maxSizeBytes uint64) (bytes uint64, err error),
) error {

	var (
		pendingBytes = atomic.Uint64{}
		trigger      = uchannel.NewNudger() // used to avoid busy waiting (using cpu) on trigger thread
		submitter    = uchannel.NewNudger() // used to avoid busy waiting (using cpu) on submitter thread
	)
	eg, ctx := errgroup.WithContext(ctx)

	// 'trigger': this thread is responsible for waking up the submitter when enough data has been accumulated or time has passed
	// if too much skew been accumulated, we back pressure block production
	eg.Go(func() error {
		ticker := time.NewTicker(maxBatchSubmitTime)
		defer ticker.Stop()

		pending := uint64(0)

		for {
			byTime := false
			select {
			case <-ctx.Done():
				return nil
			case n := <-bytesProduced:
				pending = pendingBytes.Add(uint64(n)) //nolint:gosec // bytes size is always positive
			case <-ticker.C:
				byTime = true
			}

			skewTime := batchSkewTime()
			UpdateBatchSubmissionGauges(pending, unsubmittedBlocksNum(), skewTime)

			byData := pending >= maxBatchSubmitBytes
			bySkew := skewTime >= maxSkewTime

			// check if submission is required by time or size
			if byTime || bySkew || byData {
				submitter.Nudge()
				ticker.Reset(maxBatchSubmitTime)
			}

			// if the time between the last produced block and last submitted is greater than maxSkewTime we block here until we get a progress nudge from the submitter thread
			if bySkew {
				select {
				case <-ctx.Done():
					return nil
				case <-trigger.C:
				}
			}
		}
	})

	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-submitter.C:
			}

			pending := pendingBytes.Load()
			UpdateBatchSubmissionGauges(pending, unsubmittedBlocksNum(), batchSkewTime())

			// while there are accumulated blocks, create and submit batches!!
			for {
				if ctx.Err() != nil {
					return nil
				}

				nConsumed, err := createAndSubmitBatch(maxBatchSubmitBytes)
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
				pending = uint64(unsubmittedBlocksBytes()) //nolint:gosec // bytes size is always positive
				skewTime := batchSkewTime()
				logger.Debug("Submitted a batch to both sub-layers.", "n bytes consumed from pending", nConsumed, "pending after", pending, "skew time", skewTime)

				// after new batch submitted we wake up 'trigger' thread just in case it is blocked due to too much skew
				trigger.Nudge()

				// Check if we need to submit another batch
				byData := pending >= maxBatchSubmitBytes
				bySkew := maxSkewTime < skewTime

				// check if submission is required by skew or data
				if !(bySkew || byData) {
					break
				}
			}
			// update pendingBytes with non submitted block bytes after all pending batches have been submitted
			pendingBytes.Store(pending)
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
	return uint64(b.SizeBlockAndCommitBytes()), err //nolint:gosec // size is always positive and falls in uint64
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

		drsVersion, err := m.Store.LoadDRSVersion(block.Header.Height)
		// if drsVersion is not found in store, batch is submitted using version 0 (it can happen for pending submission blocks for migrated rollapps)
		if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
			return nil, fmt.Errorf("load drs version: h: %d: %w", h, err)
		}

		// check all blocks have the same revision
		if len(batch.Blocks) > 0 && batch.Blocks[len(batch.Blocks)-1].GetRevision() != block.GetRevision() {
			return nil, fmt.Errorf("create batch: batch includes blocks with different revisions: %w", gerrc.ErrInternal)
		}

		batch.Blocks = append(batch.Blocks, block)
		batch.Commits = append(batch.Commits, commit)
		batch.DRSVersion = append(batch.DRSVersion, drsVersion)

		totalSize := batch.SizeBytes()
		if maxBatchSize < uint64(totalSize) { //nolint:gosec // size is always positive and falls in uint64

			// Remove the last block and commit from the batch
			batch.Blocks = batch.Blocks[:len(batch.Blocks)-1]
			batch.Commits = batch.Commits[:len(batch.Commits)-1]
			batch.DRSVersion = batch.DRSVersion[:len(batch.DRSVersion)-1]

			if h == startHeight {
				return nil, fmt.Errorf("block size exceeds max batch size: h %d: batch size: %d: max size: %d err:%w", h, totalSize, maxBatchSize, gerrc.ErrOutOfRange)
			}
			break
		}
	}
	batch.Revision = batch.Blocks[len(batch.Blocks)-1].GetRevision()

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

	// update last submitted block time with batch last block (used to calculate max skew time)
	m.LastSettlementHeight.Store(batch.EndHeight())
	m.LastSubmissionTime.Store(time.Now().UTC().UnixNano())

	// update metrics
	metrics.RollappHubHeightGauge.Set(float64(batch.EndHeight()))
	metrics.LastBatchSubmittedBytes.Set(float64(batch.SizeBytes()))

	return err
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

// GetLastBlockTime returns the time of the last produced block
func (m *Manager) GetLastProducedBlockTime() time.Time {
	lastProducedBlock, err := m.Store.LoadBlock(m.State.Height())
	if err != nil {
		return time.Time{}
	}
	return lastProducedBlock.Header.GetTimestamp()
}

// GetLastBlockTimeInSettlement returns the time of the last submitted block to SL
// If no height in settlement returns first block time
// If no first block produced returns now
// If different error returns empty time
func (m *Manager) GetLastBlockTimeInSettlement() time.Time {
	lastBlockInSettlement, err := m.Store.LoadBlock(m.LastSettlementHeight.Load())
	if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
		return time.Time{}
	}
	if errors.Is(err, gerrc.ErrNotFound) {
		firstBlock, err := m.Store.LoadBlock(uint64(m.Genesis.InitialHeight)) //nolint:gosec // height is non-negative and falls in int64
		if errors.Is(err, gerrc.ErrNotFound) {
			return time.Now()
		}
		if err != nil {
			return time.Time{}
		}
		return firstBlock.Header.GetTimestamp()

	}
	return lastBlockInSettlement.Header.GetTimestamp()
}

// GetBatchSkewTime returns the time between the last produced block and the last block submitted to SL
func (m *Manager) GetBatchSkewTime() time.Duration {
	return m.GetLastProducedBlockTime().Sub(m.GetLastBlockTimeInSettlement())
}

// isLastBatchRecent returns true if the last batch submitted is more recent than maxBatchSubmitTime
// in case of no submission time the first block produced is used as a reference.
func (m *Manager) isLastBatchRecent(maxBatchSubmitTime time.Duration) bool {
	var lastSubmittedTime time.Time
	if m.LastSubmissionTime.Load() == 0 {
		firstBlock, err := m.Store.LoadBlock(uint64(m.Genesis.InitialHeight)) //nolint:gosec // height is non-negative and falls in int64
		if err != nil {
			return true
		} else {
			lastSubmittedTime = firstBlock.Header.GetTimestamp()
		}
	} else {
		lastSubmittedTime = time.Unix(0, m.LastSubmissionTime.Load())
	}
	return time.Since(lastSubmittedTime) < maxBatchSubmitTime
}

func UpdateBatchSubmissionGauges(skewBytes uint64, skewBlocks uint64, skewTime time.Duration) {
	metrics.RollappPendingSubmissionsBytes.Set(float64(skewBytes))
	metrics.RollappPendingSubmissionsBlocks.Set(float64(skewBlocks))
	metrics.RollappPendingSubmissionsSkewTimeMinutes.Set(float64(skewTime.Minutes()))
}
