package block

import (
	"context"
	"fmt"
)

func (m *Manager) PruneBlocks(retainHeight uint64) error {
	if m.IsProposer() && m.NextHeightToSubmit() < retainHeight { // do not delete anything that we might submit in future
		m.logger.Debug("cannot prune blocks before they have been submitted. using height last submitted height for pruning", "retain_height", retainHeight, "height_to_submit", m.NextHeightToSubmit())
		retainHeight = m.NextHeightToSubmit() - 1
	}

	err := m.P2PClient.RemoveBlocks(context.Background(), m.State.BaseHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning blocksync store", "retain_height", retainHeight, "err", err)
	}
	pruned, err := m.Store.PruneBlocks(m.State.BaseHeight, retainHeight, m.logger)
	if err != nil {
		return fmt.Errorf("prune block store: %w", err)
	}

	// TODO: prune state/indexer and state/txindexer??

	m.State.BaseHeight = retainHeight
	_, err = m.Store.SaveState(m.State, nil)
	if err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	m.logger.Info("pruned blocks", "pruned", pruned, "retain_height", retainHeight)
	return nil
}
