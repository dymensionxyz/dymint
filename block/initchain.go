package block

import (
	"context"
	"errors"

	tmtypes "github.com/tendermint/tendermint/types"
)

func (m *Manager) RunInitChain(ctx context.Context) error {
	// get the proposer's consensus pubkey
	proposer := m.SLClient.GetProposer()
	if proposer == nil {
		return errors.New("failed to get proposer")
	}
	tmProposer, err := proposer.TMValidator()
	if err != nil {
		return err
	}
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

	targetHeight := uint64(m.Genesis.InitialHeight - 1)
	m.UpdateLastSubmittedHeight(targetHeight)
	m.UpdateTargetHeight(targetHeight)
	return nil
}
