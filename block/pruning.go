package block

import (
	"fmt"
)

func (m *Manager) pruneBlocks(retainHeight int64) (uint64, error) {
	pruned, err := m.Store.PruneBlocks(retainHeight)
	if err != nil {
		return 0, fmt.Errorf("prune block store: %w", err)
	}

	// TODO: prune state/indexer and state/txindexer??

	return pruned, nil
}
