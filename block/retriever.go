package block

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

// syncToTargetHeight syncs blocks until the target height is reached.
// It fetches the batches from the settlement, gets the DA height and gets
// the actual blocks from the DA.
func (m *Manager) syncToLastSubmittedHeight() error {

	for currH := m.State.NextHeight(); currH <= m.LastSubmittedHeight.Load(); currH = m.State.NextHeight() {
		// if we have the block locally, we don't need to fetch it from the DA
		err := m.applyLocalBlock(currH)
		if err == nil {
			m.logger.Info("Synced from local", "store height", currH, "target height", m.LastSubmittedHeight.Load())
			continue
		}
		if !errors.Is(err, gerrc.ErrNotFound) {
			m.logger.Error("Apply local block", "err", err)
		}

		err = m.syncFromDABatch()
		if err != nil {
			m.logger.Error("process next DA batch", "err", err)
		}

		m.logger.Info("Synced from DA", "store height", m.State.Height(), "target height", m.LastSubmittedHeight.Load())

	}

	err := m.attemptApplyCachedBlocks()
	if err != nil {
		uevent.MustPublish(context.TODO(), m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
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

	// update the proposer when syncing from the settlement layer
	proposer := m.State.Sequencers.GetByAddress(settlementBatch.Batch.Sequencer)
	if proposer == nil {
		return fmt.Errorf("proposer not found: batch: %d: %s", stateIndex, settlementBatch.Batch.Sequencer)
	}
	m.State.Sequencers.SetProposer(proposer)

	err = m.ProcessNextDABatch(settlementBatch)
	if err != nil {
		return fmt.Errorf("process next DA batch: %w", err)
	}
	return nil
}

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

	err = m.applyBlockWithFraudHandling(block, commit, types.BlockMetaData{Source: types.LocalDb}, true)
	if err != nil {
		return fmt.Errorf("apply block from local store: height: %d: %w", height, err)
	}

	return nil
}

func (m *Manager) ProcessNextDABatch(slBatch *settlement.ResultRetrieveBatch) error {

	batchResp, err := m.validateStateUpdate(slBatch)
	if err != nil {

	}
	m.retrieverMu.Lock()
	defer m.retrieverMu.Unlock()

	var lastAppliedHeight float64
	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			if block.Header.Height != m.State.NextHeight() {
				continue
			}
			if err := m.validateBlockBeforeApply(block, batch.Commits[i]); err != nil {
				m.logger.Error("validate block from DA", "height", block.Header.Height, "err", err)
				continue
			}

			// We dont validate because validateBlockBeforeApply already checks if the block is already applied, and we don't need to fail there.
			err := m.applyBlockWithFraudHandling(block, batch.Commits[i], types.BlockMetaData{Source: types.DA, DAHeight: slBatch.Batch.MetaData.DA.Height}, false)
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
