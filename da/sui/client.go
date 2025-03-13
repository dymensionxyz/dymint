package sui

import (
	"context"
	"fmt"
	"os"

	"cosmossdk.io/math"
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/mystenbcs"
	suisigner "github.com/block-vision/sui-go-sdk/signer"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

const (
	// 128KB
	maxSuiTxSizeBytes = 128 * 1024
	// In fact, it's less, but we want to be safe. This includes the tx metadata and signatures.
	suiReservedTxBytes = 2 * 1024
	// around 126KB
	maxBlobSizeBytes = maxSuiTxSizeBytes - suiReservedTxBytes
	// 16KB minus 4 bytes for the length prefix in the BCS encoding
	inputMaxSizeBytes = 16*1024 - 4
)

var _ da.DataAvailabilityLayerClient = new(Client)
var _ da.BatchSubmitter = new(Client)
var _ da.BatchRetriever = new(Client)

type Client struct {
	cli    sui.ISuiAPI
	signer *suisigner.Signer
}

func (c *Client) RetrieveBatches(daPath string) da.ResultRetrieveBatch {
	//TODO implement me
	panic("implement me")
}

func (c *Client) CheckBatchAvailability(daPath string) da.ResultCheckBatch {
	//TODO implement me
	panic("implement me")
}

func (c *Client) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	data, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   err,
			},
		}
	}
}

func (c *Client) Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KV, logger types.Logger, options ...da.Option) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Start() error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) GetClientType() da.Client {
	//TODO implement me
	panic("implement me")
}

func (c *Client) GetMaxBlobSizeBytes() uint64 {
	return maxBlobSizeBytes
}

func (c *Client) GetSignerBalance() (da.Balance, error) {
	ctx := context.Background()
	c.cli.SuiXGetBalance(ctx, models.SuiXGetBalanceRequest{
		Owner:    c.signer.Address,
		CoinType: "", // empty string means SUI coin by default
	})
	return da.Balance{
		Amount: math.Int{},
		Denom:  "",
	}, nil
}

func (c *Client) RollappId() string {
	//TODO implement me
	panic("implement me")
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
				Signer: c.signer.Address,
				// Noop contract package ID. The code is available in `noop` folder.
				PackageObjectId: "0xeebcec2b40048c86facb2eb51e8c1c39ca0ed536b96f6f1d1fb58451f538299d",
				Module:          "noop",
				Function:        "noop",
				TypeArguments:   []interface{}{}, // no type args; the slice must be non-nil
				Arguments: []interface{}{
					rawData,
				},
				Gas:           nil,        // pick the gas object automatically
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
