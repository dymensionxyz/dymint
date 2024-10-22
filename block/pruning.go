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
	pruned, err := m.P2PClient.RemoveBlocks(context.Background(), retainHeight)
	if err != nil {
		m.logger.Error("pruning blocksync store", "retain_height", retainHeight, "err", err)
	} else {
		m.logger.Debug("blocksync store pruned", "to", retainHeight, "pruned", pruned)
	}

	// prune blocks from indexer store
	baseHeight, err := m.Store.LoadIndexerBaseHeight()
	if err != nil {
		baseHeight = 1
	}
	pruned, err = m.IndexerService.Prune(baseHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning indexer", "retain_height", retainHeight, "err", err)
	} else {
		m.logger.Debug("indexer store pruned", "to", retainHeight, "pruned", pruned)
	}
	err = m.Store.SaveIndexerBaseHeight(retainHeight)
	if err != nil {
		m.logger.Error("saving indexer base height", "err", err)
	}

	// prune blocks from dymint store
	pruned, err = m.Store.PruneStore(retainHeight, m.logger)
	if err != nil {
		m.logger.Error("pruning block store", "retain_height", retainHeight, "err", err)
	} else {
		m.logger.Debug("dymint store pruned", "to", retainHeight, "pruned", pruned)
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
