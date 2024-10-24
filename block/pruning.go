package block

import (
	"context"
	"fmt"
)

// PruneBlocks prune all block related data from dymint store up to (but not including) retainHeight. It returns the number of blocks pruned, used for testing.
func (m *Manager) PruneBlocks(retainHeight uint64) (uint64, error) {

	// prune blocks from blocksync store
	err := m.P2PClient.RemoveBlocks(context.Background(), m.State.BaseHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning blocksync store", "retain_height", retainHeight, "err", err)
	}

	// prune blocks from indexer store
	err = m.indexerService.Prune(m.State.BaseHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning indexer", "retain_height", retainHeight, "err", err)
	}

	// prune blocks from dymint store
	pruned, err := m.Store.PruneStore(m.State.BaseHeight, retainHeight, m.logger)
	if err != nil {
		return 0, fmt.Errorf("prune block store: %w", err)
	}

	m.State.BaseHeight = retainHeight
	_, err = m.Store.SaveState(m.State, nil)
	if err != nil {
		return 0, fmt.Errorf("save state: %w", err)
	}

	m.logger.Info("pruned blocks", "pruned", pruned, "retain_height", retainHeight)

	return pruned, nil
}

func (m *Manager) PruningLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case retainHeight := <-m.pruningC:

			var pruningHeight uint64
			if m.IsProposer() { // do not delete anything that we might submit in future
				pruningHeight = min(m.NextHeightToSubmit(), uint64(retainHeight))
			} else { // do not delete anything that is not validated yet
				pruningHeight = min(m.settlementValidator.NextValidationHeight(), uint64(retainHeight))
			}

			_, err := m.PruneBlocks(pruningHeight)
			if err != nil {
				m.logger.Error("pruning blocks", "retainHeight", retainHeight, "err", err)
			}

		}
	}
}
