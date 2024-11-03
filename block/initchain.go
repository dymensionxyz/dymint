package block

import (
	"context"
	"errors"

	tmtypes "github.com/tendermint/tendermint/types"
)

func (m *Manager) RunInitChain(ctx context.Context) error {
	// FIXME: We want to get the initial proposer and not current one
	proposer := m.SLClient.GetProposer()
	if proposer == nil {
		return errors.New("failed to get proposer")
	}
	tmProposer := proposer.TMValidator()
	res, err := m.Executor.InitChain(m.Genesis, m.GenesisChecksum, []*tmtypes.Validator{tmProposer})
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
