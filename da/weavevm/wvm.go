package weavevm

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/weavevm/gateway"
	"github.com/dymensionxyz/dymint/da/weavevm/rpc"
	"github.com/dymensionxyz/dymint/da/weavevm/signer"
	weaveVMtypes "github.com/dymensionxyz/dymint/da/weavevm/types"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/metrics"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	uretry "github.com/dymensionxyz/dymint/utils/retry"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/ethereum/go-ethereum"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/pubsub"
)

type WeaveVM interface {
	SendTransaction(ctx context.Context, to string, data []byte) (string, error)
	GetTransactionReceipt(ctx context.Context, txHash string) (*ethtypes.Receipt, error)
	GetTransactionByHash(ctx context.Context, txHash string) (*ethtypes.Transaction, bool, error)
}

type Gateway interface {
	RetrieveFromGateway(ctx context.Context, txHash string) (*weaveVMtypes.WvmDymintBlob, error)
}

// TODO: adjust
const (
	defaultRpcRetryDelay    = 3 * time.Second
	defaultRpcTimeout       = 5 * time.Second
	defaultRpcRetryAttempts = 5
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// DataAvailabilityLayerClient use celestia-node public API.
type DataAvailabilityLayerClient struct {
	client       WeaveVM
	gateway      Gateway
	pubsubServer *pubsub.Server
	config       *weaveVMtypes.Config
	logger       types.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	synced       chan struct{}
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

// WithRPCClient sets rpc client.
func WithGatewayClient(gateway Gateway) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).gateway = gateway
	}
}

// WithRPCClient sets rpc client.
func WithRPCClient(rpc WeaveVM) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).client = rpc
	}
}

// WithRPCRetryDelay sets failed rpc calls retry delay.
func WithRPCRetryDelay(delay time.Duration) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).config.RetryDelay = delay
	}
}

// WithRPCAttempts sets failed rpc calls retry attempts.
func WithRPCAttempts(attempts int) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).config.RetryAttempts = &attempts
	}
}

// WithSubmitBackoff sets submit retry delay config.
func WithSubmitBackoff(c uretry.BackoffConfig) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).config.Backoff = c
	}
}

// Init initializes the WeaveVM DA client
func (c *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KV, logger types.Logger, options ...da.Option) error {
	logger.Debug("Initializing WeaveVM DA client", "config", string(config))
	var cfg weaveVMtypes.Config
	if len(config) > 0 {
		err := json.Unmarshal(config, &cfg)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	// Set initial values
	c.pubsubServer = pubsubServer
	c.logger = logger
	c.synced = make(chan struct{}, 1)

	// Validate and set defaults for config
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultRpcTimeout
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = defaultRpcRetryDelay
	}
	if cfg.RetryAttempts == nil {
		attempts := defaultRpcRetryAttempts
		cfg.RetryAttempts = &attempts
	}
	if cfg.Backoff == (uretry.BackoffConfig{}) {
		cfg.Backoff = defaultSubmitBackoff
	}
	if cfg.ChainID == 0 {
		return fmt.Errorf("chain ID must be set")
	}

	c.config = &cfg

	c.gateway = gateway.NewGatewayClient(c.config, logger)

	// Apply options
	for _, apply := range options {
		apply(c)
	}
	metrics.RollappConsecutiveFailedDASubmission.Set(0)

	// Initialize context
	c.ctx, c.cancel = context.WithCancel(context.Background())

	// Initialize client if not set through options
	if c.client == nil {
		if cfg.Web3SignerEndpoint != "" {
			// Initialize with web3signer
			web3signer, err := signer.NewWeb3SignerClient(&cfg, logger)
			if err != nil {
				return fmt.Errorf("failed to initialize web3signer client: %w", err)
			}
			client, err := rpc.NewWvmRPCClient(logger, &cfg, web3signer)
			if err != nil {
				return fmt.Errorf("failed to initialize rpc client: %w", err)
			}
			c.client = client
		} else if cfg.PrivateKeyHex != "" {
			// Initialize with private key
			privateKeySigner := signer.NewPrivateKeySigner(cfg.PrivateKeyHex, logger, cfg.ChainID)
			client, err := rpc.NewWvmRPCClient(logger, &cfg, privateKeySigner)
			if err != nil {
				return fmt.Errorf("failed to initialize rpc client: %w", err)
			}
			c.client = client
		} else {
			return fmt.Errorf("either web3signer endpoint or private key must be provided")
		}
	}

	return nil
}

// Start starts DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Start() error {
	c.synced <- struct{}{}
	return nil
}

// Stop stops DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.cancel()
	close(c.synced)
	return nil
}

// WaitForSyncing is used to check when the DA light client finished syncing
func (m *DataAvailabilityLayerClient) WaitForSyncing() {
	<-m.synced
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.WeaveVM
}

