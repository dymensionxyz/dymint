package block

import (
	"context"
)

// PruneBlocks prune all block related data from dymint store up to (but not including) retainHeight.
func (m *Manager) PruneBlocks(retainHeight uint64) {
	nextSubmissionHeight := m.NextHeightToSubmit()
	if m.IsSequencer() && nextSubmissionHeight < retainHeight { // do not delete anything that we might submit in future
		m.logger.Debug("cannot prune blocks before they have been submitted. using height last submitted height for pruning", "retain_height", retainHeight, "height_to_submit", m.NextHeightToSubmit())
		retainHeight = nextSubmissionHeight
	}

	// logging pruning result
	logResult := func(err error, source string, retainHeight uint64, pruned uint64) {
		if err != nil {
			m.logger.Error("pruning", "from", source, "retain_height", retainHeight, "err", err)
		} else {
			m.logger.Debug("pruned", "from", source, "retain_height", retainHeight, "pruned", pruned)
		}
	}

	// prune blocks from blocksync store
	pruned, err := m.P2PClient.RemoveBlocks(context.Background(), retainHeight)
	logResult(err, "blocksync", retainHeight, pruned)

	// prune blocks from indexer store
	// load indexer base height
	indexerBaseHeight, err := m.Store.LoadIndexerBaseHeight()
	if err != nil {
		indexerBaseHeight = 1
	}
	pruned, err = m.IndexerService.Prune(indexerBaseHeight, retainHeight)
	logResult(err, "indexer", retainHeight, pruned)

	// store indexer base height
	err = m.Store.SaveIndexerBaseHeight(retainHeight)
	if err != nil {
		m.logger.Error("saving indexer base height", "err", err)
	}

	// prune blocks from dymint store
	pruned, err = m.Store.PruneStore(retainHeight, m.logger)
	logResult(err, "dymint store", retainHeight, pruned)
}

func (m *Manager) PruningLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case retainHeight := <-m.pruningC:

			var pruningHeight uint64
			if m.IsSequencer() { // do not delete anything that we might submit in future
				pruningHeight = min(m.NextHeightToSubmit(), uint64(retainHeight))
			}
			m.PruneBlocks(pruningHeight)
		}
	}
}
