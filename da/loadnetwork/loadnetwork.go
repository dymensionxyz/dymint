package loadnetwork

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
	"github.com/dymensionxyz/dymint/da/loadnetwork/gateway"
	"github.com/dymensionxyz/dymint/da/loadnetwork/rpc"
	"github.com/dymensionxyz/dymint/da/loadnetwork/signer"
	loadnetworktypes "github.com/dymensionxyz/dymint/da/loadnetwork/types"
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

type LoadNetwork interface {
	SendTransaction(ctx context.Context, to string, data []byte) (string, error)
	GetTransactionReceipt(ctx context.Context, txHash string) (*ethtypes.Receipt, error)
	GetTransactionByHash(ctx context.Context, txHash string) (*ethtypes.Transaction, bool, error)
}

type Gateway interface {
	RetrieveFromGateway(ctx context.Context, txHash string) (*loadnetworktypes.LNDymintBlob, error)
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
	client       LoadNetwork
	gateway      Gateway
	pubsubServer *pubsub.Server
	config       *loadnetworktypes.Config
	logger       types.Logger
	ctx          context.Context
	cancel       context.CancelFunc
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
func WithRPCClient(rpc LoadNetwork) da.Option {
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

// Init initializes the LoadNetwork DA client
func (c *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KV, logger types.Logger, options ...da.Option) error {
	logger.Debug("Initializing LoadNetwork DA client", "config", string(config))
	var cfg loadnetworktypes.Config
	if len(config) > 0 {
		err := json.Unmarshal(config, &cfg)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	// Set initial values
	c.pubsubServer = pubsubServer
	c.logger = logger

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
			client, err := rpc.NewLNRPCClient(logger, &cfg, web3signer)
			if err != nil {
				return fmt.Errorf("failed to initialize rpc client: %w", err)
			}
			c.client = client
		} else if cfg.PrivateKeyHex != "" {
			// Initialize with private key
			privateKeySigner := signer.NewPrivateKeySigner(cfg.PrivateKeyHex, logger, cfg.ChainID)
			client, err := rpc.NewLNRPCClient(logger, &cfg, privateKeySigner)
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
	return nil
}

// Stop stops DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.cancel()
	return nil
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.LoadNetwork
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

	if len(data) > loadnetworktypes.LoadNetworkMaxTransactionSize {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("size bigger than maximum blob size: max n bytes: %d", loadnetworktypes.LoadNetworkMaxTransactionSize),
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

			daMetaData := &SubmitMetaData{
				Height:     submitMeta.LNBlockNumber.Uint64(),
				Commitment: commitment,
				LNTxHash:   submitMeta.LNTxHash,
			}

			c.logger.Debug("Submitted blob to DA successfully.")

			metrics.RollappConsecutiveFailedDASubmission.Set(0)
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "Submission successful",
				},
				SubmitMetaData: &da.DASubmitMetaData{
					Client: da.LoadNetwork,
					DAPath: daMetaData.ToPath(),
				},
			}
		}
	}
}

func (c *DataAvailabilityLayerClient) RetrieveBatches(daPath string) da.ResultRetrieveBatch {
	submitMetadata := &SubmitMetaData{}
	daMetaData, err := submitMetadata.FromPath(daPath)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Unable to get DA metadata",
			},
		}
	}
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
					if resultRetrieveBatch.Error != nil {
						switch resultRetrieveBatch.Error {
						case da.ErrRetrieval:
							c.logger.Error("Retrieve batch failed with retrieval error. Retrying retrieve attempt.", "error", resultRetrieveBatch.Error)
							return resultRetrieveBatch.Error // Trigger retry
						case da.ErrBlobNotFound, da.ErrBlobNotIncluded, da.ErrProofNotMatching:
							return retry.Unrecoverable(resultRetrieveBatch.Error)
						default:
							return retry.Unrecoverable(resultRetrieveBatch.Error)
						}
					}
					return nil
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

func (c *DataAvailabilityLayerClient) retrieveBatches(daMetaData *SubmitMetaData) da.ResultRetrieveBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	c.logger.Debug("Getting blob from LoadNetwork DA.")

	// 1. Try LoadNetwork RPC first
	data, errRpc := c.retrieveFromLoadNetwork(ctx, daMetaData.LNTxHash)
	if errRpc == nil {
		return c.processRetrievedData(data, daMetaData)
	}
	if isRpcTransactionNotFoundErr(errRpc) {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Errorf("failed to find transaction data in loadnetwork: %w", errRpc).Error(),
				Error:   da.ErrBlobNotFound,
			},
		}
	}

	c.logger.Error("Failed to retrieve blob from loadnetwork rpc, we will try to use loadnetwork gateway",
		"loadnetwork_tx_hash", daMetaData.LNTxHash, "error", errRpc)

	// 2. Try gateway
	data, errGateway := c.gateway.RetrieveFromGateway(ctx, daMetaData.LNTxHash)
	if errGateway == nil {
		if isGatewayTransactionNotFoundErr(data) {
			return da.ResultRetrieveBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: "failed to find transaction data in loadnetwork using gateway",
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
			Message: fmt.Errorf("failed to retrieve data from loadnetwork gateway: %w", errGateway).Error(),
			Error:   da.ErrRetrieval,
		},
	}
}