// SubmitBatch submits a batch to the DA layer.
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

	commitment := generateCommitment(data)

	if len(data) > weaveVMtypes.WeaveVMMaxTransactionSize {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("size bigger than maximum blob size: max n bytes: %d", weaveVMtypes.WeaveVMMaxTransactionSize),
				Error:   errors.New("blob size too big"),
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
			submitMeta, err := c.submit(data)
			if errors.Is(err, gerrc.ErrInternal) {
				err = fmt.Errorf("submit: %w", err)
				return da.ResultSubmitBatch{
					BaseResult: da.BaseResult{
						Code:    da.StatusError,
						Message: err.Error(),
						Error:   err,
					},
				}
			}

			if err != nil {
				c.logger.Error("Submit blob.", "error", err)
				metrics.RollappConsecutiveFailedDASubmission.Inc()
				backoff.Sleep()
				continue
			}

			daMetaData := &da.DASubmitMetaData{
				Client:       da.WeaveVM,
				Height:       submitMeta.WvmBlockNumber.Uint64(),
				Commitment:   commitment,
				WvmTxHash:    submitMeta.WvmTxHash,
				WvmBlockHash: submitMeta.WvmBlockHash,
			}

			c.logger.Debug("Submitted blob to DA successfully.")

			metrics.RollappConsecutiveFailedDASubmission.Set(0)
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "Submission successful",
				},
				SubmitMetaData: daMetaData,
			}
		}
	}
}

func (c *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled.")
			return da.ResultRetrieveBatch{}
		default:
			var resultRetrieveBatch da.ResultRetrieveBatch
			err := retry.Do(
				func() error {
					resultRetrieveBatch = c.retrieveBatches(daMetaData)
					switch resultRetrieveBatch.Error {
					case da.ErrRetrieval:
						c.logger.Error("Retrieve batch failed with retrieval error. Retrying availability check.", "error", resultRetrieveBatch.Error)
						return resultRetrieveBatch.Error // Trigger retry
					case da.ErrBlobNotFound, da.ErrBlobNotIncluded, da.ErrProofNotMatching:
						return retry.Unrecoverable(resultRetrieveBatch.Error)
					default:
						return retry.Unrecoverable(resultRetrieveBatch.Error)
					}
				},
				retry.Attempts(uint(*c.config.RetryAttempts)), //nolint:gosec // RetryAttempts should be always positive
				retry.DelayType(retry.FixedDelay),
				retry.Delay(c.config.RetryDelay),
			)
			if err != nil {
				c.logger.Error("RetrieveBatches process failed.", "error", err)
			}
			return resultRetrieveBatch

		}
	}
}

func (c *DataAvailabilityLayerClient) retrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	c.logger.Debug("Getting blob from weaveVM DA.")

	// 1. Try WeaveVM RPC first
	data, errRpc := c.retrieveFromWeaveVM(ctx, daMetaData.WvmTxHash)
	if errRpc == nil {
		return c.processRetrievedData(data, daMetaData)
	}
	if isRpcTransactionNotFoundErr(errRpc) {
		errRpc = fmt.Errorf("failed to retrieve data from weavevm rpc: %w", errRpc)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Errorf("failed to retrieve data from weavevm rpc: %w", errRpc).Error(),
				Error:   da.ErrBlobNotFound,
			},
		}
	}

	c.logger.Error("Failed to retrieve blob from weavevm rpc, we will try to use weavevm gateway",
		"wvm_tx_hash", daMetaData.WvmTxHash, "error", errRpc)

	// 2. Try gateway
	data, errGateway := c.gateway.RetrieveFromGateway(ctx, daMetaData.WvmTxHash)
	if errGateway == nil {
		if isGatewayTransactionNotFoundErr(data) {
			return da.ResultRetrieveBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: fmt.Sprintf("failed to find transaction data in weavevm gateway"),
					Error:   da.ErrBlobNotFound,
				},
			}
		}
		return c.processRetrievedData(data, daMetaData)
	}
	// if we are here it means that gateway call get some kinda of error
	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusError,
			Message: fmt.Errorf("failed to retrieve data from weave vm gateway: %w", errGateway).Error(),
			Error:   da.ErrRetrieval,
		},
	}
}

func (c *DataAvailabilityLayerClient) retrieveFromWeaveVM(ctx context.Context, txHash string) (*weaveVMtypes.WvmDymintBlob, error) {
	tx, _, err := c.client.GetTransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}

	return &weaveVMtypes.WvmDymintBlob{Blob: tx.Data(), WvmTxHash: txHash}, nil
}

