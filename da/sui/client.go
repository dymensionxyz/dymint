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
	freeTxSpace = maxSuiTxSizeBytes - suiReservedTxBytes
	// 16KB is the default size. BCS encoding takes around 5 bytes, but we reserve 16 just to have some room.
	// Base64 encoding increases the size by 4/3, so we need to reserve some space for that.
	// Finally, max input is around 11.99KB.
	inputMaxSizeBytes = (16*1024 - 16) * 3 / 4
	// 16*8 bytes is a space reserved for BCS encoding given that the max number of inputs is 8: we might have
	// max 8 inputs of the maximum size. Base64 encoding increases the size by 4/3, so we need to reserve
	// some space for that. Finally, max blob size is around 94.7KB.
	maxBlobSizeBytes = (freeTxSpace - 16*8) * 3 / 4
)

var _ da.DataAvailabilityLayerClient = new(DataAvailabilityLayerClient)

// DataAvailabilityLayerClient implements the Data Availability Layer Client interface for Sui
type DataAvailabilityLayerClient struct {
	cli          sui.ISuiAPI
	signer       *suisigner.Signer
	logger       types.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	config       Config
	pubsubServer *pubsub.Server
}

// RetrieveBatches retrieves batch data from Sui using the transaction digest
func (c *DataAvailabilityLayerClient) RetrieveBatches(daPath string) da.ResultRetrieveBatch {
	c.logger.Debug("Getting blob from Sui DA.", "digest", daPath)

	var resultRetrieveBatch da.ResultRetrieveBatch
	err := retry.Do(
		func() error {
			resultRetrieveBatch = c.retrieveBatches(daPath)
			return resultRetrieveBatch.Error
		},
		retry.Attempts(uint(*c.config.RetryAttempts)), //nolint:gosec // RetryAttempts should be always positive
		retry.DelayType(retry.FixedDelay),
		retry.Delay(c.config.RetryDelay),
	)
	if err != nil {
		c.logger.Error("Retrieve batch", "digest", daPath, "error", err)
	}
	return resultRetrieveBatch
}

// retrieveBatches downloads a batch from Sui and returns the batch included
func (c *DataAvailabilityLayerClient) retrieveBatches(digest string) da.ResultRetrieveBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// Get the transaction using the digest
	resp, err := c.cli.SuiGetTransactionBlock(ctx, models.SuiGetTransactionBlockRequest{
		Digest: digest,
		Options: models.SuiTransactionBlockOptions{
			ShowEffects: true, // fetch effects to verify the transaction status
			ShowInput:   true, // retrieve the input to get the data chunks
		},
	})
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Failed to retrieve transaction: %v", err),
				Error:   fmt.Errorf("failed to retrieve transaction: %w", err),
			},
		}
	}

	// Status might be either "success" or "failure": https://docs.sui.io/sui-api-ref#executionstatus
	if resp.Effects.Status.Status != "success" {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Transaction failed: status: %s: error: %s", resp.Effects.Status.Status, resp.Effects.Status.Error),
				Error:   fmt.Errorf("transaction failed: status: %s: error: %s", resp.Effects.Status.Status, resp.Effects.Status.Error),
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
				Error:   fmt.Errorf("failed to collect chunks: %w", err),
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

	c.logger.Debug("Blob retrieved successfully from Sui DA.", "digest", digest)

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
func (c *DataAvailabilityLayerClient) collectChunks(inputs []models.SuiCallArg) ([]byte, error) {
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
			return nil, fmt.Errorf("invalid input value: expected base64-encoded string, got %s", input["value"])
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
func (c *DataAvailabilityLayerClient) parseBatch(batchData []byte) (*types.Batch, error) {
	var batch pb.Batch
	err := proto.Unmarshal(batchData, &batch)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling batch: %w", err)
	}

	// Convert from proto format to our batch type
	parsedBatch := new(types.Batch)
	err = parsedBatch.FromProto(&batch)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling batch: %w", err)
	}

	return parsedBatch, nil
}

// CheckBatchAvailability checks if a batch is available on Sui by verifying the transaction exists
func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daPath string) da.ResultCheckBatch {
	var result da.ResultCheckBatch
	err := retry.Do(
		func() error {
			result = c.checkBatchAvailability(daPath)
			return result.Error
		},
		retry.Attempts(uint(*c.config.RetryAttempts)), //nolint:gosec // RetryAttempts should be always positive
		retry.DelayType(retry.FixedDelay),
		retry.Delay(c.config.RetryDelay),
	)
	if err != nil {
		c.logger.Error("CheckBatchAvailability", "digest", daPath, "error", err)
	}
	return result
}

