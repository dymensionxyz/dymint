package block

import (
	"context"
	"fmt"
)

func (m *Manager) PruneBlocks(retainHeight uint64) (uint64, error) {
	if m.IsSequencer() && m.NextHeightToSubmit() < retainHeight { // do not delete anything that we might submit in future
		m.logger.Debug("cannot prune blocks before they have been submitted. using height last submitted height for pruning", "retain_height", retainHeight, "height_to_submit", m.NextHeightToSubmit())
		retainHeight = m.NextHeightToSubmit() - 1
		if retainHeight <= m.State.BaseHeight {
			return 0, nil
		}
	}

	// prune blocks from blocksync store
<<<<<<< HEAD
	err := m.p2pClient.RemoveBlocks(context.Background(), m.State.BaseHeight, retainHeight)
=======
	err := m.P2PClient.RemoveBlocks(context.Background(), m.State.BaseHeight, retainHeight)
>>>>>>> 4f5758d (feat(store): indexer pruning (#1061))
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
			pruned, err := m.PruneBlocks(uint64(retainHeight))
			if err != nil {
				m.logger.Error("pruning blocks", "retainHeight", retainHeight, "err", err)
			}
			m.logger.Debug("blocks pruned", "heights pruned", pruned)
		}
	}
}
