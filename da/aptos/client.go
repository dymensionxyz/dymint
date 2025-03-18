package aptos

import (
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/api"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
)

type DataAvailabilityClient struct {
	cli    aptos.AptosClient
	singer *aptos.Account
}

func NewClient() (*DataAvailabilityClient, error) {
	cfg := aptos.TestnetConfig

	cli, err := aptos.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create aptos client: %w", err)
	}

	priKeyHex := "0x638802252197206baa5160bf2ac60e0b95491d2128a265e6ee51e0c1b0a59d9f"

	priKey := new(crypto.Ed25519PrivateKey)
	err = priKey.FromHex(priKeyHex)
	if err != nil {
		return nil, fmt.Errorf("create aptos account: %w", err)
	}

	singer, err := aptos.NewAccountFromSigner(priKey)
	if err != nil {
		return nil, fmt.Errorf("create aptos account: %w", err)
	}

	return &DataAvailabilityClient{
		cli:    cli,
		singer: singer,
	}, nil
}

func (c *DataAvailabilityClient) TestSendTx() error {
	data := []byte("hello world")
	raw, err := bcs.SerializeBytes(data)
	if err != nil {
		return fmt.Errorf("serialize data: %w", err)
	}
	tx, err := c.cli.BuildSignAndSubmitTransaction(c.singer, aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: c.singer.AccountAddress(), // TODO: change to dedicated module address
				Name:    "noop",
			},
			Function: "noop",
			ArgTypes: []aptos.TypeTag{},
			Args: [][]byte{
				raw,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("build transaction: %w", err)
	}
	resTx, err := c.cli.WaitForTransaction(tx.Hash)
	if err != nil {
		return fmt.Errorf("wait for transaction: %w", err)
	}

	fmt.Println(resTx)

	// our data is stored here:
	_ = resTx.Payload.Inner.(*api.TransactionPayloadEntryFunction).Arguments[0].(string)

	return nil
}
