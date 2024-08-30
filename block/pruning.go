package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

func (m *Manager) PruneBlocks(retainHeight uint64) error {
	if m.IsProposer() && m.NextHeightToSubmit() < retainHeight { // do not delete anything that we might submit in future
		m.logger.Error("skipping block pruning. next height to submit is previous to retain_height.", "retain_height", retainHeight, "next_submit_height", m.NextHeightToSubmit())
		return fmt.Errorf("skipping block pruning. next height to submit is previous to retain_height.t %d: next height to submit: %d: %w",
			retainHeight,
			m.NextHeightToSubmit(),
			gerrc.ErrInvalidArgument)
	}

	//
	err := m.P2PClient.RemoveBlocks(context.Background(), m.State.BaseHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning blocksync store", "retain_height", retainHeight, "err", err)
	}

	pruned, err := m.Store.PruneBlocks(m.State.BaseHeight, retainHeight)
	if err != nil {
		m.logger.Error("pruning dymint store", "retain_height", retainHeight, "err", err)
		return fmt.Errorf("pruning dymint store: %w", err)
	}

	// TODO: prune state/indexer and state/txindexer??

	m.State.BaseHeight = retainHeight
	_, err = m.Store.SaveState(m.State, nil)
	if err != nil {
		m.logger.Error("saving state.", "retain_height", retainHeight, "err", err)
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
