package block

import (
	"fmt"
)

func (m *Manager) pruneBlocks(retainHeight uint64) (uint64, error) {
	syncTarget := m.SyncTarget.Load()

	if retainHeight > syncTarget {
		return 0, fmt.Errorf("cannot prune uncommitted blocks")
	}

	pruned, err := m.Store.PruneBlocks(m.State.BaseHeight, retainHeight)
	if err != nil {
		return 0, fmt.Errorf("prune block store: %w", err)
	}

	// TODO: prune state/indexer and state/txindexer??

	newState := m.State
	newState.BaseHeight = retainHeight
	_, err = m.Store.SaveState(newState, nil)
	if err != nil {
		return 0, fmt.Errorf("final update state: %w", err)
	}
	m.State = newState

	m.logger.Info("pruned blocks", "pruned", pruned, "retain_height", retainHeight)
	return pruned, nil
}
