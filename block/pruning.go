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

	return pruned, nil
}
