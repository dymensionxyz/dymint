package block

import (
	"context"
	"fmt"
	"sync/atomic"

	"code.cloudfoundry.org/go-diodes"
	"github.com/dymensionxyz/dymint/da"
)

// RetriveLoop listens for new sync messages written to a ring buffer and in turn
// runs syncUntilTarget on the latest message in the ring buffer.
func (m *Manager) RetriveLoop(ctx context.Context) {
	m.logger.Info("Started retrieve loop")
	syncTargetpoller := diodes.NewPoller(m.syncTargetDiode)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get only the latest sync target
			syncTarget := syncTargetpoller.Next()
			m.syncUntilTarget(ctx, *(*uint64)(syncTarget))
			// Check if after we sync we are synced or a new syncTarget was already set.
			// If we are synced then signal all goroutines waiting on isSyncedCond.
			if m.store.Height() >= atomic.LoadUint64(&m.syncTarget) {
				m.logger.Info("Synced at height", "height", m.store.Height())
				m.isSyncedCond.L.Lock()
				m.isSyncedCond.Signal()
				m.isSyncedCond.L.Unlock()
			}
		}
	}
}

// syncUntilTarget syncs the block until the syncTarget is reached.
// It fetches the batches from the settlement, gets the DA height and gets
// the actual blocks from the DA.
func (m *Manager) syncUntilTarget(ctx context.Context, syncTarget uint64) {
	currentHeight := m.store.Height()
	for currentHeight < syncTarget {
		m.logger.Info("Syncing until target", "current height", currentHeight, "syncTarget", syncTarget)
		resultRetrieveBatch, err := m.settlementClient.RetrieveBatch(atomic.LoadUint64(&m.lastState.SLStateIndex) + 1)
		if err != nil {
			m.logger.Error("Failed to sync until target. error while retrieving batch", "error", err)
			continue
		}
		err = m.processNextDABatch(ctx, resultRetrieveBatch.MetaData.DA.Height)
		if err != nil {
			m.logger.Error("Failed to sync until target. error while processing next DA batch", "error", err)
			break
		}
		err = m.updateStateIndex(resultRetrieveBatch.StateIndex)
		if err != nil {
			return
		}
		currentHeight = m.store.Height()
	}
}

func (m *Manager) updateStateIndex(stateIndex uint64) error {
	atomic.StoreUint64(&m.lastState.SLStateIndex, stateIndex)
	_, err := m.store.UpdateState(m.lastState, nil)
	if err != nil {
		m.logger.Error("Failed to update state", "error", err)
		return err
	}
	return nil
}

func (m *Manager) processNextDABatch(ctx context.Context, daHeight uint64) error {
	m.logger.Debug("trying to retrieve batch from DA", "daHeight", daHeight)
	batchResp, err := m.fetchBatch(daHeight)
	if err != nil {
		m.logger.Error("failed to retrieve batch from DA", "daHeight", daHeight, "error", err)
		return err
	}
	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "daHeight", daHeight)
	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			err := m.applyBlock(ctx, block, batch.Commits[i], blockMetaData{source: daBlock, daHeight: daHeight})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) fetchBatch(daHeight uint64) (da.ResultRetrieveBatch, error) {
	var err error
	batchRes := m.retriever.RetrieveBatches(daHeight)
	switch batchRes.Code {
	case da.StatusError:
		err = fmt.Errorf("failed to retrieve batch: %s", batchRes.Message)
	case da.StatusTimeout:
		err = fmt.Errorf("timeout during retrieve batch: %s", batchRes.Message)
	}
	return batchRes, err
}
