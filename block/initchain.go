package block

import (
	"context"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	tmtypes "github.com/tendermint/tendermint/types"
)

func (m *Manager) RunInitChain(ctx context.Context) error {
	// get the proposer's consensus pubkey
	proposer := m.settlementClient.GetProposer()
	tmPubKey, err := cryptocodec.ToTmPubKeyInterface(proposer.PublicKey)
	if err != nil {
		return err
	}
	gensisValSet := []*tmtypes.Validator{tmtypes.NewValidator(tmPubKey, 1)}

	// call initChain with both addresses
	res, err := m.executor.InitChain(m.genesis, gensisValSet)
	if err != nil {
		return err
	}

	// update the state with only the consensus pubkey
	m.executor.UpdateStateAfterInitChain(&m.lastState, res, gensisValSet)
	m.executor.UpdateMempoolAfterInitChain(&m.lastState)

	if _, err := m.store.UpdateState(m.lastState, nil); err != nil {
		return err
	}

	return nil
}
