package sui

import (
	"context"
	"fmt"
	"os"

	"cosmossdk.io/math"
	"github.com/avast/retry-go/v4"
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/mystenbcs"
	suisigner "github.com/block-vision/sui-go-sdk/signer"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/pubsub"
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
	cli          sui.ISuiAPI
	signer       *suisigner.Signer
	logger       types.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	config       Config
	pubsubServer *pubsub.Server
}

// RetrieveBatches retrieves batch data from Sui using the transaction digest
func (c *Client) RetrieveBatches(daPath string) da.ResultRetrieveBatch {
	// Parse the DA path to get the transaction digest
	submitMetaData := &da.DASubmitMetaData{}
	daMetaData, err := submitMetaData.FromPath(daPath)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Unable to parse DA path",
				Error:   err,
			},
		}
	}
	digest := daMetaData.DAPath

	c.logger.Debug("Getting blob from Sui DA.", "digest", digest)

	var resultRetrieveBatch da.ResultRetrieveBatch
	err = retry.Do(
		func() error {
			resultRetrieveBatch = c.retrieveBatches(daMetaData)
			return resultRetrieveBatch.Error
		},
		retry.Attempts(uint(*c.config.RetryAttempts)),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(c.config.RetryDelay),
	)
	if err != nil {
		c.logger.Error("Retrieve batch", "digest", digest, "error", err)
	}
	return resultRetrieveBatch
}

// retrieveBatches downloads a batch from Sui and returns the batch included
func (c *Client) retrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// Get the transaction using the digest
	resp, err := c.cli.SuiGetTransactionBlock(ctx, models.SuiGetTransactionBlockRequest{
		Digest: daMetaData.DAPath,
		Options: models.SuiTransactionBlockOptions{
			ShowInput: true, // we need to retrieve the input to get the data chunks
		},
	})
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Failed to retrieve transaction: %v", err),
				Error:   fmt.Errorf("failed to retrieve transaction: %v", err),
			},
		}
	}

	if !resp.ConfirmedLocalExecution {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Transaction not confirmed locally",
				Error:   fmt.Errorf("transaction not confirmed locally"),
			},
		}
	}

	// Status might be either "success" or "failure": https://docs.sui.io/sui-api-ref#executionstatus
	if resp.Effects.Status.Status != "success" {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Transaction failed: %s", resp.Effects.Status.Error),
				Error:   fmt.Errorf("transaction failed: %s", resp.Effects.Status.Error),
			},
		}
	}

	// Transaction must be a ProgrammableTransaction in order to have inputs:
	// https://docs.sui.io/sui-api-ref#transactionblockkind
	if resp.Transaction.Data.Transaction.Kind != "ProgrammableTransaction" {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Transaction kind is not ProgrammableTransaction: %s", resp.Transaction.Data.Transaction.Kind),
				Error:   fmt.Errorf("transaction kind is not ProgrammableTransaction: %s", resp.Transaction.Data.Transaction.Kind),
			},
		}
	}

	batchData, err := c.collectChunks(resp.Transaction.Data.Transaction.Inputs)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Failed to collect chunks: %s", err),
				Error:   fmt.Errorf("failed to collect chunks: %s", err),
			},
		}
	}

	if len(batchData) == 0 {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "No batch data found in transaction",
				Error:   da.ErrBlobNotFound,
			},
		}
	}

	parsedBatch, err := c.parseBatch(batchData)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Failed to parse batch: %s", err),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}

	c.logger.Debug("Blob retrieved successfully from Sui DA.", "digest", daMetaData.DAPath)

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Batch retrieval successful",
		},
		Batches: []*types.Batch{parsedBatch},
	}
}