// checkBatchAvailability checks if a batch is available on Sui by verifying the transaction exists
func (c *DataAvailabilityLayerClient) checkBatchAvailability(digest string) da.ResultCheckBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// Check if the transaction exists by trying to fetch it
	resp, err := c.cli.SuiGetTransactionBlock(ctx, models.SuiGetTransactionBlockRequest{
		Digest: digest,
		Options: models.SuiTransactionBlockOptions{
			ShowEffects: true, // fetch effects to verify the transaction status
		},
	})
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Check batch availability: transaction not found: %v", err),
				Error:   fmt.Errorf("check batch availability: transaction not found: %w", err),
			},
		}
	}

	// Status might be either "success" or "failure": https://docs.sui.io/sui-api-ref#executionstatus
	if resp.Effects.Status.Status != "success" {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Check batch availability: transaction failed: %s", resp.Effects.Status.Error),
				Error:   fmt.Errorf("check batch availability: transaction failed: %s", resp.Effects.Status.Error),
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

func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
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

			result := c.checkBatchAvailability(daMetaData.DAPath)
			if result.Error != nil {
				c.logger.Error("Check batch availability: submitted batch but did not get availability success status.", "error", result.Error)
				backoff.Sleep()
				continue
			}

			c.logger.Debug("Submitted blob to Sui DA successfully.", "blob_size", len(data), "digest", daMetaData.DAPath)

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
func (c *DataAvailabilityLayerClient) submit(data []byte) (*da.DASubmitMetaData, error) {
	if len(data) > maxBlobSizeBytes {
		return nil, fmt.Errorf("batch do not fit into tx: %d bytes: limit: %d bytes", len(data), maxBlobSizeBytes)
	}

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
			return nil, fmt.Errorf("marshal to BCS: %w", err)
		}

		txs = append(txs, models.RPCTransactionRequestParams{
			MoveCallRequestParams: &models.MoveCallRequest{
				Signer:          c.signer.Address,
				PackageObjectId: c.config.NoopContractAddress,
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
		Gas:                            nil, // pick the gas object automatically
		GasBudget:                      c.config.GasBudget,
		SuiTransactionBlockBuilderMode: "Commit",
	})
	if err != nil {
		return nil, fmt.Errorf("batch transaction: %w", err)
	}

	// Transaction fails if:
	// - Insufficient balance to pay for the gas
	// - Too low gas budget
	// - One of the arguments is too big after encoding
	// - The transaction is too big
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
		TxnMetaData: models.TxnMetaData(unsignedTx),
		PriKey:      c.signer.PriKey,
		Options: models.SuiTransactionBlockOptions{
			ShowInput:    true,
			ShowRawInput: true,
		},
		RequestType: "WaitForLocalExecution",
	})
	if err != nil {
		return nil, fmt.Errorf("sign and execute transaction block: %w", err)
	}

	// "If the node fails to execute the transaction locally in a timely manner, a bool type in
	// the response is set to `false` to indicate the case."
	// Reference: https://docs.sui.io/sui-api-ref#sui_executetransactionblock
	//
	// TODO: What is "timely manner"? What to do if ConfirmLocalExecution == false?
	//  For now, just retry.
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
func (c *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
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
func (c *DataAvailabilityLayerClient) Start() error {
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
func (c *DataAvailabilityLayerClient) Stop() error {
	c.logger.Info("Stopping Sui Data Availability Layer Client.")
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Sui
}

func (c *DataAvailabilityLayerClient) RollappId() string {
	return ""
}

// GetSignerBalance returns the balance for the Sui account.
func (c *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.cli.SuiXGetBalance(ctx, models.SuiXGetBalanceRequest{
		Owner:    c.signer.Address,
		CoinType: "0x2::sui::SUI",
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
		Denom:  suiSymbol,
	}, nil
}

func (c *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return maxBlobSizeBytes
}

// TestRequestCoins requests coins from the Sui faucet. Only for testing purposes.
func (c *DataAvailabilityLayerClient) TestRequestCoins() error {
	err := sui.RequestSuiFromFaucet("https://faucet.devnet.sui.io", c.signer.Address, map[string]string{})
	if err != nil {
		return fmt.Errorf("request coins: %w", err)
	}
	return nil
}
