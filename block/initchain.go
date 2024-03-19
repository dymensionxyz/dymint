package block

import (
	"context"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	tmtypes "github.com/tendermint/tendermint/types"
)

func (m *Manager) RunInitChain(ctx context.Context) error {
	proposer := m.settlementClient.GetProposer()

	tmPubKey, err := cryptocodec.ToTmPubKeyInterface(proposer.PublicKey)
	if err != nil {
		return err
	}
	consensusPubkey := tmtypes.NewValidator(tmPubKey, 1)

	//FIXME: temp hack for testing
	pubkey, err := getOperatorPubkey("/Users/mtsitrin/.rollapp_evm", "michael")
	if err != nil {
		return err
	}
	tmPubKey, err = cryptocodec.ToTmPubKeyInterface(pubkey)
	if err != nil {
		return err
	}
	operatorPubkey := tmtypes.NewValidator(tmPubKey, 1)

	res, err := m.executor.InitChain(m.genesis, []*tmtypes.Validator{consensusPubkey, operatorPubkey})
	if err != nil {
		return err
	}

	m.executor.UpdateStateAfterInitChain(&m.lastState, res, []*tmtypes.Validator{consensusPubkey})
	if _, err := m.store.UpdateState(m.lastState, nil); err != nil {
		return err
	}

	return nil
}

func getOperatorPubkey(keyDir, accountName string) (cryptotypes.PubKey, error) {
	// open keyring
	//load pubkey
	keyring, err := cosmosaccount.New(
		cosmosaccount.WithKeyringBackend("test"),
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
