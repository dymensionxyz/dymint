package block

import (
	"context"
	"errors"
	"fmt"
	"time"

	tmtypes "github.com/tendermint/tendermint/types"
)

func (m *Manager) RunInitChain(ctx context.Context) error {
	// Get the proposer at the initial height. If we're at genesis the height will be 0.
	proposer, err := m.SLClient.GetProposerAtHeight(int64(m.State.Height()) + 1)
	if err != nil {
		return fmt.Errorf("get proposer at height: %w", err)
	}
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

	err = m.SetLastSettlementBlockTime(time.Now())
	if err != nil {
		return err
	}
	return nil
}
