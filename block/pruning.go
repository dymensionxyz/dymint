package block

import (
	"context"
)

// PruneBlocks prune all block related data from dymint store up to (but not including) retainHeight.
func (m *Manager) PruneBlocks(retainHeight uint64) {
	nextSubmissionHeight := m.NextHeightToSubmit()
	if m.IsProposer() && nextSubmissionHeight < retainHeight { // do not delete anything that we might submit in future
		m.logger.Debug("cannot prune blocks before they have been submitted. using height last submitted height for pruning", "retain_height", retainHeight, "height_to_submit", m.NextHeightToSubmit())
		retainHeight = nextSubmissionHeight
	}

	// prune blocks from blocksync store
	_, err := m.P2PClient.RemoveBlocks(context.Background(), m.State.BaseBlocksyncHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning blocksync store", "retain_height", retainHeight, "err", err)
	}
	m.State.BaseBlocksyncHeight = retainHeight

	// prune blocks from indexer store
	_, err = m.indexerService.Prune(m.State.BaseIndexerHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning indexer", "retain_height", retainHeight, "err", err)
	}

	// prune blocks from dymint store
	_, err = m.Store.PruneStore(m.State.BaseHeight, retainHeight, m.logger)
	if err != nil {
		m.logger.Error("pruning block store", "retain_height", retainHeight, "err", err)
	}

	m.State.BaseHeight = retainHeight
	_, err = m.Store.SaveState(m.State, nil)
	if err != nil {
		m.logger.Error("save state", "err", err)
	}

}

func (m *Manager) PruningLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case retainHeight := <-m.pruningC:
			m.PruneBlocks(uint64(retainHeight))
		}
	}
}
