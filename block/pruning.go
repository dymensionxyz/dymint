package block

import (
	"context"
)

<<<<<<< HEAD
// PruneBlocks prune all block related data from dymint store up to (but not including) retainHeight. It returns the number of blocks pruned, used for testing.
func (m *Manager) PruneBlocks(retainHeight uint64) (uint64, error) {
=======
// PruneBlocks prune all block related data from dymint store up to (but not including) retainHeight.
func (m *Manager) PruneBlocks(retainHeight uint64) {
	nextSubmissionHeight := m.NextHeightToSubmit()
	if m.IsProposer() && nextSubmissionHeight < retainHeight { // do not delete anything that we might submit in future
		m.logger.Debug("cannot prune blocks before they have been submitted. using height last submitted height for pruning", "retain_height", retainHeight, "height_to_submit", m.NextHeightToSubmit())
		retainHeight = nextSubmissionHeight
	}

>>>>>>> daf62fd (separate pruning services)
	// prune blocks from blocksync store
	pruned, err := m.P2PClient.RemoveBlocks(context.Background(), m.State.BaseBlocksyncHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning blocksync store", "retain_height", retainHeight, "err", err)
	} else {
		m.logger.Debug("blocksync store pruned", "from", m.State.BaseBlocksyncHeight, "to", retainHeight, "pruned", pruned)
		m.State.BaseBlocksyncHeight = retainHeight
	}
	// prune blocks from indexer store
	pruned, err = m.IndexerService.Prune(m.State.BaseIndexerHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning indexer", "retain_height", retainHeight, "err", err)
	} else {
		m.logger.Debug("indexer store pruned", "from", m.State.BaseIndexerHeight, "to", retainHeight, "pruned", pruned)
		m.State.BaseIndexerHeight = retainHeight
	}
	// prune blocks from dymint store
	pruned, err = m.Store.PruneStore(m.State.BaseHeight, retainHeight, m.logger)
	if err != nil {
		m.logger.Error("pruning block store", "retain_height", retainHeight, "err", err)
	} else {
		m.logger.Debug("dymint store pruned", "from", m.State.BaseHeight, "to", retainHeight, "pruned", pruned)
		m.State.BaseHeight = retainHeight
	}

	// store state with base heights updates
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
			var pruningHeight uint64
			if m.RunMode == RunModeProposer { // do not delete anything that we might submit in future
				pruningHeight = min(m.NextHeightToSubmit(), uint64(retainHeight))
			} else { // do not delete anything that is not validated yet
				pruningHeight = min(m.SettlementValidator.NextValidationHeight(), uint64(retainHeight))
			}

			_, err := m.PruneBlocks(pruningHeight)
			if err != nil {
				m.logger.Error("pruning blocks", "retainHeight", retainHeight, "err", err)
			}

		}
	}
}
