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
	uchannel "github.com/dymensionxyz/dymint/utils/channel"
)

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
		m.Conf.BatchSubmitTime,
		m.Conf.BatchSubmitBytes,
		m.CreateAndSubmitBatchGetSizeBlocksCommits,
	)
}

func SubmitLoopInner(
	ctx context.Context,
	logger types.Logger,
	bytesProduced chan int,
	maxSkewTime time.Duration,
	unsubmittedBlocksNum func() uint64,
	unsubmittedBlocksBytes func() int,
	batchSkewTime func() time.Duration,
	maxBatchSubmitTime time.Duration,
	maxBatchSubmitBytes uint64,
	createAndSubmitBatch func(maxSizeBytes uint64) (bytes uint64, err error),
) error {
	eg, ctx := errgroup.WithContext(ctx)

	pendingBytes := atomic.Uint64{}

	trigger := uchannel.NewNudger()
	submitter := uchannel.NewNudger()

	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case n := <-bytesProduced:
				pendingBytes.Add(uint64(n))
			}

			submitter.Nudge()

			if maxSkewTime < batchSkewTime() {
				select {
				case <-ctx.Done():
					return nil
				case <-trigger.C:
				}
			}

		}
	})

	eg.Go(func() error {
		ticker := time.NewTicker(maxBatchSubmitTime)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			case <-submitter.C:
			}

			pending := pendingBytes.Load()

			for {
				done := ctx.Err() != nil
				nothingToSubmit := pending == 0

				lastSubmissionIsRecent := batchSkewTime() < maxBatchSubmitTime
				maxDataNotExceeded := pending <= maxBatchSubmitBytes

				UpdateBatchSubmissionGauges(pending, unsubmittedBlocksNum(), batchSkewTime())

				if done || nothingToSubmit || (lastSubmissionIsRecent && maxDataNotExceeded) {
					break
				}

				nConsumed, err := createAndSubmitBatch(maxBatchSubmitBytes)
				if err != nil {
					err = fmt.Errorf("create and submit batch: %w", err)
					if errors.Is(err, gerrc.ErrInternal) {
						panic(err)
					}

					if errors.Is(err, gerrc.ErrAlreadyExists) {
						panic(err)
					}
					return err
				}
				pending = uint64(unsubmittedBlocksBytes())

				if batchSkewTime() < maxSkewTime {
					trigger.Nudge()
				}
			}

			pendingBytes.Store(pending)
		}
	})

	return eg.Wait()
}

func (m *Manager) CreateAndSubmitBatchGetSizeBlocksCommits(maxSize uint64) (uint64, error) {
	b, err := m.CreateAndSubmitBatch(maxSize, false)
	if b == nil {
		return 0, err
	}
	return uint64(b.SizeBlockAndCommitBytes()), err
}

func (m *Manager) CreateAndSubmitBatch(maxSizeBytes uint64, lastBatch bool) (*types.Batch, error) {
	startHeight := m.NextHeightToSubmit()
	endHeightInclusive := m.State.Height()

	if endHeightInclusive < startHeight {
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

	if lastBatch && b.EndHeight() == endHeightInclusive {
		b.LastBatch = true
	}

	types.LastBatchSubmittedBytes.Set(float64(b.SizeBytes()))

	if err := m.SubmitBatch(b); err != nil {
		return nil, fmt.Errorf("submit batch: %w", err)
	}
	return b, nil
}

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
		if err != nil {
			return nil, fmt.Errorf("load drs version: h: %d: %w", h, err)
		}

		if len(batch.Blocks) > 0 && batch.Blocks[len(batch.Blocks)-1].GetRevision() != block.GetRevision() { // should be method
			return nil, fmt.Errorf("create batch: batch includes blocks with different revisions: %w", gerrc.ErrInternal)
		}

		batch.Blocks = append(batch.Blocks, block)
		batch.Commits = append(batch.Commits, commit)
		batch.DRSVersion = append(batch.DRSVersion, drsVersion)

		totalSize := batch.SizeBytes()
		if maxBatchSize < uint64(totalSize) {

			batch.Blocks = batch.Blocks[:len(batch.Blocks)-1]
			batch.Commits = batch.Commits[:len(batch.Commits)-1]
			batch.DRSVersion = batch.DRSVersion[:len(batch.DRSVersion)-1]

			if h == startHeight {
				// internal
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

	err := m.SLClient.SubmitBatch(batch, m.DAClient.GetClientType(), &resultSubmitToDA)
	if err != nil {
		return fmt.Errorf("sl client submit batch: start height: %d: end height: %d: %w", batch.StartHeight(), batch.EndHeight(), err)
	}

	types.RollappHubHeightGauge.Set(float64(batch.EndHeight()))
	m.LastSettlementHeight.Store(batch.EndHeight())

	m.LastBlockTimeInSettlement.Store(batch.Blocks[len(batch.Blocks)-1].Header.GetTimestamp().UTC().UnixNano())

	return err
}

func (m *Manager) GetUnsubmittedBytes() int {
	total := 0

	currH := m.State.Height()

	for h := m.NextHeightToSubmit(); h <= currH; h++ {
		block, err := m.Store.LoadBlock(h)
		if err != nil {
			if !errors.Is(err, gerrc.ErrNotFound) {
			}
			break
		}
		commit, err := m.Store.LoadCommit(h)
		if err != nil {
			if !errors.Is(err, gerrc.ErrNotFound) {
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

func (m *Manager) UpdateLastSubmittedHeight(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatch)
	if !ok {
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

func (m *Manager) GetBatchSkewTime() time.Duration {
	lastProducedTime := time.Unix(0, m.LastBlockTime.Load())
	lastSubmittedTime := time.Unix(0, m.LastBlockTimeInSettlement.Load())
	return lastProducedTime.Sub(lastSubmittedTime)
}

func UpdateBatchSubmissionGauges(skewBytes uint64, skewBlocks uint64, skewTime time.Duration) {
	types.RollappPendingSubmissionsSkewBytes.Set(float64(skewBytes))
	types.RollappPendingSubmissionsSkewBlocks.Set(float64(skewBlocks))
	types.RollappPendingSubmissionsSkewTimeMinutes.Set(float64(skewTime.Minutes()))
}
