package block

import (
	"fmt"
	"sync/atomic"
)

func (m *Manager) pruneBlocks(retainHeight int64) (uint64, error) {
	syncTarget := atomic.LoadUint64(&m.SyncTarget)

	if retainHeight > int64(syncTarget) {
		return 0, fmt.Errorf("cannot prune uncommitted blocks")
	}

	pruned, err := m.Store.PruneBlocks(retainHeight)
	if err != nil {
		return 0, fmt.Errorf("failed to prune block store: %w", err)
	}

	//TODO: prune state/indexer and state/txindexer??

	return pruned, nil
}
