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
	syncTargetpoller := diodes.NewPoller(m.syncTargetDiode, diodes.WithPollingContext(ctx))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get only the latest sync target
			syncTarget := syncTargetpoller.Next()
			err := m.syncUntilTarget(ctx, *(*uint64)(syncTarget))
			if err != nil {
				panic(err)
			}
			// Check if after we sync we are synced or a new syncTarget was already set.
			// If we are synced then signal all goroutines waiting on isSyncedCond.
			if m.store.Height() >= m.syncTarget.Load() {
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

		err = m.processNextDABatch(ctx, settlementBatch.MetaData.DA)
		if err != nil {
			return err
		}

		currentHeight = m.store.Height()

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
		m.logger.Error("update state", "error", err)
		return err
	}
	return nil
}

func (m *Manager) processNextDABatch(ctx context.Context, daMetaData *da.DASubmitMetaData) error {
	m.logger.Debug("trying to retrieve batch from DA", "daHeight", daMetaData.Height)
	batchResp := m.fetchBatch(daMetaData)
	if batchResp.Code != da.StatusSuccess {
		return batchResp.Error
	}

	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "daHeight", daMetaData.Height)

	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			if block.Header.Height != m.store.NextHeight() {
				continue
			}
			err := m.applyBlock(ctx, block, batch.Commits[i], blockMetaData{source: daBlock, daHeight: daMetaData.Height})
			if err != nil {
				return err
			}
		}
	}

	err := m.attemptApplyCachedBlocks(ctx)
	if err != nil {
		m.logger.Debug("Error applying previous cached blocks", "err", err)
	}
	return nil
}

func (m *Manager) fetchBatch(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	// Check batch availability
	availabilityRes := m.retriever.CheckBatchAvailability(daMetaData)
	if availabilityRes.Code != da.StatusSuccess {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Error fetching batch: %s", availabilityRes.Message),
				Error:   availabilityRes.Error,
			},
		}
	}
	// batchRes.MetaData includes proofs necessary to open disputes with the Hub
	batchRes := m.retriever.RetrieveBatches(daMetaData)
	// TODO(srene) : for invalid transactions there is no specific error code since it will need to be validated somewhere else for fraud proving.
	// NMT proofs (availRes.MetaData.Proofs) are included in the result batchRes, necessary to be included in the dispute
	return batchRes
}
