package block

import (
	"fmt"
)

func (m *Manager) pruneBlocks(retainHeight uint64) (uint64, error) {
	pruned, err := m.Store.PruneBlocks(m.State.BaseHeight, retainHeight)
	if err != nil {
		return 0, fmt.Errorf("prune block store: %w", err)
	}

	// TODO: prune state/indexer and state/txindexer??

	m.State.BaseHeight = retainHeight
	_, err = m.Store.SaveState(m.State, nil)
	if err != nil {
		return 0, fmt.Errorf("save state: %w", err)
	}

	m.logger.Info("pruned blocks", "pruned", pruned, "retain_height", retainHeight)
	return pruned, nil
}
