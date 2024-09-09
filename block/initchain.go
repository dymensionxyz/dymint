package block

import (
	"errors"
)

func (m *Manager) RunInitChain() error {
	// get the proposer's consensus pubkey
	proposer := m.SLClient.GetProposer()
	if proposer == nil {
		return errors.New("failed to get proposer")
	}

	res, err := m.Executor.InitChain(m.Genesis, proposer)
	if err != nil {
		return err
	}
	// update the state with only the consensus pubkey
	m.Executor.UpdateStateAfterInitChain(m.State, res)
	m.Executor.UpdateMempoolAfterInitChain(m.State)
	if _, err = m.Store.SaveState(m.State, nil); err != nil {
		return err
	}
	return nil
}
