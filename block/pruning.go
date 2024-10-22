package block

import (
	"context"

	"github.com/dymensionxyz/dymint/types"
)

// PruneBlocks prune all block related data from dymint store up to (but not including) retainHeight.
func (m *Manager) PruneBlocks(retainHeight uint64) {
	nextSubmissionHeight := m.NextHeightToSubmit()
	if m.IsSequencer() && nextSubmissionHeight < retainHeight { // do not delete anything that we might submit in future
		m.logger.Debug("cannot prune blocks before they have been submitted. using height last submitted height for pruning", "retain_height", retainHeight, "height_to_submit", m.NextHeightToSubmit())
		retainHeight = nextSubmissionHeight
	}

	// logging pruning result
	logResult := func(err error, logger types.Logger, retainHeight uint64, pruned uint64) {
		if err != nil {
			m.logger.Error("pruning blocksync store", "retain_height", retainHeight, "err", err)
		} else {
			m.logger.Debug("blocksync store pruned", "to", retainHeight, "pruned", pruned)
		}
	}

	// prune blocks from blocksync store
	pruned, err := m.p2pClient.RemoveBlocks(context.Background(), retainHeight)
	logResult(err, m.logger, retainHeight, pruned)

	// prune blocks from indexer store
	baseHeight, err := m.Store.LoadIndexerBaseHeight()
	if err != nil {
		baseHeight = 1
	}
	pruned, err = m.IndexerService.Prune(baseHeight, retainHeight)
	logResult(err, m.logger, retainHeight, pruned)

	err = m.Store.SaveIndexerBaseHeight(retainHeight)
	if err != nil {
		m.logger.Error("saving indexer base height", "err", err)
	}

	// prune blocks from dymint store
	pruned, err = m.Store.PruneStore(retainHeight, m.logger)
	logResult(err, m.logger, retainHeight, pruned)
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