// collectChunks collects all the data chunks from the transaction inputs and concatenates them into a single batch.
// Each input is expected to be a Base64-BCS-encoded vector<u8> containing a chunk of the batch data.
func (c *Client) collectChunks(inputs []models.SuiCallArg) ([]byte, error) {
	var batchData []byte
	for _, input := range inputs {
		if input["type"] != "pure" {
			return nil, fmt.Errorf("invalid input type: expected pure, got %s", input["type"])
		}
		if input["valueType"] != "vector<u8>" {
			return nil, fmt.Errorf("invalid input value type: expected vector<u8>, got %s", input["valueType"])
		}

		rawValueBase64, ok := input["value"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid input value: expected string, got %T", input["value"])
		}

		// Decode from base64
		rawValue, err := mystenbcs.FromBase64(rawValueBase64)
		if err != nil {
			return nil, fmt.Errorf("invalid input value: expected base64-encoded sting, got %s", input["value"])
		}

		// Unmarshal the BCS-encoded bytes to get the actual chunk data
		var chunk []byte
		_, err = mystenbcs.Unmarshal(rawValue, &chunk)
		if err != nil {
			return nil, fmt.Errorf("invalid input value: expected Base64-BCS-encoded chunk, got %s", input["value"])
		}

		// Add the chunk to our batch data
		batchData = append(batchData, chunk...)
	}
	return batchData, nil
}

// parseBatch parses the raw batch data into a types.Batch
func (c *Client) parseBatch(batchData []byte) (*types.Batch, error) {
	var batch pb.Batch
	err := proto.Unmarshal(batchData, &batch)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling batch: %s", err)
	}

	// Convert from proto format to our batch type
	parsedBatch := new(types.Batch)
	err = parsedBatch.FromProto(&batch)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling batch: %s", err)
	}

	return parsedBatch, nil
}

// CheckBatchAvailability checks if a batch is available on Sui by verifying the transaction exists
func (c *Client) CheckBatchAvailability(daPath string) da.ResultCheckBatch {
	// Parse the DA path to get the transaction digest
	submitMetaData := &da.DASubmitMetaData{}
	daMetaData, err := submitMetaData.FromPath(daPath)
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Unable to parse DA path",
				Error:   err,
			},
		}
	}
	digest := daMetaData.DAPath

	var result da.ResultCheckBatch
	err = retry.Do(
		func() error {
			result = c.checkBatchAvailability(daMetaData)
			return result.Error
		},
		retry.Attempts(uint(*c.config.RetryAttempts)),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(c.config.RetryDelay),
	)
	if err != nil {
		c.logger.Error("CheckBatchAvailability", "digest", digest, "error", err)
	}
	return result
}

// checkBatchAvailability checks if a batch is available on Sui by verifying the transaction exists
func (c *Client) checkBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// Check if the transaction exists by trying to fetch it
	resp, err := c.cli.SuiGetTransactionBlock(ctx, models.SuiGetTransactionBlockRequest{
		Digest: daMetaData.DAPath,
		Options: models.SuiTransactionBlockOptions{
			ShowEffects: true, // fetch effects to verify the transaction status
		},
	})
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Transaction not found: %v", err),
				Error:   err,
			},
		}
	}

	if !resp.ConfirmedLocalExecution {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Transaction not confirmed locally",
				Error:   fmt.Errorf("transaction not confirmed locally"),
			},
		}
	}

	// Status might be either "success" or "failure": https://docs.sui.io/sui-api-ref#executionstatus
	if resp.Effects.Status.Status != "success" {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Transaction failed: %s", resp.Effects.Status.Error),
				Error:   fmt.Errorf("transaction failed: %s", resp.Effects.Status.Error),
			},
		}
	}

	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Batch is available",
		},
	}
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

	backoff := c.config.Backoff.Backoff()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled.")
			return da.ResultSubmitBatch{}
		default:
			daMetaData, err := c.submit(data)
			if err != nil {
				c.logger.Error("Submit blob.", "error", err)
				backoff.Sleep()
				continue
			}

			c.logger.Debug("Submitted blob to DA successfully.")

			result := c.checkBatchAvailability(daMetaData)
			if result.Error != nil {
				c.logger.Error("Check batch availability: submitted batch but did not get availability success status.", "error", result.Error)
				backoff.Sleep()
				continue
			}

			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "Batch submitted successfully to SUI",
				},
				SubmitMetaData: &da.DASubmitMetaData{
					Client: da.Sui,
					DAPath: daMetaData.DAPath,
				},
			}
		}
	}
}

