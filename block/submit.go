package block

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	uchannel "github.com/dymensionxyz/dymint/utils/channel"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"golang.org/x/sync/errgroup"
)

// SubmitLoop is the main loop for submitting blocks to the DA and SL layers.
// It submits a batch when either
// 1) It accumulates enough block data, so it's necessary to submit a batch to avoid exceeding the max size
// 2) Enough time passed since the last submitted batch, so it's necessary to submit a batch to avoid exceeding the max time
// It will back pressure (pause) block production if it falls too far behind.
func (m *Manager) SubmitLoop(ctx context.Context,
	bytesProduced chan int,
) (err error) {
	return SubmitLoopInner(ctx,
		bytesProduced,
		m.Conf.MaxBatchSkew,
		m.Conf.BatchSubmitMaxTime,
		m.Conf.BatchMaxSizeBytes,
		m.CreateAndSubmitBatchGetSizeEstimate,
	)
}

// SubmitLoopInner is a unit testable impl of SubmitLoop
func SubmitLoopInner(ctx context.Context,
	bytesProduced chan int, // a channel of block and commit bytes produced
	maxBatchSkew uint64, // max number of batches that submitter is allowed to have pending
	maxBatchTime time.Duration, // max time to allow between batches
	maxBatchBytes uint64, // max size of serialised batch in bytes
	createAndSubmitBatchGetSizeEstimate func(maxSize uint64) (uint64, error),
) error {
	eg, ctx := errgroup.WithContext(ctx)

	pendingBytes := atomic.Uint64{}
	counter := uchannel.NewNudger()   // used to avoid busy waiting (using cpu) on counter thread
	submitter := uchannel.NewNudger() // used to avoid busy waiting (using cpu) on submitter thread

	eg.Go(func() error {
		// 'counter': we need one thread to continuously consume the bytes produced channel, and to monitor timer
		ticker := time.NewTicker(maxBatchTime)
		defer ticker.Stop()
		for {
			if maxBatchSkew*maxBatchBytes < pendingBytes.Load() {
				// too much stuff is pending submission
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-counter.C: // await a nudge from the submitter to know he has made progress
				case <-ticker.C:
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
			for ctx.Err() == nil && (0 < pending && maxBatchTime < time.Since(timeLastSubmission) || maxBatchBytes < pending) {
				nConsumed, err := createAndSubmitBatchGetSizeEstimate(min(pending, maxBatchBytes))
				if err != nil {
					return fmt.Errorf("create and submit batch: %w", err)
				}
				timeLastSubmission = time.Now()
				fmt.Println(fmt.Sprintf("pending: %d: consumed: %d: ", pending, nConsumed))
				pending = pendingBytes.Add(^(nConsumed - 1)) // subtract
			}
			counter.Nudge()
		}
	})

	return eg.Wait()
}

func (m *Manager) CreateAndSubmitBatchGetSizeEstimate(maxSize uint64) (uint64, error) {
	b, err := m.CreateAndSubmitBatch(maxSize)
	if b == nil {
		return 0, err
	}
	return uint64(b.SizeBytesEstimate()), err
}

// CreateAndSubmitBatch creates and submits a batch to the DA and SL.
func (m *Manager) CreateAndSubmitBatch(maxSize uint64) (*types.Batch, error) {
	b, err := CreateBatch(m.Store, maxSize, m.NextHeightToSubmit(), m.State.Height())
	if err != nil {
		return nil, fmt.Errorf("create batch: %w", err)
	}

	if 0 == b.NumBlocks() {
		return nil, nil
	}

	if err := m.SubmitBatch(b); err != nil {
		return nil, fmt.Errorf("submit batch: %w", err)
	}
	return b, nil
}

func CreateBatch(store store.Store, maxBatchSize uint64, startHeight uint64, endHeightInclusive uint64) (*types.Batch, error) {
	batchSize := endHeightInclusive - startHeight + 1
	batch := &types.Batch{
		Blocks:  make([]*types.Block, 0, batchSize),
		Commits: make([]*types.Commit, 0, batchSize),
	}

	for height := startHeight; height <= endHeightInclusive; height++ {
		block, err := store.LoadBlock(height)
		if err != nil {
			return nil, fmt.Errorf("load block: height: %d: %w", height, err)
		}
		commit, err := store.LoadCommit(height)
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
		return fmt.Errorf("da client submit batch: %s", resultSubmitToDA.Message)
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

// GetUnsubmittedBytes returns the total number of unsubmitted bytes produced
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
