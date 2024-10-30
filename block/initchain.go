package block

import (
	"context"
	"errors"
	"fmt"

	tmtypes "github.com/tendermint/tendermint/types"
)

func (m *Manager) RunInitChain(ctx context.Context) error {
	// Get the proposer at the initial height
	proposer, err := m.SLClient.GetProposerAtHeight(1)
	if err != nil {
		return fmt.Errorf("get proposer at height: %w", err)
	}
	if proposer == nil {
		return errors.New("failed to get proposer")
	}
	tmProposer := proposer.TMValidator()
	res, err := m.Executor.InitChain(m.Genesis, []*tmtypes.Validator{tmProposer})
	if err != nil {
		return err
	}
	// update the state with only the consensus pubkey
	m.Executor.UpdateStateAfterInitChain(m.State, res)
	m.Executor.UpdateMempoolAfterInitChain(m.State)
	if _, err := m.Store.SaveState(m.State, nil); err != nil {
		return err
	}
	return nil
}
