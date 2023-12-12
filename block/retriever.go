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
func (m *Manager) RetrieveLoop(ctx context.Context) {
	m.logger.Info("Started retrieve loop")
	syncTargetpoller := diodes.NewPoller(m.syncTargetDiode)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get only the latest sync target
			m.logger.Info("Retrieveloop", "synctarget", m.store.Height())
			syncTarget := syncTargetpoller.Next()
			m.logger.Info("Retrieveloop", "synctarget", m.store.Height())
			err := m.syncUntilTarget(ctx, *(*uint64)(syncTarget))
			if err != nil {
				panic(err)
			}
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
func (m *Manager) syncUntilTarget(ctx context.Context, syncTarget uint64) error {
	currentHeight := m.store.Height()
	for currentHeight < syncTarget {
		currStateIdx := atomic.LoadUint64(&m.lastState.SLStateIndex) + 1
		m.logger.Info("Syncing until target", "height", currentHeight, "state_index", currStateIdx, "syncTarget", syncTarget)
		settlementBatch, err := m.settlementClient.RetrieveBatch(currStateIdx)
		if err != nil {
			return err
		}

		if settlementBatch.StartHeight != currentHeight+1 {
			return fmt.Errorf("settlement batch start height (%d) on index (%d) is not the expected", settlementBatch.StartHeight, currStateIdx)
		}

		err = m.processNextDABatch(ctx, settlementBatch.MetaData.DA.Height)
		if err != nil {
			return err
		}

		currentHeight = m.store.Height()
		if currentHeight != settlementBatch.EndHeight {
			return fmt.Errorf("after applying state index (%d), the height (%d) is not as expected (%d)", currStateIdx, currentHeight, settlementBatch.EndHeight)
		}

		err = m.updateStateIndex(settlementBatch.StateIndex)
		if err != nil {
			return err
		}
	}
	return nil
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
	m.logger.Info("Fetching batch")
	batchRes := m.retriever.RetrieveBatches(daHeight)
	switch batchRes.Code {
	case da.StatusError:
		err = fmt.Errorf("failed to retrieve batch from height %d: %s", daHeight, batchRes.Message)
	case da.StatusTimeout:
		err = fmt.Errorf("timeout during retrieve batch from height %d: %s", daHeight, batchRes.Message)
	}

	if len(batchRes.Batches) == 0 {
		err = fmt.Errorf("no batches found on height %d", daHeight)
	}

	return batchRes, err
}
