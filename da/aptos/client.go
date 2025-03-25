package aptos

import (
	"context"
	"fmt"
	"os"

	"cosmossdk.io/math"
	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/api"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const (
	// References:
	//  https://arc.net/l/quote/kxtfqemu
	//  https://github.com/aptos-labs/aptos-core/blob/main/aptos-move/aptos-gas-schedule/src/gas_schedule/transaction.rs#L71-L75
	payloadSize      = 64 * 1024
	maxBlobSizeBytes = payloadSize
)

var _ da.DataAvailabilityLayerClient = new(DataAvailabilityLayerClient)

// DataAvailabilityLayerClient implements the Data Availability Layer Client interface for Aptos
type DataAvailabilityLayerClient struct {
	cli          aptos.AptosClient
	signer       *aptos.Account
	logger       types.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	config       Config
	pubsubServer *pubsub.Server
}

// RetrieveBatches retrieves batch data from Aptos using the transaction hash
func (c *DataAvailabilityLayerClient) RetrieveBatches(daPath string) da.ResultRetrieveBatch {
	// Parse the DA path to get the transaction hash
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
	hash := daMetaData.DAPath

	c.logger.Debug("Getting blob from Aptos DA.", "hash", hash)

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
		c.logger.Error("Retrieve batch", "hash", hash, "error", err)
	}
	return resultRetrieveBatch
}

// retrieveBatches downloads a batch from Aptos and returns the batch included
func (c *DataAvailabilityLayerClient) retrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	// Get the transaction using the hash
	resTx, err := c.cli.TransactionByHash(daMetaData.DAPath)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Failed to retrieve transaction: %v", err),
				Error:   fmt.Errorf("failed to retrieve transaction: %v", err),
			},
		}
	}

	success := resTx.Success()
	if success == nil || !*success {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Transaction failed or still pending",
				Error:   fmt.Errorf("transaction failed or still pending"),
			},
		}
	}

	tx, ok := resTx.Inner.(*api.UserTransaction)
	if resTx.Type != api.TransactionVariantUser || !ok {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Transaction type is not user_transaction: %s", resTx.Type),
				Error:   fmt.Errorf("transaction type is not user_transaction: %s", resTx.Type),
			},
		}
	}

	payload, ok := tx.Payload.Inner.(*api.TransactionPayloadEntryFunction)
	if tx.Payload.Type != api.TransactionPayloadVariantEntryFunction || !ok {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Transaction response type is not EntryFunction",
				Error:   fmt.Errorf("transaction response type is not EntryFunction"),
			},
		}
	}

	if len(payload.Arguments) != 1 {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Transaction response arguments length should always be 1",
				Error:   fmt.Errorf("transaction response arguments length should always be 1"),
			},
		}
	}

	argumentHex, ok := payload.Arguments[0].(string)
	if !ok {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Transaction response argument should be a string",
				Error:   fmt.Errorf("transaction response argument should be a string"),
			},
		}
	}

	rawArgumentBcs, err := aptos.ParseHex(argumentHex)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Failed to decode argument from hex: %v", err),
				Error:   fmt.Errorf("failed to decode argument from hex: %v", err),
			},
		}
	}

	deserializer := bcs.NewDeserializer(rawArgumentBcs)
	rawArgument := deserializer.ReadFixedBytes(len(rawArgumentBcs))
	err = deserializer.Error()
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Failed to deserialize argument: %v", err),
				Error:   fmt.Errorf("failed to deserialize argument: %v", err),
			},
		}
	}

	parsedBatch, err := c.parseBatch(rawArgument)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Failed to parse batch: %s", err),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}

	c.logger.Debug("Blob retrieved successfully from Aptos DA.", "hash", daMetaData.DAPath)

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Batch retrieval successful",
		},
		Batches: []*types.Batch{parsedBatch},
	}
}

// parseBatch parses the raw batch data into a types.Batch
func (c *DataAvailabilityLayerClient) parseBatch(batchData []byte) (*types.Batch, error) {
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

// CheckBatchAvailability checks if a batch is available on Aptos by verifying the transaction exists
func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daPath string) da.ResultCheckBatch {
	// Parse the DA path to get the transaction hash
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
	hash := daMetaData.DAPath

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
		c.logger.Error("CheckBatchAvailability", "hash", hash, "error", err)
	}
	return result
}

