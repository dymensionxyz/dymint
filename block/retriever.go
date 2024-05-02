package block

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"

	"code.cloudfoundry.org/go-diodes"
	"github.com/dymensionxyz/dymint/da"
)

// RetrieveLoop listens for new target sync heights and then syncs the chain by
// fetching batches from the settlement layer and then fetching the actual blocks
// from the DA.
func (m *Manager) RetrieveLoop(ctx context.Context) {
	m.logger.Info("started retrieve loop")
	syncTargetPoller := diodes.NewPoller(m.SyncTargetDiode, diodes.WithPollingContext(ctx))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get only the latest sync target
			targetHeight := syncTargetPoller.Next()
			err := m.syncUntilTarget(*(*uint64)(targetHeight))
			if err != nil {
				panic(fmt.Errorf("sync until target: %w", err))
			}
		}
	}
}

// syncUntilTarget syncs blocks until the target height is reached.
// It fetches the batches from the settlement, gets the DA height and gets
// the actual blocks from the DA.
func (m *Manager) syncUntilTarget(targetHeight uint64) error {
	for currH := m.Store.Height(); currH < targetHeight; {

		// It's important that we query the state index before fetching the batch, rather
		// than e.g. keep it and increment it, because we might be concurrently applying blocks
		// and may require a higher index than expected.
		stateIndex, err := m.queryStateIndex()
		if err != nil {
			return fmt.Errorf("query state index: %w", err)
		}

		settlementBatch, err := m.SLClient.RetrieveBatch(stateIndex)
		if err != nil {
			return fmt.Errorf("retrieve batch: %w", err)
		}

		m.logger.Info("Retrieved batch.", "state_index", stateIndex)

		err = m.ProcessNextDABatch(settlementBatch.MetaData.DA)
		if err != nil {
			return fmt.Errorf("process next DA batch: %w", err)
		}

	}

	m.logger.Info("Synced", "store height", m.Store.Height(), "target height", targetHeight)

	err := m.attemptApplyCachedBlocks()
	if err != nil {
		m.logger.Error("Attempt apply cached blocks.", "err", err)
	}

	return nil
}

// TODO: we could encapsulate the retry in the SL client
func (m *Manager) queryStateIndex() (uint64, error) {
	var stateIndex uint64
	return stateIndex, retry.Do(
		func() error {
			res, err := m.SLClient.GetHeightState(m.Store.Height() + 1)
			if err != nil {
				m.logger.Debug("sl client get height state", "error", err)
				return err
			}
			stateIndex = res.State.StateIndex
			return nil
		},
		retry.Attempts(0), // try forever
		retry.Delay(500*time.Millisecond),
		retry.LastErrorOnly(true),
		retry.DelayType(retry.FixedDelay),
	)
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
