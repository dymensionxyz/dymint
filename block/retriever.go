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
		m.logger.Error("Failed to update state", "error", err)
		return err
	}
	return nil
}

func (m *Manager) processNextDABatch(ctx context.Context, daMetaData *da.DASubmitMetaData) error {
	m.logger.Debug("trying to retrieve batch from DA", "daHeight", daMetaData.Height)
	batchResp, err := m.fetchBatch(daMetaData)
	if err != nil {
		return err
	}

	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "daHeight", daMetaData.Height)

	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			err := m.applyBlock(ctx, block, batch.Commits[i], blockMetaData{source: daBlock, daHeight: daMetaData.Height})
			if err != nil {
				return err
			}
		}
	}
	err = m.attemptApplyCachedBlocks(ctx)
	if err != nil {
		m.logger.Debug("Error applying previous cached blocks", "err", err)
	}
	return nil
}

func (m *Manager) fetchBatch(daMetaData *da.DASubmitMetaData) (da.ResultRetrieveBatch, error) {
	var err error
	availRes := m.retriever.CheckBatchAvailability(daMetaData)

	switch availRes.Code {
	case da.StatusUnableToGetProofs:
		//Is not possible to obtain proofs for the specific commitment in the specific height.
		//This means there is no matching blob for that commitment
		err = fmt.Errorf("Unable to get proof on height %d for the commitment", daMetaData.Height)

	case da.StatusProofNotMatching:
		//The proofs are obtained for the commitment, but not matching with the span (index, length) committed to the Hub.
		err = fmt.Errorf("Span not matching the commitment")
	case da.StatusError:
		//There's been an issue validating proofs
		err = fmt.Errorf("Error validating batch")
	}
	if !availRes.DataAvailable {
		//There is no point on fetching the data
		batchRes := da.ResultRetrieveBatch{}
		batchRes.Code = da.StatusError
		batchRes.Message = "Error validating data"
		batchRes.CheckMetaData = availRes.CheckMetaData
		if batchRes.CheckMetaData != nil {
			batchRes.CheckMetaData.SLIndex = atomic.LoadUint64(&m.lastState.SLStateIndex)
		}
		return batchRes, err

	}
	//batchRes.MetaData includes proofs necessary to open disputes with the Hub
	batchRes := m.retriever.RetrieveBatches(daMetaData)

	switch batchRes.Code {
	case da.StatusError:
		err = fmt.Errorf("failed to retrieve batch from height %d: %s", daMetaData.Height, batchRes.Message)
	case da.StatusTimeout:
		err = fmt.Errorf("timeout during retrieve batch from height %d: %s", daMetaData.Height, batchRes.Message)
	//TODO (srene) : Send dispute to the Hub for these cases
	case da.StatusBlobNotFound:
		//The blob is not found for the specified share commitment
		err = fmt.Errorf("blob not found for the specified commitment %d: %s", daMetaData.Height, batchRes.Message)

	}

	if len(batchRes.Batches) == 0 {
		err = fmt.Errorf("no batches found on height %d", daMetaData.Height)
	}
	if err == nil {
		batchRes.CheckMetaData = availRes.CheckMetaData
	}
	//TODO(srene) : for invalid transactions there is no specific error code since it will need to be validated somewhere else for fraud proving.
	//NMT proofs (availRes.MetaData.Proofs) are included in the result batchRes, necessary to be included in the dispute
	return batchRes, err
}
