package block

import (
	"context"
	"fmt"
	"sync/atomic"

	"code.cloudfoundry.org/go-diodes"
	"github.com/dymensionxyz/dymint/da"
)

// RetrieveLoop listens for new sync messages written to a ring buffer and in turn
// runs syncUntilTarget on the latest message in the ring buffer.
func (m *Manager) RetrieveLoop(ctx context.Context) {
	m.logger.Info("started retrieve loop")
	syncTargetPoller := diodes.NewPoller(m.syncTargetDiode, diodes.WithPollingContext(ctx))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get only the latest sync target
			syncTarget := syncTargetPoller.Next()
			err := m.syncUntilTarget(*(*uint64)(syncTarget))
			if err != nil {
				panic(fmt.Errorf("sync until target: %w", err))
			}
		}
	}
}

// syncUntilTarget syncs the block until the syncTarget is reached.
// It fetches the batches from the settlement, gets the DA height and gets
// the actual blocks from the DA.
func (m *Manager) syncUntilTarget(syncTarget uint64) error {
	currentHeight := m.store.Height()

	if currentHeight >= syncTarget {
		m.logger.Info("Already synced", "current height", currentHeight, "syncTarget", syncTarget)
		return nil
	}

	for currentHeight < syncTarget {
		currStateIdx := atomic.LoadUint64(&m.lastState.SLStateIndex) + 1
		m.logger.Info("Syncing until target", "height", currentHeight, "state_index", currStateIdx, "syncTarget", syncTarget)
		settlementBatch, err := m.settlementClient.RetrieveBatch(currStateIdx)
		if err != nil {
			return err
		}

		err = m.processNextDABatch(settlementBatch.MetaData.DA)
		if err != nil {
			return err
		}

		currentHeight = m.store.Height()

		err = m.updateStateIndex(settlementBatch.StateIndex)
		if err != nil {
			return err
		}

	}
	// check for cached blocks
	err := m.attemptApplyCachedBlocks(ctx)
	if err != nil {
		m.logger.Debug("Error applying previous cached blocks", "err", err)
	}

	m.logger.Info("Synced", "current height", currentHeight, "syncTarget", syncTarget)
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

func (m *Manager) processNextDABatch(daMetaData *da.DASubmitMetaData) error {
	m.logger.Debug("trying to retrieve batch from DA", "daHeight", daMetaData.Height)
	batchResp := m.fetchBatch(daMetaData)
	if batchResp.Code != da.StatusSuccess {
		return batchResp.Error
	}

	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "daHeight", daMetaData.Height)

	m.executeBlockMutex.Lock()
	defer m.executeBlockMutex.Unlock()

	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			if block.Header.Height != m.store.NextHeight() {
				continue
			}
			if err := m.validateBlock(block, batch.Commits[i]); err != nil {
				m.logger.Error("validate block from DA - someone is behaving badly", "height", block.Header.Height, "err", err)
				continue
			}
			err := m.applyBlock(block, batch.Commits[i], blockMetaData{source: daBlock, daHeight: daMetaData.Height})
			if err != nil {
				return fmt.Errorf("apply block: height: %d: %w", block.Header.Height, err)
			}
		}
	}

	err := m.attemptApplyCachedBlocks()
	if err != nil {
		m.logger.Error("applying previous cached blocks", "err", err)
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

func (m *Manager) attemptApplyCachedBlocks(ctx context.Context) error {
	m.executeBlockMutex.Lock()
	defer m.executeBlockMutex.Unlock()

	for {
		expectedHeight := m.store.NextHeight()

		prevCachedBlock, blockExists := m.prevBlock[expectedHeight]
		prevCachedCommit, commitExists := m.prevCommit[expectedHeight]

		if !blockExists || !commitExists {
			break
		}

		m.logger.Debug("Applying cached block", "height", expectedHeight)
		err := m.applyBlock(ctx, prevCachedBlock, prevCachedCommit, blockMetaData{source: gossipedBlock})
		if err != nil {
			m.logger.Debug("apply previously cached block", "err", err)
			return err
		}
	}

	for k := range m.prevBlock {
		if k <= m.store.Height() {
			delete(m.prevBlock, k)
			delete(m.prevCommit, k)
		}
	}
	return nil
}
