package block

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go/v4"

	"code.cloudfoundry.org/go-diodes"
	"github.com/dymensionxyz/dymint/da"
)

// RetrieveLoop listens for new sync messages written to a ring buffer and in turn
// runs syncUntilTarget on the latest message in the ring buffer.
func (m *Manager) RetrieveLoop(ctx context.Context) {
	m.logger.Info("started retrieve loop")
	syncTargetPoller := diodes.NewPoller(m.SyncTargetDiode, diodes.WithPollingContext(ctx))

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
	currentHeight := m.Store.Height()

	if currentHeight >= syncTarget {
		m.logger.Info("Already synced", "current height", currentHeight, "syncTarget", syncTarget)
		return nil
	}

	var stateIndex uint64
	h := m.Store.Height()
	err := retry.Do(
		func() error {
			res, err := m.SLClient.GetHeightState(h)
			if err != nil {
				m.logger.Debug("sl client get height state", "error", err)
				return err
			}
			stateIndex = res.State.StateIndex
			return nil
		},
		retry.Attempts(0),
		retry.Delay(500*time.Millisecond),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		return fmt.Errorf("get height state: %w", err)
	}
	m.updateStateIndex(stateIndex - 1)
	m.logger.Debug("Sync until target: updated state index pre loop", "stateIndex", stateIndex, "height", h, "syncTarget", syncTarget)

	for currentHeight < syncTarget {
		currStateIdx := atomic.LoadUint64(&m.LastState.SLStateIndex) + 1
		m.logger.Info("Syncing until target", "height", currentHeight, "state_index", currStateIdx, "syncTarget", syncTarget)
		settlementBatch, err := m.SLClient.RetrieveBatch(currStateIdx)
		if err != nil {
			return err
		}

		err = m.ProcessNextDABatch(settlementBatch.MetaData.DA)
		if err != nil {
			return err
		}

		currentHeight = m.Store.Height()

		err = m.updateStateIndex(settlementBatch.StateIndex)
		if err != nil {
			return err
		}
	}
	m.logger.Info("Synced", "current height", currentHeight, "syncTarget", syncTarget)

	// check for cached blocks
	err = m.attemptApplyCachedBlocks()
	if err != nil {
		m.logger.Error("applying previous cached blocks", "err", err)
	}

	return nil
}

func (m *Manager) updateStateIndex(stateIndex uint64) error {
	atomic.StoreUint64(&m.LastState.SLStateIndex, stateIndex)
	_, err := m.Store.UpdateState(m.LastState, nil)
	if err != nil {
		m.logger.Error("update state", "error", err)
		return err
	}
	return nil
}

func (m *Manager) ProcessNextDABatch(daMetaData *da.DASubmitMetaData) error {
	m.logger.Debug("trying to retrieve batch from DA", "daHeight", daMetaData.Height)
	batchResp := m.fetchBatch(daMetaData)
	if batchResp.Code != da.StatusSuccess {
		m.logger.Error("fetching batch from DA", batchResp.Message)
		return batchResp.Error
	}

	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "daHeight", daMetaData.Height)

	m.retrieverMutex.Lock()
	defer m.retrieverMutex.Unlock()

	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			if block.Header.Height != m.Store.NextHeight() {
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

			delete(m.blockCache, block.Header.Height)
		}
	}
	return nil
}

func (m *Manager) fetchBatch(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	// Check DA client
	if daMetaData.Client != m.DAClient.GetClientType() {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("DA client for the batch does not match node config: DA client batch: %s: DA client config: %s", daMetaData.Client, m.DAClient.GetClientType()),
				Error:   ErrWrongDA,
			},
		}
	}
	// Check batch availability
	availabilityRes := m.Retriever.CheckBatchAvailability(daMetaData)
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
	batchRes := m.Retriever.RetrieveBatches(daMetaData)
	// TODO(srene) : for invalid transactions there is no specific error code since it will need to be validated somewhere else for fraud proving.
	// NMT proofs (availRes.MetaData.Proofs) are included in the result batchRes, necessary to be included in the dispute
	return batchRes
}
