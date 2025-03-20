package aptos

import (
	"encoding/hex"
	"fmt"
	"strings"

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

	priKeyHexFormated, err := crypto.FormatPrivateKey(priKeyHex, crypto.PrivateKeyVariantEd25519)
	if err != nil {
		return nil, fmt.Errorf("format private key: %w", err)
	}

	priKey := new(crypto.Ed25519PrivateKey)
	err = priKey.FromHex(priKeyHexFormated)
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

func (c *DataAvailabilityClient) TestSendTx(data []byte) (string, error) {
	raw, err := bcs.SerializeBytes(data)
	if err != nil {
		return "", fmt.Errorf("serialize data: %w", err)
	}

	//hexStr := hex.EncodeToString(raw)
	hexData := make([]byte, hex.EncodedLen(len(raw)))
	_ = hex.Encode(hexData, raw)
	fmt.Println("Data Bytes        : ", data)
	fmt.Println("BCS  Bytes        : ", raw)
	fmt.Println("Data Str          : ", string(data))
	fmt.Println("BCS  Str          : ", string(raw))
	fmt.Println("Encoded Hex Bytes : ", hexData)
	fmt.Println("Encoded Hex Str   : ", string(hexData))
	//fmt.Println(hex.DecodeString(hexStr))
	//fmt.Println(hex.DecodeString(string(raw)))

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
		return "", fmt.Errorf("build transaction: %w", err)
	}
	userTx, err := c.cli.WaitForTransaction(tx.Hash)
	if err != nil {
		return "", fmt.Errorf("wait for transaction: %w", err)
	}

	fmt.Println(userTx)

	return tx.Hash, nil
}

func (c *DataAvailabilityClient) TestCheckBatchAvailable(txHash string) error {
	resTx, err := c.cli.TransactionByHash(txHash)
	if err != nil {
		return fmt.Errorf("get transaction: %w", err)
	}

	success := resTx.Success()
	if success == nil || *success == false {
		return fmt.Errorf("transaction failed or still pending")
	}

	if resTx.Type != api.TransactionVariantUser {
		return fmt.Errorf("transaction type is not user_transaction: %s", resTx.Type)
	}

	return nil
}

func (c *DataAvailabilityClient) TestRetrieveBatch(txHash string) ([]byte, error) {
	resTx, err := c.cli.TransactionByHash(txHash)
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}

	success := resTx.Success()
	if success == nil || *success == false {
		return nil, fmt.Errorf("transaction failed or still pending")
	}

	tx, ok := resTx.Inner.(*api.UserTransaction)
	if resTx.Type != api.TransactionVariantUser || !ok {
		return nil, fmt.Errorf("transaction type is not user_transaction: %s", resTx.Type)
	}

	payload, ok := tx.Payload.Inner.(*api.TransactionPayloadEntryFunction)
	if tx.Payload.Type != api.TransactionPayloadVariantEntryFunction || !ok {
		return nil, fmt.Errorf("transaction response type is not EntryFunction")
	}

	if len(payload.Arguments) != 1 {
		return nil, fmt.Errorf("transaction response arguments length should always be 1")
	}

	argumentHex := payload.Arguments[0].(string)

	rawArgument, err := hex.DecodeString(strings.TrimPrefix(argumentHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode argument hex: %w", err)
	}

	deserializer := bcs.NewDeserializer(rawArgument)
	b := deserializer.ReadFixedBytes(len(rawArgument))
	err = deserializer.Error()
	if err != nil {
		return nil, fmt.Errorf("deserialize argument: %w", err)
	}

	return b, nil
}
