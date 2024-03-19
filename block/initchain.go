package block

import (
	"context"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/evmos/evmos/v12/crypto/hd"
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
	c, err := cosmosaccount.New(
		cosmosaccount.WithKeyringBackend(cosmosaccount.KeyringBackend(keyringBackend)),
		cosmosaccount.WithHome(keyDir),
	)
	if err != nil {
		return nil, err
	}

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	customKeyring, err := keyring.New("operatorAddr", keyringBackend, keyDir, os.Stdin, cdc, hd.EthSecp256k1Option())
	if err != nil {
		return nil, err
	}
	c.Keyring = customKeyring

	// Get account from the keyring
	account, err := c.GetByName(accountName)
	if err != nil {
		return nil, err
	}

	pubkey, err := account.Record.GetPubKey()
	if err != nil {
		return nil, err
	}

	return pubkey, nil
}
