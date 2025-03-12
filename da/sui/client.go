package sui

import (
	"context"
	"fmt"
	"os"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/mystenbcs"
	suisigner "github.com/block-vision/sui-go-sdk/signer"
	"github.com/block-vision/sui-go-sdk/sui"
)

type Client struct {
	cli    sui.ISuiAPI
	signer *suisigner.Signer
}

func NewClient(rpcUrl string, mnemonicEnvVar string) (*Client, error) {
	// configurable variables:
	// - RPC address
	// - Mnemonic environment variable

	mnemonic := os.Getenv(mnemonicEnvVar)
	if mnemonic == "" {
		return nil, fmt.Errorf("mnemonic environment variable is not set or empty")
	}

	signer, err := suisigner.NewSignertWithMnemonic(mnemonic)
	if err != nil {
		return nil, fmt.Errorf("create signer from mnemonic: %w", err)
	}

	return &Client{
		cli:    sui.NewSuiClient(rpcUrl),
		signer: signer,
	}, nil
}

func (c *Client) TestMoveCall(ctx context.Context) error {
	var txs []models.RPCTransactionRequestParams
	for i := range 3 {
		value := fmt.Sprintf("Hello World %d", i)
		rawData, err := mystenbcs.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal to BCS: %w", err)
		}

		txs = append(txs, models.RPCTransactionRequestParams{
			MoveCallRequestParams: &models.MoveCallRequest{
				Signer:          c.signer.Address,
				PackageObjectId: "0xeebcec2b40048c86facb2eb51e8c1c39ca0ed536b96f6f1d1fb58451f538299d", // standard address for the Move standard library
				Module:          "noop",
				Function:        "noop",
				TypeArguments:   []interface{}{}, // no type args; the slice must be non-nil
				Arguments: []interface{}{
					rawData,
				},
				GasBudget:     "10000000", // 0.01 SUI
				ExecutionMode: models.TransactionExecutionCommit,
			},
		})
	}

	unsignedTx, err := c.cli.BatchTransaction(ctx, models.BatchTransactionRequest{
		Signer:                         c.signer.Address,
		RPCTransactionRequestParams:    txs,
		Gas:                            nil,        // pick the gas object automatically
		GasBudget:                      "10000000", // 0.01 SUI
		SuiTransactionBlockBuilderMode: "Commit",
	})
	if err != nil {
		return fmt.Errorf("batch transaction: %w", err)
	}

	res, err := c.cli.SignAndExecuteTransactionBlock(ctx, models.SignAndExecuteTransactionBlockRequest{
		TxnMetaData: models.TxnMetaData{
			Gas:          unsignedTx.Gas,
			InputObjects: unsignedTx.InputObjects,
			TxBytes:      unsignedTx.TxBytes,
		},
		PriKey: c.signer.PriKey,
		Options: models.SuiTransactionBlockOptions{
			ShowInput:    true,
			ShowRawInput: true,
		},
		RequestType: "WaitForEffectsCert",
	})
	if err != nil {
		return fmt.Errorf("sign and execute transaction block: %w", err)
	}

	fmt.Printf("Transaction block executed: %v\n", res)

	return nil
}
