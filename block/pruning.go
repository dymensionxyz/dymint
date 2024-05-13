package block

import (
	"fmt"
)

func (m *Manager) pruneBlocks(retainHeight int64) (uint64, error) {
	if int64(m.Store.NextHeight()) < retainHeight { // TODO(danwt): check
		return 0, fmt.Errorf("cannot prune uncommitted blocks")
	}

	pruned, err := m.Store.PruneBlocks(retainHeight)
	if err != nil {
		return 0, fmt.Errorf("prune block store: %w", err)
	}

	// TODO: prune state/indexer and state/txindexer??

	return pruned, nil
}
