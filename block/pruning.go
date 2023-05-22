package block

import "fmt"

func (m *Manager) pruneBlocks(retainHeight int64) (uint64, error) {
	if retainHeight <= int64(m.store.Base()) {
		return 0, nil
	}
	pruned, err := m.store.PruneBlocks(retainHeight)
	if err != nil {
		return 0, fmt.Errorf("failed to prune block store: %w", err)
	}

	//TODO: prune state/indexer and state/txindexer??

	return pruned, nil
}
