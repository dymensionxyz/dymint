package block

import (
	"context"
)

func (m *Manager) Prune(retainHeight uint64) {
	logResult := func(err error, source string, retainHeight uint64, pruned uint64) {
		if err != nil {
		} else {
		}
	}

	pruned, err := m.P2PClient.RemoveBlocks(context.Background(), retainHeight) // ctx background?
	logResult(err, "blocksync", retainHeight, pruned)

	pruned, err = m.IndexerService.Prune(retainHeight, m.Store)
	logResult(err, "indexer", retainHeight, pruned)

	pruned, err = m.Store.PruneStore(retainHeight, m.logger)
	logResult(err, "dymint store", retainHeight, pruned)
}

func (m *Manager) PruningLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case retainHeight := <-m.pruningC:
			var pruningHeight uint64
			if m.RunMode == RunModeProposer {
				pruningHeight = min(m.NextHeightToSubmit(), uint64(retainHeight)) // hopefully it doesn't prune something needed from next height -1
			} else {
				pruningHeight = min(m.SettlementValidator.NextValidationHeight(), uint64(retainHeight))
			}
			m.Prune(pruningHeight)
		}
	}
}
