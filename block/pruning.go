package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

func (m *Manager) PruneBlocks(retainHeight uint64) error {
	if m.IsSequencer() && m.NextHeightToSubmit() < retainHeight { // do not delete anything that we might submit in future
		return fmt.Errorf("cannot prune blocks before they have been submitted: retain height %d: next height to submit: %d: %w",
			retainHeight,
			m.NextHeightToSubmit(),
			gerrc.ErrInvalidArgument)
	}

	// prune blocks from blocksync store
	err := m.p2pClient.RemoveBlocks(context.Background(), m.State.BaseHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning blocksync store", "retain_height", retainHeight, "err", err)
	}

	// prune blocks from indexer store
	err = m.indexerService.Prune(m.State.BaseHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning indexer", "retain_height", retainHeight, "err", err)
	}

	// prune blocks from dymint store
	pruned, err := m.Store.PruneBlocks(m.State.BaseHeight, retainHeight)
	if err != nil {
		return fmt.Errorf("prune block store: %w", err)
	}

	m.State.BaseHeight = retainHeight
	_, err = m.Store.SaveState(m.State, nil)
	if err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	m.logger.Info("pruned blocks", "pruned", pruned, "retain_height", retainHeight)
	return nil
}

func (m *Manager) PruningLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case retainHeight := <-m.pruningC:
			err := m.PruneBlocks(uint64(retainHeight))
			if err != nil {
				m.logger.Error("pruning blocks", "retainHeight", retainHeight, "err", err)
			}

		}
	}
}
