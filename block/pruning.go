package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

func (m *Manager) pruneBlocks(retainHeight uint64) error {
	if m.IsSequencer() && m.NextHeightToSubmit() < retainHeight { // do not delete anything that we might submit in future
		return fmt.Errorf("cannot prune blocks before they have been submitted: retain height %d: next height to submit: %d: %w",
			retainHeight,
			m.NextHeightToSubmit(),
			gerrc.ErrInvalidArgument)
	}

	err := m.p2pClient.RemoveBlocks(context.Background(), m.State.BaseHeight, retainHeight)
	if err != nil {
		return fmt.Errorf("pruning blocksync store: %w", err)
	}
	pruned, err := m.Store.PruneBlocks(m.State.BaseHeight, retainHeight)
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
