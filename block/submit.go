package block

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/store"
	uevent "github.com/dymensionxyz/dymint/utils/event"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"golang.org/x/sync/errgroup"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
)

func (m *Manager) GetUnsubmittedBytes() int64 {
	total := int64(0)

	/*
		On node start we want to include the count of any blocks which were produced and not submitted in a previous instance
	*/
	currH := m.State.Height()
	for h := m.NextHeightToSubmit(); h <= currH; h++ {
		block := m.MustLoadBlock(h)
		commit := m.MustLoadCommit(h)
		total += int64(block.ToProto().Size()) + int64(commit.ToProto().Size())
	}
	return total
}

// SubmitLoop is the main loop for submitting blocks to the DA and SL layers.
// It submits a batch when either
// 1) It accumulates enough block data, so it's necessary to submit a batch to avoid exceeding the max size
// 2) Enough time passed since the last submitted batch, so it's necessary to submit a batch to avoid exceeding the max time
// It will back pressure (pause) block production if it falls too far behind.
func (m *Manager) SubmitLoop(ctx context.Context, bytesProduced chan int64) (err error) {
	unsubmittedBytes := atomic.Int64{}
	submitC := make(chan struct{}, m.Conf.MaxBatchSkew)

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		// this thread submits batches (consumes bytes)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-submitC:
				nConsumed, err := m.CreateAndSubmitBatch()
				if err != nil {
					return fmt.Errorf("create and submit batch: %w", err)
				}
				n := unsubmittedBytes.Load()
				// The consumption loop may count more than the production loop, because it includes
				// the entire batch data structure, not just the individual blocks and commits.
				// So we must be sure not to underflow.
				nConsumed = min(n, nConsumed)
				unsubmittedBytes.Add(-nConsumed)
			}
		}
	})

	eg.Go(func() error {
		ticker := time.NewTicker(m.Conf.BatchSubmitMaxTime)
		defer ticker.Stop()
		for {
			mustSubmitBatch := false
			select {
			case <-ctx.Done():
				return ctx.Err()
			case n := <-bytesProduced:
				unsubmittedBytes.Add(n)
				mustSubmitBatch = m.Conf.BatchMaxSizeBytes < uint64(unsubmittedBytes.Load())
			case <-ticker.C:
				mustSubmitBatch = true
			}
			if !mustSubmitBatch {
				continue
			}
			pauseBlockProduction := len(submitC) == cap(submitC)
			if pauseBlockProduction {
				evt := &events.DataHealthStatus{Error: fmt.Errorf("submission channel is full: %w", gerrc.ErrResourceExhausted)}
				uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
				m.logger.Error("Enough bytes to build a batch have been accumulated, but too many batches are pending submission.	 " +
					"Pausing block production until a batch submission signal is consumed.")
			}
			submitC <- struct{}{}
			if pauseBlockProduction {
				evt := &events.DataHealthStatus{Error: nil}
				uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
				m.logger.Info("Resumed block production.")
			}
			ticker.Reset(m.Conf.BatchSubmitMaxTime)
		}
	})

	return eg.Wait()
}

func (m *Manager) CreateAndSubmitBatch() (int64, error) {
	batch, err := CreateBatch(m.Store, int(m.Conf.BatchMaxSizeBytes), m.NextHeightToSubmit(), m.State.Height())
	if err != nil {
		return 0, fmt.Errorf("create batch: %w", err)
	}

	if 0 == batch.NumBlocks() {
		return 0, nil
	}

	if err := m.SubmitBatch(batch); err != nil {
		return 0, fmt.Errorf("submit batch: %w", err)
	}
	return int64(batch.SizeBytes()), nil
}

func CreateBatch(store store.Store, maxBatchSize int, startHeight uint64, endHeightInclusive uint64) (*types.Batch, error) {
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
		if maxBatchSize < totalSize {

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
		return fmt.Errorf("submit next batch to da: %s", resultSubmitToDA.Message)
	}
	m.logger.Info("Submitted batch to DA.", "start height", batch.StartHeight(), "end height", batch.EndHeight())

	err := m.SLClient.SubmitBatch(batch, m.DAClient.GetClientType(), &resultSubmitToDA)
	if err != nil {
		return fmt.Errorf("sl client submit batch: start height: %d: inclusive end height: %d: %w", batch.StartHeight(), batch.EndHeight(), err)
	}
	m.logger.Info("Submitted batch to SL.", "start height", batch.StartHeight(), "end height", batch.EndHeight())

	types.RollappHubHeightGauge.Set(float64(batch.EndHeight()))
	m.LastSubmittedHeight.Store(batch.EndHeight())
	return nil
}
