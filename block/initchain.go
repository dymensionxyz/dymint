package block

import (
	"context"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	tmtypes "github.com/tendermint/tendermint/types"
)

func (m *Manager) RunInitChain(ctx context.Context) error {
	// get the proposer's consensus pubkey
	proposer := m.SLClient.GetProposer()
	tmPubKey, err := cryptocodec.ToTmPubKeyInterface(proposer.PublicKey)
	if err != nil {
		return err
	}
	gensisValSet := []*tmtypes.Validator{tmtypes.NewValidator(tmPubKey, 1)}

	// call initChain with both addresses
	res, err := m.Executor.InitChain(m.Genesis, gensisValSet)
	if err != nil {
		return err
	}

	// update the state with only the consensus pubkey
	m.Executor.UpdateStateAfterInitChain(m.State, res, gensisValSet)
	m.Executor.UpdateMempoolAfterInitChain(m.State)
	if _, err := m.Store.SaveState(m.State, nil); err != nil {
		return err
	}

	return nil
}
