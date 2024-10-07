package block

import (
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
)

func (m *Manager) syncFromDABatch() error {

	settlementBatch, err := m.SLClient.GetBatchAtHeight(m.State.NextHeight())
	if err != nil {
		return fmt.Errorf("retrieve batch: %w", err)
	}
	m.logger.Info("Retrieved batch.", "state_index", settlementBatch.StateIndex)

	// update the proposer when syncing from the settlement layer
	proposer := m.State.Sequencers.GetByAddress(settlementBatch.Batch.Sequencer)
	if proposer == nil {
		return fmt.Errorf("proposer not found: batch: %d: %s", settlementBatch.StateIndex, settlementBatch.Batch.Sequencer)
	}
	m.State.Sequencers.SetProposer(proposer)

	err = m.ProcessNextDABatch(settlementBatch.MetaData.DA)
	if err != nil {
		return fmt.Errorf("process next DA batch: %w", err)
	}
	return nil
}

// Used it when doing local rollback, and applying same blocks (instead of producing new ones)
// it was used for an edge case, eg:
// seq produced block H and gossiped
// bug in code produces app mismatch across nodes
// bug fixed, state rolled back to H-1
// if seq produces new block H, it can lead to double signing, as the old block can still be in the p2p network
// ----
// when this scenario encountered previously, we wanted to apply same block instead of producing new one
func (m *Manager) applyLocalBlock(height uint64) error {

	defer m.retrieverMu.Unlock()
	m.retrieverMu.Lock()

	block, err := m.Store.LoadBlock(height)
	if err != nil {
		return fmt.Errorf("load block: %w", gerrc.ErrNotFound)
	}
	commit, err := m.Store.LoadCommit(height)
	if err != nil {
		return fmt.Errorf("load commit: %w", gerrc.ErrNotFound)
	}

	err = m.applyBlockWithFraudHandling(block, commit, types.BlockMetaData{Source: types.LocalDb})
	if err != nil {
		return fmt.Errorf("apply block from local store: height: %d: %w", height, err)
	}

	return nil
}

func (m *Manager) ProcessNextDABatch(daMetaData *da.DASubmitMetaData) error {

	m.logger.Debug("trying to retrieve batch from DA", "daHeight", daMetaData.Height)
	batchResp := m.fetchBatch(daMetaData)
	if batchResp.Code != da.StatusSuccess {
		return batchResp.Error
	}

	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "daHeight", daMetaData.Height)

	m.retrieverMu.Lock()
	defer m.retrieverMu.Unlock()

	var lastAppliedHeight float64
	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			if block.Header.Height != m.State.NextHeight() {
				continue
			}

			// We dont validate because validateBlockBeforeApply already checks if the block is already applied, and we don't need to fail there.
			err := m.applyBlockWithFraudHandling(block, batch.Commits[i], types.BlockMetaData{Source: types.DA, DAHeight: daMetaData.Height})
			if err != nil {
				return fmt.Errorf("apply block: height: %d: %w", block.Header.Height, err)
			}

			lastAppliedHeight = float64(block.Header.Height)

			m.blockCache.Delete(block.Header.Height)

		}
	}
	types.LastReceivedDAHeightGauge.Set(lastAppliedHeight)

	return nil
}

func (m *Manager) fetchBatch(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	// Check DA client
	if daMetaData.Client != m.DAClient.GetClientType() {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("DA client for the batch does not match node config: DA client batch: %s: DA client config: %s", daMetaData.Client, m.DAClient.GetClientType()),
				Error:   da.ErrDAMismatch,
			},
		}
	}

	// batchRes.MetaData includes proofs necessary to open disputes with the Hub
	batchRes := m.Retriever.RetrieveBatches(daMetaData)
	// TODO(srene) : for invalid transactions there is no specific error code since it will need to be validated somewhere else for fraud proving.
	// NMT proofs (availRes.MetaData.Proofs) are included in the result batchRes, necessary to be included in the dispute
	return batchRes
}