// submit submits a blob to Sui, including data bytes
func (c *Client) submit(data []byte) (*da.DASubmitMetaData, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// Split the data into chunks that fit within SUI's input size limit
	var chunks [][]byte
	for i := 0; i < len(data); i += inputMaxSizeBytes {
		end := i + inputMaxSizeBytes
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}

	// Create transaction parameters for each chunk
	var txs []models.RPCTransactionRequestParams
	for _, chunk := range chunks {
		rawData, err := mystenbcs.Marshal(chunk)
		if err != nil {
			return nil, fmt.Errorf("marshal to BCS: %v", err)
		}

		txs = append(txs, models.RPCTransactionRequestParams{
			MoveCallRequestParams: &models.MoveCallRequest{
				Signer:          c.signer.Address,
				PackageObjectId: "0xeebcec2b40048c86facb2eb51e8c1c39ca0ed536b96f6f1d1fb58451f538299d", // Noop contract package ID
				Module:          "noop",
				Function:        "noop",
				TypeArguments:   []interface{}{}, // no type args; the slice must be non-nil
				Arguments: []interface{}{
					rawData,
				},
				Gas:           nil, // pick the gas object automatically
				GasBudget:     "",  // budget is set for the entire batch
				ExecutionMode: models.TransactionExecutionCommit,
			},
		})
	}

	unsignedTx, err := c.cli.BatchTransaction(ctx, models.BatchTransactionRequest{
		Signer:                         c.signer.Address,
		RPCTransactionRequestParams:    txs,
		Gas:                            nil,        // pick the gas object automatically
		GasBudget:                      "10000000", // 0.01 SUI TODO: think how to set this value dynamically; config?
		SuiTransactionBlockBuilderMode: "Commit",
	})
	if err != nil {
		return nil, fmt.Errorf("batch transaction: %v", err)
	}

	// TODO: What will happen if the transaction fails?
	// - insufficient balance to pay for the gas
	// - to low gas budget
	// - too big transaction / too big inputs
	//
	// SignAndExecuteTransactionBlock uses WaitForLocalExecution request type, which ensures
	// that the transaction is executed on the local node and is accessible for querying.
	// At this point, it reaches Settlement Finality:
	// https://docs.sui.io/concepts/sui-architecture/transaction-lifecycle#settlement-finality
	//
	// The noop smart contract is as follows:
	//
	//   module noop::noop {
	//      public fun noop(_: vector<u8>) {
	//          // Do nothing
	//      }
	//   }
	//
	// It does not have any shared objects, that require consensus validation, thus the transaction
	// can achieve fast finality in under half a second.
	//
	// The call is blocking.
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
		return nil, fmt.Errorf("sign and execute transaction block: %v", err)
	}

	// "If the node fails to execute the transaction locally in a timely manner, a bool type in
	// the response is set to `false` to indicate the case."
	// Reference: https://docs.sui.io/sui-api-ref#sui_executetransactionblock
	//
	// TODO: What is "timely manner"? What to do if ConfirmLocalExecution == false?
	// Wait for a while (until the checkpoint end?) and retry? Or retry immediately? -> Increased gas costs
	if !res.ConfirmedLocalExecution {
		return nil, fmt.Errorf("transaction not confirmed locally")
	}

	// Extract digest from the transaction. Digest is a DA path.
	digest := res.Digest
	if digest == "" {
		return nil, fmt.Errorf("empty transaction digest")
	}

	return &da.DASubmitMetaData{
		Client: da.Sui,
		DAPath: digest,
	}, nil
}

// Init initializes the Sui DataAvailabilityLayerClient instance.
func (c *Client) Init(config []byte, pubsubServer *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
	c.logger = logger
	var err error
	c.config, err = createConfig(config)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.pubsubServer = pubsubServer

	// Apply options
	for _, apply := range options {
		apply(c)
	}

	return nil
}

// Start prepares the Sui client to work.
func (c *Client) Start() error {
	c.logger.Info("Starting Sui Data Availability Layer Client.")

	// client has already been set (likely through options)
	if c.cli != nil && c.signer != nil {
		c.logger.Info("Sui client already set.")
		return nil
	}

	mnemonic := os.Getenv(c.config.MnemonicEnv)
	if mnemonic == "" {
		return fmt.Errorf("mnemonic environment variable %s is not set or empty", c.config.MnemonicEnv)
	}

	signer, err := suisigner.NewSignertWithMnemonic(mnemonic)
	if err != nil {
		return fmt.Errorf("create signer from mnemonic: %w", err)
	}

	c.cli = sui.NewSuiClient(c.config.RPCURL)
	c.signer = signer

	c.logger.Info("Sui client initialized successfully", "address", c.signer.Address)
	return nil
}