func (c *DataAvailabilityLayerClient) processRetrievedData(data *weaveVMtypes.WvmDymintBlob, daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	var batches []*types.Batch
	if data.Blob == nil || len(data.Blob) == 0 {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Blob not included",
				Error:   da.ErrBlobNotIncluded,
			},
		}
	}

	// Verify blob data integrity
	if err := c.verifyBlobData(daMetaData.Commitment, data.Blob); err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrProofNotMatching,
			},
		}
	}
	var batch pb.Batch
	err := proto.Unmarshal(data.Blob, &batch)
	if err != nil {
		c.logger.Error("Unmarshal blob.",
			"wvm_block_number", daMetaData.Height,
			"wvm_block_hash", daMetaData.WvmBlockHash,
			"wvm_tx_hash", daMetaData.WvmTxHash,
			"arweave_block_hash", daMetaData.WvmArweaveBlockHash, "error", err)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}

	c.logger.Debug("Blob retrieved successfully from WeaveVM DA.", "wvm_tx_hash", daMetaData.WvmTxHash)

	parsedBatch := new(types.Batch)
	err = parsedBatch.FromProto(&batch)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}
	batches = append(batches, parsedBatch)
	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Batch retrieval successful",
		},
		Batches: batches,
	}
}

func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	var availabilityResult da.ResultCheckBatch
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled")
			return da.ResultCheckBatch{}
		default:
			err := retry.Do(
				func() error {
					result := c.checkBatchAvailability(daMetaData)
					availabilityResult = result
					switch result.Error {
					case da.ErrRetrieval:
						c.logger.Error("Retrieve batch failed with retrieval error. Retrying availability check.", "error", result.Error)
						return result.Error // Trigger retry
					case da.ErrBlobNotFound, da.ErrBlobNotIncluded, da.ErrProofNotMatching:
						return retry.Unrecoverable(result.Error)
					default:
						return retry.Unrecoverable(result.Error)
					}
				},
				retry.Attempts(uint(*c.config.RetryAttempts)), //nolint:gosec // RetryAttempts should be always positive
				retry.DelayType(retry.FixedDelay),
				retry.Delay(c.config.RetryDelay),
			)
			if err != nil {
				c.logger.Error("CheckAvailability process failed.", "error", err)
			}
			return availabilityResult
		}
	}
}

func (c *DataAvailabilityLayerClient) checkBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	DACheckMetaData := &da.DACheckMetaData{
		Client:       daMetaData.Client,
		Height:       daMetaData.Height,
		WvmTxHash:    daMetaData.WvmTxHash,
		WvmBlockHash: daMetaData.WvmBlockHash,
		Commitment:   daMetaData.Commitment,
	}

	data, errRpc := c.retrieveFromWeaveVM(ctx, daMetaData.WvmTxHash)
	if errRpc == nil {
		return c.processAvailabilityData(data, DACheckMetaData)
	}
	if isRpcTransactionNotFoundErr(errRpc) {
		errRpc = fmt.Errorf("failed to retrieve data from weavevm rpc: %w", errRpc)
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Errorf("failed to retrieve data from weavevm rpc: %w", errRpc).Error(),
				Error:   da.ErrBlobNotFound,
			},
			CheckMetaData: DACheckMetaData,
		}
	}

	c.logger.Error("Failed to retrieve blob from weavevm rpc, we will try to use weavevm gateway",
		"wvm_tx_hash", daMetaData.WvmTxHash, "error", errRpc)

	data, errGateway := c.gateway.RetrieveFromGateway(ctx, daMetaData.WvmTxHash)
	if errGateway == nil {
		if isGatewayTransactionNotFoundErr(data) {
			return da.ResultCheckBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: fmt.Sprintf("failed to find transaction data in weavevm gateway"),
					Error:   da.ErrBlobNotFound,
				},
				CheckMetaData: DACheckMetaData,
			}
		}
		return c.processAvailabilityData(data, DACheckMetaData)
	}
	// if we are here it means that gateway call get some kinda of error
	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusError,
			Message: fmt.Errorf("failed to retrieve data from weave vm gateway: %w", errGateway).Error(),
			Error:   da.ErrRetrieval,
		},
		CheckMetaData: DACheckMetaData,
	}
}

