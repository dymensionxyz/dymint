package block

import (
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

func (m *Manager) ApplyBatchFromSL(slBatch *settlement.Batch) error {
	m.logger.Debug("trying to retrieve batch from DA", "daHeight", slBatch.MetaData.DA.Height)
	batchResp := m.fetchBatch(slBatch.MetaData.DA)
	if batchResp.Code != da.StatusSuccess {
		return batchResp.Error
	}

	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "daHeight", slBatch.MetaData.DA.Height)

	m.retrieverMu.Lock()
	defer m.retrieverMu.Unlock()

	if m.State.Height() > slBatch.EndHeight {
		return nil
	}

	blockIndex := 0
	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {

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

			err := m.applyBlockWithFraudHandling(block, batch.Commits[i], types.BlockMetaData{Source: types.DA, DAHeight: slBatch.MetaData.DA.Height})
			if err != nil {
				return fmt.Errorf("apply block: height: %d: %w", block.Header.Height, err)
			}

			m.blockCache.Delete(block.Header.Height)
		}
	}

	if m.State.Height() != slBatch.EndHeight {
		return fmt.Errorf("state height mismatch: state height: %d: batch end height: %d", m.State.Height(), slBatch.EndHeight)
	}

	return nil
}

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

	err = m.applyBlockWithFraudHandling(block, commit, types.BlockMetaData{Source: source})
	if err != nil {
		return fmt.Errorf("apply block from local store: height: %d: %w", height, err)
	}

	return nil
}

func (m *Manager) fetchBatch(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	if daMetaData.Client != m.DAClient.GetClientType() {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("DA client for the batch does not match node config: DA client batch: %s: DA client config: %s", daMetaData.Client, m.DAClient.GetClientType()),
				Error:   da.ErrDAMismatch,
			},
		}
	}

	batchRes := m.Retriever.RetrieveBatches(daMetaData)

	return batchRes
}
