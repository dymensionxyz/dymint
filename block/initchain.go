package block

import (
	"context"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	tmtypes "github.com/tendermint/tendermint/types"
)

func (m *Manager) RunInitChain(ctx context.Context) error {
	//get the proposer's consensus pubkey
	proposer := m.settlementClient.GetProposer()
	tmPubKey, err := cryptocodec.ToTmPubKeyInterface(proposer.PublicKey)
	if err != nil {
		return err
	}
	consensusPubkey := tmtypes.NewValidator(tmPubKey, 1)

	//get the operator's pubkey
	pubkey, err := getOperatorPubkey(m.conf.OperatorKeyringHomeDir, m.conf.OperatorKeyringBackend, m.conf.OperatorAccountName)
	if err != nil {
		return err
	}
	tmPubKey, err = cryptocodec.ToTmPubKeyInterface(pubkey)
	if err != nil {
		return err
	}
	operatorPubkey := tmtypes.NewValidator(tmPubKey, 1)

	//call initChain with both addresses
	res, err := m.executor.InitChain(m.genesis, []*tmtypes.Validator{consensusPubkey, operatorPubkey})
	if err != nil {
		return err
	}

	//update the state with only the consensus pubkey
	m.executor.UpdateStateAfterInitChain(&m.lastState, res, []*tmtypes.Validator{consensusPubkey})
	if _, err := m.store.UpdateState(m.lastState, nil); err != nil {
		return err
	}

	return nil
}

func getOperatorPubkey(keyDir, keyringBackend, accountName string) (cryptotypes.PubKey, error) {
	// open keyring
	keyring, err := cosmosaccount.New(
		cosmosaccount.WithKeyringBackend(cosmosaccount.KeyringBackend(keyringBackend)),
		cosmosaccount.WithHome(keyDir),
	)
	if err != nil {
		return nil, err
	}

	// Get account from the keyring
	account, err := keyring.GetByName(accountName)
	if err != nil {
		return nil, err
	}

	pubkey, err := account.Record.GetPubKey()
	if err != nil {
		return nil, err
	}

	return pubkey, nil
}