func (c *DataAvailabilityLayerClient) processAvailabilityData(data *weaveVMtypes.WvmDymintBlob, daMetaData *da.DACheckMetaData) da.ResultCheckBatch {
	if data.Blob == nil || len(data.Blob) == 0 {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Blob not included",
				Error:   da.ErrBlobNotIncluded,
			},
			CheckMetaData: daMetaData,
		}
	}

	if err := c.verifyBlobData(daMetaData.Commitment, data.Blob); err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrProofNotMatching,
			},
			CheckMetaData: daMetaData,
		}
	}

	// If ArweaveBlockHash is missing in metadata but available in the blob, update it.
	if daMetaData.WvmArweaveBlockHash == "" && data.ArweaveBlockHash != "" {
		daMetaData.WvmArweaveBlockHash = data.ArweaveBlockHash
	}

	if daMetaData.Height < data.WvmBlockNumber {
		// Update metadata only if the blob represents a higher block (reorg case)
		daMetaData.WvmArweaveBlockHash = data.ArweaveBlockHash
		daMetaData.WvmBlockHash = data.WvmBlockHash
		daMetaData.Height = data.WvmBlockNumber
	}

	// Ensure WvmBlockHash matches the latest blob hash for consistency
	if daMetaData.WvmBlockHash != data.WvmBlockHash {
		daMetaData.WvmBlockHash = data.WvmBlockHash
	}

	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "batch available",
		},
		CheckMetaData: daMetaData,
	}
}

type WvmSubmitBlobMeta struct {
	WvmBlockNumber      *big.Int
	WvmBlockHash        string
	WvmTxHash           string
	WvmArweaveBlockHash string
}

// Submit submits the Blobs to Data Availability layer.
func (c *DataAvailabilityLayerClient) submit(daBlob da.Blob) (*WvmSubmitBlobMeta, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	txHash, err := c.client.SendTransaction(ctx, weaveVMtypes.ArchivePoolAddress, daBlob)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	c.logger.Info("wvm tx hash", "hash", txHash)

	receipt, err := c.waitForTxReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx receipt: %w", err)
	}

	c.logger.Info("data available in weavevm",
		"wvm_tx", receipt.TxHash.Hex(),
		"wvm_block", receipt.BlockHash.Hex(),
		"wvm_block_number", receipt.BlockNumber)

	return &WvmSubmitBlobMeta{WvmBlockNumber: receipt.BlockNumber, WvmBlockHash: receipt.BlockHash.Hex(), WvmTxHash: receipt.TxHash.Hex()}, nil
}

func (c *DataAvailabilityLayerClient) waitForTxReceipt(ctx context.Context, txHash string) (*ethtypes.Receipt, error) {
	var receipt *ethtypes.Receipt
	err := retry.Do(
		func() error {
			var err error
			receipt, err = c.client.GetTransactionReceipt(ctx, txHash)
			if err != nil {
				// Mark network/temporary errors as retryable
				return fmt.Errorf("get receipt failed: %w", err)
			}
			if receipt == nil {
				// Receipt not found yet - this is retryable
				return fmt.Errorf("receipt not found")
			}
			if receipt.BlockNumber == nil || receipt.BlockNumber.Cmp(big.NewInt(0)) == 0 {
				return fmt.Errorf("no block number in receipt")
			}
			return nil
		},
		retry.Context(ctx),
		retry.Attempts(uint(*c.config.RetryAttempts)),
		retry.Delay(c.config.RetryDelay),
		retry.DelayType(retry.FixedDelay), // Force fixed delay between attempts
		retry.LastErrorOnly(true),         // Only log the last error
		retry.OnRetry(func(n uint, err error) {
			c.logger.Debug("waiting for receipt",
				"txHash", txHash,
				"attempt", n,
				"error", err)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt after %d attempts: %w",
			*c.config.RetryAttempts, err)
	}

	return receipt, nil
}

// GetMaxBlobSizeBytes returns the maximum allowed blob size in the DA, used to check the max batch size configured
func (c *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint32 {
	return weaveVMtypes.WeaveVMMaxTransactionSize
}

// GetSignerBalance returns the balance for a specific address
func (c *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	return da.Balance{}, nil
}

func generateCommitment(data []byte) []byte {
	return crypto.Keccak256(data)
}

func (c *DataAvailabilityLayerClient) verifyBlobData(commitment []byte, data []byte) error {
	h := crypto.Keccak256Hash(data)
	if !bytes.Equal(h[:], commitment) {
		c.logger.Debug("commitment verification failed",
			"expected", hex.EncodeToString(commitment),
			"got", h.Hex())
		return fmt.Errorf("commitment does not match data, expected: %s got: %s",
			hex.EncodeToString(commitment), h.Hex())
	}
	return nil
}

func isRpcTransactionNotFoundErr(err error) bool {
	return errors.Is(err, ethereum.NotFound)
}

// isGatewayTransactionNotFoundErr checks if the transaction is absent in the Gateway.
// Gateway indicates a missing transaction by setting WvmBlockHash to "0x".
func isGatewayTransactionNotFoundErr(data *weaveVMtypes.WvmDymintBlob) bool {
	return data.WvmBlockHash == "0x"
}

func (c *DataAvailabilityLayerClient) DAPath() string {
	return fmt.Sprintf("%d", c.config.ChainID)
}
