package block

import (
	"context"
)

// Prune function prune all block related data from dymint store and blocksync store up to (but not including) retainHeight.
func (m *Manager) Prune(retainHeight uint64) {
	// logging pruning result
	logResult := func(err error, source string, retainHeight uint64, pruned uint64) {
		if err != nil {
			m.logger.Error("pruning", "from", source, "retain height", retainHeight, "err", err)
		} else {
			m.logger.Debug("pruned", "from", source, "retain height", retainHeight, "pruned", pruned)
		}
	}

	// prune blocks from blocksync store
	pruned, err := m.P2PClient.RemoveBlocks(context.Background(), retainHeight)
	logResult(err, "blocksync", retainHeight, pruned)

	// prune indexed block and txs and associated events
	pruned, err = m.IndexerService.Prune(retainHeight, m.Store)
	logResult(err, "indexer", retainHeight, pruned)

	// prune blocks from dymint store
	pruned, err = m.Store.PruneStore(retainHeight, m.logger)
	logResult(err, "dymint store", retainHeight, pruned)
}

//nolint:gosec // height is non-negative and falls in int64
func (m *Manager) PruningLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case retainHeight := <-m.pruningC:
			var pruningHeight uint64
			if m.RunMode == RunModeProposer { // do not delete anything that we might submit in future
				pruningHeight = min(m.LastSettlementHeight.Load(), uint64(retainHeight))
			} else { // do not delete anything that is not validated yet
				pruningHeight = min(m.SettlementValidator.NextValidationHeight(), uint64(retainHeight))
			}
			m.Prune(pruningHeight)
		}
	}
}