// checkBatchAvailability checks if a batch is available on Aptos by verifying the transaction exists
func (c *DataAvailabilityLayerClient) checkBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	// Check if the transaction exists by trying to fetch it
	resTx, err := c.cli.TransactionByHash(daMetaData.DAPath)
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Transaction not found: %v", err),
				Error:   err,
			},
		}
	}

	success := resTx.Success()
	if success == nil || !*success {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Transaction failed or still pending",
				Error:   fmt.Errorf("transaction failed or still pending"),
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

			result := c.checkBatchAvailability(daMetaData)
			if result.Error != nil {
				c.logger.Error("Check batch availability: submitted batch but did not get availability success status.", "error", result.Error)
				backoff.Sleep()
				continue
			}

			c.logger.Debug("Submitted blob to Aptos DA successfully.", "blob_size", len(data), "hash", daMetaData.DAPath)

			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "Batch submitted successfully to Aptos",
				},
				SubmitMetaData: &da.DASubmitMetaData{
					Client: da.Aptos,
					DAPath: daMetaData.DAPath,
				},
			}
		}
	}
}

// submit submits a blob to Aptos, including data bytes
func (c *DataAvailabilityLayerClient) submit(data []byte) (*da.DASubmitMetaData, error) {
	if len(data) > maxBlobSizeBytes {
		return nil, fmt.Errorf("batch do not fit into tx: %d bytes: limit: %d bytes", len(data), maxBlobSizeBytes)
	}

	// Create transaction parameters for each chunk
	rawData, err := bcs.SerializeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("marshal to BCS: %v", err)
	}

	// Submit the transaction
	tx, err := c.cli.BuildSignAndSubmitTransaction(c.signer, aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: c.signer.AccountAddress(), // Address of the signer
				Name:    "noop",
			},
			Function: "noop",
			ArgTypes: []aptos.TypeTag{},
			Args: [][]byte{
				rawData,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build transaction: %v", err)
	}

	// Wait for transaction confirmation
	userTx, err := c.cli.WaitForTransaction(tx.Hash)
	if err != nil {
		return nil, fmt.Errorf("wait for transaction: %v", err)
	}

	if !userTx.Success {
		return nil, fmt.Errorf("transaction failed: vm_status: %s", userTx.VmStatus)
	}

	return &da.DASubmitMetaData{
		Client: da.Aptos,
		DAPath: tx.Hash,
	}, nil
}

// Init initializes the Aptos DataAvailabilityLayerClient instance.
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

// Start prepares the Aptos client to work.
func (c *DataAvailabilityLayerClient) Start() error {
	c.logger.Info("Starting Aptos Data Availability Layer Client.")

	// client has already been set (likely through options)
	if c.cli != nil && c.signer != nil {
		c.logger.Info("Aptos client already set.")
		return nil
	}

	priKeyHex := os.Getenv(c.config.PriKeyEnv)
	if priKeyHex == "" {
		return fmt.Errorf("private key environment %s is not set or empty", c.config.PriKeyEnv)
	}

	priKeyHexFormated, err := crypto.FormatPrivateKey(priKeyHex, crypto.PrivateKeyVariantEd25519)
	if err != nil {
		return fmt.Errorf("format private key: %w", err)
	}

	priKey := new(crypto.Ed25519PrivateKey)
	err = priKey.FromHex(priKeyHexFormated)
	if err != nil {
		return fmt.Errorf("create aptos account: %w", err)
	}

	signer, err := aptos.NewAccountFromSigner(priKey)
	if err != nil {
		return fmt.Errorf("create aptos account: %w", err)
	}

	var cfg aptos.NetworkConfig
	switch c.config.Network {
	case "devnet":
		cfg = aptos.DevnetConfig
	case "mainnet":
		cfg = aptos.MainnetConfig
	default:
		cfg = aptos.TestnetConfig
	}

	// Default timeout is 60 seconds
	cli, err := aptos.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("create aptos client: %w", err)
	}

	c.cli = cli
	c.signer = signer

	address := c.signer.AccountAddress()
	c.logger.Info("Aptos client initialized successfully", "address", address.String())
	return nil
}

// Stop stops the Aptos Data Availability Layer Client.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.logger.Info("Stopping Aptos Data Availability Layer Client.")
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Aptos
}

func (c *DataAvailabilityLayerClient) RollappId() string {
	return ""
}

// GetSignerBalance returns the balance for the Aptos account.
func (c *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	balance, err := c.cli.AccountAPTBalance(c.signer.AccountAddress())
	if err != nil {
		return da.Balance{}, fmt.Errorf("get balance: %w", err)
	}

	return da.Balance{
		Amount: math.NewIntFromUint64(balance),
		Denom:  aptSymbol,
	}, nil
}

func (c *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return maxBlobSizeBytes
}