// Stop stops the Sui Data Availability Layer Client.
func (c *Client) Stop() error {
	c.logger.Info("Stopping Sui Data Availability Layer Client.")
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

// GetClientType returns client type.
func (c *Client) GetClientType() da.Client {
	return da.Sui
}

func (c *Client) RollappId() string {
	return fmt.Sprintf("%d", c.config.ChainID)
}

// GetSignerBalance returns the balance for the Sui account.
func (c *Client) GetSignerBalance() (da.Balance, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.cli.SuiXGetBalance(ctx, models.SuiXGetBalanceRequest{
		Owner:    c.signer.Address,
		CoinType: "", // empty string means SUI coin by default
	})
	if err != nil {
		return da.Balance{}, fmt.Errorf("get balance: %w", err)
	}

	amount, ok := math.NewIntFromString(resp.TotalBalance)
	if !ok {
		return da.Balance{}, fmt.Errorf("parse total balance: %s", resp.TotalBalance)
	}

	return da.Balance{
		Amount: amount,
		Denom:  "SUI",
	}, nil
}

func (c *Client) GetMaxBlobSizeBytes() uint64 {
	return maxBlobSizeBytes
}

// TODO: remove; this is for testing purposes only
func (c *Client) TestGetTransaction(ctx context.Context, digest string) error {
	resp, err := c.cli.SuiGetTransactionBlock(ctx, models.SuiGetTransactionBlockRequest{
		Digest: digest,
		Options: models.SuiTransactionBlockOptions{
			ShowInput:    true,
			ShowRawInput: true,
		},
	})
	if err != nil {
		return fmt.Errorf("get transaction block: %w", err)
	}
	fmt.Println(resp.Transaction.Data.Transaction.Kind) // Kind should be programmable
	for _, input := range resp.Transaction.Data.Transaction.Inputs {
		fmt.Println(input)
	}
	return nil
}

// TODO: remove; this is for testing purposes only
func (c *Client) TestMoveCall(ctx context.Context) (string, error) {
	var txs []models.RPCTransactionRequestParams
	for i := range 3 {
		value := fmt.Sprintf("Hello World %d", i)
		rawData, err := mystenbcs.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("marshal to BCS: %w", err)
		}

		txs = append(txs, models.RPCTransactionRequestParams{
			MoveCallRequestParams: &models.MoveCallRequest{
				Signer: c.signer.Address,
				// Noop contract package ID. The code is available in `noop` folder.
				PackageObjectId: "0xfc77c7cf837bdad12f49a2d6a8b19d77c5307730bda1fcd6cafebd7df9142b4f",
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
		return "", fmt.Errorf("batch transaction: %w", err)
	}

	// SignAndExecuteTransactionBlock uses WaitForLocalExecution request type, which ensures
	// that the transaction is executed on the local node and is accessible for querying.
	// At this point, it reaches Settlement Finality:
	// https://docs.sui.io/concepts/sui-architecture/transaction-lifecycle#settlement-finality
	//
	// The noop smart contract is as follows:
	//
	//   module noop::noop {
	//      public fun noop(_: vector<u8>) {
	//          // Do nothing
	//      }
	//   }
	//
	// It does not have any shared objects, that require consensus validation, thus the transaction
	// can achieve fast finality in under half a second.
	//
	// The call is blocking.
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
		RequestType: "WaitForLocalExecution",
	})
	if err != nil {
		return "", fmt.Errorf("sign and execute transaction block: %w", err)
	}
	// > If the node fails to execute the transaction locally in a timely manner, a bool type in
	// the response is set to `false` to indicate the case.
	// Reference: https://docs.sui.io/sui-api-ref#sui_executetransactionblock
	//
	// What is "timely manner"? What to do if ConfirmLocalExecution == false?
	// Wait for a while (until the checkpoint end?) and retry? Or retry immediately? -> Increased gas costs
	if !res.ConfirmedLocalExecution {

	}

	fmt.Printf("Transaction block executed: %v\n", res)

	return res.Digest, nil
}
