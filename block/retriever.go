package block

import (
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

func (m *Manager) ApplyBatchFromSL(slBatch *settlement.Batch) error {
	m.logger.Debug("trying to retrieve batch from DA", "DA Metadata", slBatch.MetaData)
	batchResp := m.fetchBatch(slBatch.MetaData)
	if batchResp.Code != da.StatusSuccess {
		return batchResp.Error
	}

	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "dametadata", slBatch.MetaData)

	m.retrieverMu.Lock()
	defer m.retrieverMu.Unlock()

	// if batch blocks have already been applied skip, otherwise it will fail in endheight validation (it can happen when syncing from blocksync in parallel).
	if m.State.Height() > slBatch.EndHeight {
		return nil
	}

	blockIndex := 0
	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			// We dont apply a block if not included in the block descriptor (adds support for rollback)
			if blockIndex >= len(slBatch.BlockDescriptors) {
				break
			}
			blockIndex++

			if block.Header.Height != m.State.NextHeight() {
				continue
			}

			if block.GetRevision() != m.State.GetRevision() {
				err := m.checkForkUpdate(fmt.Sprintf("syncing to fork height. received block revision: %d node revision: %d. please restart the node.", block.GetRevision(), m.State.GetRevision()))
				return err
			}

			// We dont validate because validateBlockBeforeApply already checks if the block is already applied, and we don't need to fail there.
			err := m.validateAndApplyBlock(block, batch.Commits[i], types.BlockMetaData{Source: types.DA})
			if err != nil {
				return fmt.Errorf("apply block: height: %d: %w", block.Header.Height, err)
			}

			m.blockCache.Delete(block.Header.Height)
		}
	}

	// validate the batch applied successfully and we are at the end height
	if m.State.Height() != slBatch.EndHeight {
		return fmt.Errorf("state height mismatch: state height: %d: batch end height: %d", m.State.Height(), slBatch.EndHeight)
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
func (m *Manager) applyLocalBlock() error {
	defer m.retrieverMu.Unlock()
	m.retrieverMu.Lock()

	height := m.State.NextHeight()

	block, err := m.Store.LoadBlock(height)
	if err != nil {
		return fmt.Errorf("load block: %w", gerrc.ErrNotFound)
	}

	commit, err := m.Store.LoadCommit(height)
	if err != nil {
		return fmt.Errorf("load commit: %w", gerrc.ErrNotFound)
	}

	source, err := m.Store.LoadBlockSource(height)
	if err != nil {
		return fmt.Errorf("load source: %w", gerrc.ErrNotFound)
	}

	err = m.validateAndApplyBlock(block, commit, types.BlockMetaData{Source: source})
	if err != nil {
		return fmt.Errorf("apply block from local store: height: %d: %w", height, err)
	}

	return nil
}

func (m *Manager) fetchBatch(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	// Check DA client
	var retriever da.BatchRetriever
	for _, client := range m.DAClient {
		if daMetaData.Client == client.GetClientType() {
			retriever = client
			break
		}
	}
	if retriever == nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("DA client for the batch does not match node config: DA client batch: %s", daMetaData.Client),
				Error:   da.ErrDAMismatch,
			},
		}
	}

	// batchRes.MetaData includes proofs necessary to open disputes with the Hub
	batchRes := retriever.RetrieveBatches(daMetaData.DAPath)
	// TODO(srene) : for invalid transactions there is no specific error code since it will need to be validated somewhere else for fraud proving.
	// NMT proofs (availRes.MetaData.Proofs) are included in the result batchRes, necessary to be included in the dispute
	return batchRes
}