func (c *DataAvailabilityLayerClient) retrieveFromLoadNetwork(ctx context.Context, txHash string) (*loadnetworktypes.LNDymintBlob, error) {
	tx, _, err := c.client.GetTransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}

	return &loadnetworktypes.LNDymintBlob{Blob: tx.Data(), LNTxHash: txHash}, nil
}

func (c *DataAvailabilityLayerClient) processRetrievedData(data *loadnetworktypes.LNDymintBlob, daMetaData *SubmitMetaData) da.ResultRetrieveBatch {
	var batches []*types.Batch
	if len(data.Blob) == 0 {
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
			"ln_block_number", daMetaData.Height,
			"ln_tx_hash", daMetaData.LNTxHash,
		)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}

	c.logger.Debug("Blob retrieved successfully from LoadNetwork DA.", "ln_tx_hash", daMetaData.LNTxHash)

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

func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daPath string) da.ResultCheckBatch {
	submitMetadata := &SubmitMetaData{}
	daMetaData, err := submitMetadata.FromPath(daPath)
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Unable to get DA metadata",
			},
		}
	}

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
						c.logger.Error("CheckBatchAvailability failed with retrieval error. Retrying availability check.", "error", result.Error)
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

func (c *DataAvailabilityLayerClient) checkBatchAvailability(daMetaData *SubmitMetaData) da.ResultCheckBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	data, errRpc := c.retrieveFromLoadNetwork(ctx, daMetaData.LNTxHash)
	if errRpc == nil {
		return c.processAvailabilityData(data, daMetaData)
	}
	if isRpcTransactionNotFoundErr(errRpc) {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Errorf("failed to find transaction data in loadnetwork: %w", errRpc).Error(),
				Error:   da.ErrBlobNotFound,
			},
		}
	}

	c.logger.Error("Failed to retrieve blob from loadnetwork rpc, we will try to use loadnetwork gateway",
		"ln_tx_hash", daMetaData.LNTxHash, "error", errRpc)

	data, errGateway := c.gateway.RetrieveFromGateway(ctx, daMetaData.LNTxHash)
	if errGateway == nil {
		if isGatewayTransactionNotFoundErr(data) {
			return da.ResultCheckBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: "failed to find transaction data in loadnetwork using gateway",
					Error:   da.ErrBlobNotFound,
				},
			}
		}
		return c.processAvailabilityData(data, daMetaData)
	}
	// if we are here it means that gateway call get some kinda of error
	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusError,
			Message: fmt.Errorf("failed to retrieve data from loadnetwork gateway: %w", errGateway).Error(),
			Error:   da.ErrRetrieval,
		},
	}
}

func (c *DataAvailabilityLayerClient) processAvailabilityData(data *loadnetworktypes.LNDymintBlob, daMetaData *SubmitMetaData) da.ResultCheckBatch {
	if len(data.Blob) == 0 {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Blob not included",
				Error:   da.ErrBlobNotIncluded,
			},
		}
	}

	if err := c.verifyBlobData(daMetaData.Commitment, data.Blob); err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrProofNotMatching,
			},
		}
	}

	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "batch available",
		},
	}
}

type LNSubmitBlobMeta struct {
	LNBlockNumber *big.Int
	LNBlockHash   string
	LNTxHash      string
}

// Submit submits the Blobs to Data Availability layer.
func (c *DataAvailabilityLayerClient) submit(daBlob da.Blob) (*LNSubmitBlobMeta, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	txHash, err := c.client.SendTransaction(ctx, loadnetworktypes.ArchivePoolAddress, daBlob)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	c.logger.Info("loadnetwork tx hash", "hash", txHash)

	receipt, err := c.waitForTxReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx receipt: %w", err)
	}

	c.logger.Info("data available in loadnetwork",
		"ln_tx", receipt.TxHash.Hex(),
		"ln_block", receipt.BlockHash.Hex(),
		"ln_block_number", receipt.BlockNumber)

	return &LNSubmitBlobMeta{LNBlockNumber: receipt.BlockNumber, LNBlockHash: receipt.BlockHash.Hex(), LNTxHash: receipt.TxHash.Hex()}, nil
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
		retry.Attempts(uint(*c.config.RetryAttempts)), //nolint:gosec // RetryAttempts should be always positive
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
func (c *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return loadnetworktypes.LoadNetworkMaxTransactionSize
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
// TODO: Gateway indicates a missing transaction by setting LNBlockHash to "0x".
// it will be fixed in the future
func isGatewayTransactionNotFoundErr(data *loadnetworktypes.LNDymintBlob) bool {
	return data.LNBlockHash == "0x"
}

func (c *DataAvailabilityLayerClient) RollappId() string {
	return fmt.Sprintf("%d", c.config.ChainID)
}
