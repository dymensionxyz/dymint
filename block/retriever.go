package block

import (
	"context"
	"fmt"

	"code.cloudfoundry.org/go-diodes"

	"github.com/dymensionxyz/dymint/da"
)

// RetrieveLoop listens for new target sync heights and then syncs the chain by
// fetching batches from the settlement layer and then fetching the actual blocks
// from the DA.
func (m *Manager) RetrieveLoop(ctx context.Context) {
	m.logger.Info("Started retrieve loop.")
	p := diodes.NewPoller(m.targetSyncHeight, diodes.WithPollingContext(ctx))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			targetHeight := p.Next() // We only care about the latest one
			err := m.syncToTargetHeight(*(*uint64)(targetHeight))
			if err != nil {
				panic(fmt.Errorf("sync until target: %w", err))
			}
		}
	}
}

// syncToTargetHeight syncs blocks until the target height is reached.
// It fetches the batches from the settlement, gets the DA height and gets
// the actual blocks from the DA.
func (m *Manager) syncToTargetHeight(targetHeight uint64) error {
	for currH := m.State.NextHeight(); currH <= targetHeight; currH = m.State.NextHeight() {
		// if we have the block locally, we don't need to fetch it from the DA
		err := m.processLocalBlock(currH)
		if err == nil {
			m.logger.Info("Synced from local", "store height", currH, "target height", targetHeight)
			continue
		}

		err = m.syncFromDABatch()
		if err != nil {
			return fmt.Errorf("process next DA batch: %w", err)
		}
		m.logger.Info("Synced from DA", "store height", m.State.Height(), "target height", targetHeight)
	}

	err := m.attemptApplyCachedBlocks()
	if err != nil {
		m.logger.Error("Attempt apply cached blocks.", "err", err)
	}

	return nil
}

func (m *Manager) syncFromDABatch() error {
	// It's important that we query the state index before fetching the batch, rather
	// than e.g. keep it and increment it, because we might be concurrently applying blocks
	// and may require a higher index than expected.
	res, err := m.SLClient.GetHeightState(m.State.NextHeight())
	if err != nil {
		return fmt.Errorf("retrieve state: %w", err)
	}
	stateIndex := res.State.StateIndex

	settlementBatch, err := m.SLClient.GetBatchAtIndex(stateIndex)
	if err != nil {
		return fmt.Errorf("retrieve batch: %w", err)
	}

	m.logger.Info("Retrieved batch.", "state_index", stateIndex)

	err = m.ProcessNextDABatch(settlementBatch.MetaData.DA)
	if err != nil {
		return fmt.Errorf("process next DA batch: %w", err)
	}
	return nil
}

func (m *Manager) processLocalBlock(height uint64) error {
	block, err := m.Store.LoadBlock(height)
	if err != nil {
		return err
	}
	commit, err := m.Store.LoadCommit(height)
	if err != nil {
		return err
	}
	if err := m.validateBlock(block, commit); err != nil {
		return fmt.Errorf("validate block from local store: height: %d: %w", height, err)
	}

	m.retrieverMu.Lock()
	err = m.applyBlock(block, commit, blockMetaData{source: localDbBlock})
	if err != nil {
		return fmt.Errorf("apply block from local store: height: %d: %w", height, err)
	}
	m.retrieverMu.Unlock()

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

	m.retrieverMu.Lock()
	defer m.retrieverMu.Unlock()

	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			if block.Header.Height != m.State.NextHeight() {
				continue
			}
			if err := m.validateBlock(block, batch.Commits[i]); err != nil {
				m.logger.Error("validate block from DA", "height", block.Header.Height, "err", err)
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

	// batchRes.MetaData includes proofs necessary to open disputes with the Hub
	batchRes := m.Retriever.RetrieveBatches(daMetaData)
	// TODO(srene) : for invalid transactions there is no specific error code since it will need to be validated somewhere else for fraud proving.
	// NMT proofs (availRes.MetaData.Proofs) are included in the result batchRes, necessary to be included in the dispute
	return batchRes
}
