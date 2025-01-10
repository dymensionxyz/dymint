package weave_vm

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/weave_vm/rpc"
	"github.com/dymensionxyz/dymint/da/weave_vm/signer"
	weaveVMtypes "github.com/dymensionxyz/dymint/da/weave_vm/types"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	uretry "github.com/dymensionxyz/dymint/utils/retry"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

	// Apply options
	for _, apply := range options {
		apply(c)
	}
	types.RollappConsecutiveFailedDASubmission.Set(0)

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
				types.RollappConsecutiveFailedDASubmission.Inc()
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

			types.RollappConsecutiveFailedDASubmission.Set(0)
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

					if errors.Is(resultRetrieveBatch.Error, da.ErrRetrieval) {
						c.logger.Error("Retrieve batch.", "error", resultRetrieveBatch.Error)
						return resultRetrieveBatch.Error
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

func (c *DataAvailabilityLayerClient) retrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	c.logger.Debug("Getting blob from weaveVM DA.")

	// 1. Try WeaveVM RPC first
	data, errRpc := c.retrieveFromWeaveVM(ctx, daMetaData.WvmTxHash)
	if errRpc != nil {
		c.logger.Error("Failed to retrieve blob from weavevm rpc, we will try to use weavevm gateway",
			"wvm_tx_hash", daMetaData.WvmTxHash, "error", errRpc)
		errRpc = fmt.Errorf("unable to retrieve data from weavevm chain rpc: %w", errRpc)
	}
	if errRpc == nil {
		return c.processRetrievedData(data, daMetaData)
	}

	// 2. Try gateway
	data, errGateway := c.retrieveFromGateway(ctx, daMetaData.WvmTxHash)
	if errGateway == nil {
		return c.processRetrievedData(data, daMetaData)
	}

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusError,
			Message: fmt.Sprintf("failed to retrieve blob from both endpoints: %s :%s", errGateway.Error(), errRpc.Error()),
			Error:   da.ErrRetrieval,
		},
	}
}

func (c *DataAvailabilityLayerClient) retrieveFromWeaveVM(ctx context.Context, txHash string) (*WvmDymintBlob, error) {
	tx, _, err := c.client.GetTransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}

	return &WvmDymintBlob{Blob: tx.Data(), WvmTxHash: txHash}, nil
}

func (c *DataAvailabilityLayerClient) processRetrievedData(data *WvmDymintBlob, daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	var batches []*types.Batch
	if data.Blob == nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Blob not found",
				Error:   da.ErrBlobNotFound,
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

const weaveVMGatewayURL = "https://gateway.wvm.dev/v1/calldata/%s"

type WvmDymintBlob struct {
	ArweaveBlockHash string
	WvmBlockHash     string
	WvmTxHash        string
	WvmBlockNumber   uint64
	Blob             []byte
}

// Modified get function with improved error handling
func (c *DataAvailabilityLayerClient) retrieveFromGateway(ctx context.Context, txHash string) (*WvmDymintBlob, error) {
	// TODO: add block number in response
	type WvmRetrieverResponse struct {
		ArweaveBlockHash   string `json:"arweave_block_hash"`
		Calldata           string `json:"calldata"`
		WarDecodedCalldata string `json:"war_decoded_calldata"`
		WvmBlockHash       string `json:"weavevm_block_hash"`
		WvmBlockNumber     uint64 `json:"wvm_block_id"`
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf(weaveVMGatewayURL,
			txHash), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	r.Header.Set("Accept", "application/json")
	client := &http.Client{
		Timeout: c.config.Timeout,
	}

	c.logger.Debug("sending request to WeaveVM data retriever",
		"url", r.URL.String(),
		"headers", r.Header)

	resp, err := client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to call weaveVM-data-retriever: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := validateResponse(resp, body); err != nil {
		c.logger.Error("invalid response from WeaveVM data retriever",
			"status", resp.Status,
			"content_type", resp.Header.Get("Content-Type"),
			"body", string(body))
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	var weaveVMData WvmRetrieverResponse
	if err = json.Unmarshal(body, &weaveVMData); err != nil {
		c.logger.Error("failed to unmarshal response",
			"error", err,
			"body", string(body),
			"content_type", resp.Header.Get("Content-Type"))
		return nil, fmt.Errorf("failed to unmarshal response: %w, body: %s", err, string(body))
	}

	c.logger.Info("weaveVM backend: get data from weaveVM",
		"arweave_block_hash", weaveVMData.ArweaveBlockHash,
		"weavevm_block_hash", weaveVMData.WvmBlockHash,
		"calldata_length", len(weaveVMData.Calldata))

	blob, err := hexutil.Decode(weaveVMData.Calldata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode calldata: %w", err)
	}

	if len(blob) == 0 {
		return nil, fmt.Errorf("decoded blob has length zero")
	}

	return &WvmDymintBlob{ArweaveBlockHash: weaveVMData.ArweaveBlockHash, WvmBlockHash: weaveVMData.WvmBlockHash, WvmTxHash: txHash, Blob: blob}, nil
}

func validateResponse(resp *http.Response, body []byte) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("unexpected content type: %s, body: %s", contentType, string(body))
	}

	return nil
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

					if result.Code != da.StatusSuccess {
						c.logger.Error("Blob submitted not found in DA. Retrying availability check.")
						return da.ErrBlobNotFound
					}

					return nil
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
	}

	wvmBlob, err := c.retrieveFromGateway(ctx, daMetaData.WvmTxHash)
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrBlobNotFound,
			},
			CheckMetaData: DACheckMetaData,
		}
	}

	if err := c.verifyBlobData(daMetaData.Commitment, wvmBlob.Blob); err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrProofNotMatching,
			},
		}
	}

	// If ArweaveBlockHash is missing in metadata but available in the blob, update it.
	if DACheckMetaData.WvmArweaveBlockHash == "" && wvmBlob.ArweaveBlockHash != "" {
		DACheckMetaData.WvmArweaveBlockHash = wvmBlob.ArweaveBlockHash
	}

	if DACheckMetaData.Height < wvmBlob.WvmBlockNumber {
		// Update metadata only if the blob represents a higher block (reorg case)
		DACheckMetaData.WvmArweaveBlockHash = wvmBlob.ArweaveBlockHash
		DACheckMetaData.WvmBlockHash = wvmBlob.WvmBlockHash
		DACheckMetaData.Height = wvmBlob.WvmBlockNumber
	}

	// Ensure WvmBlockHash matches the latest blob hash for consistency
	if DACheckMetaData.WvmBlockHash != wvmBlob.WvmBlockHash {
		DACheckMetaData.WvmBlockHash = wvmBlob.WvmBlockHash
	}

	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "batch available",
		},
		CheckMetaData: DACheckMetaData,
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
