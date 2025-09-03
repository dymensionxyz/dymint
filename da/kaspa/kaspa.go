package kaspa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/kaspa/client"
	"github.com/dymensionxyz/dymint/da/stub"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/metrics"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// ToPath converts a SubmitMetaData to a path.
func (d *SubmitMetaData) ToPath() string {
	path := []string{}

	path = append(path, d.TxHash...)
	path = append(path, d.BlobHash)
	for i, part := range path {
		path[i] = strings.Trim(part, da.PathSeparator)
	}
	return strings.Join(path, da.PathSeparator)
}

// FromPath parses a path to a SubmitMetaData.
func (d *SubmitMetaData) FromPath(path string) (*SubmitMetaData, error) {
	pathParts := strings.FieldsFunc(path, func(r rune) bool { return r == rune(da.PathSeparator[0]) })
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid DA path: expected at least 2 parts, got %d", len(pathParts))
	}

	submitData := &SubmitMetaData{
		TxHash:   []string{},
		BlobHash: pathParts[len(pathParts)-1],
	}

	for i := 0; i < len(pathParts)-1; i++ {
		submitData.TxHash = append(submitData.TxHash, pathParts[i])
	}
	return submitData, nil
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

type DataAvailabilityLayerClient struct {
	stub.Layer
	client             client.KaspaClient
	pubsubServer       *pubsub.Server
	config             client.Config
	logger             types.Logger
	ctx                context.Context
	cancel             context.CancelFunc
	batchRetryDelay    time.Duration
	batchRetryAttempts uint
}

// SubmitMetaData contains meta data about a batch on the Data Availability Layer.
type SubmitMetaData struct {
	TxHash   []string
	BlobHash string
}

// WithKaspaClient sets kaspa client.
func WithKaspaClient(client client.KaspaClient) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).client = client
	}
}

// WithBatchRetryDelay is an option which sets the delay between batch retries.
func WithBatchRetryDelay(delay time.Duration) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).batchRetryDelay = delay
	}
}

// WithBatchRetryAttempts is an option which sets the number of batch retries.
func WithBatchRetryAttempts(attempts uint) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).batchRetryAttempts = attempts
	}
}

// Init initializes DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
	c.logger = logger

	var err error
	c.config, err = client.CreateConfig(config)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}

	if len(config) > 0 {
		err := json.Unmarshal(config, &c.config)
		if err != nil {
			return err
		}
	}

	// Set defaults
	c.pubsubServer = pubsubServer

	c.batchRetryDelay = c.config.RetryDelay
	c.batchRetryAttempts = *c.config.RetryAttempts

	// Apply options
	for _, apply := range options {
		apply(c)
	}

	metrics.RollappConsecutiveFailedDASubmission.Set(0)
	return nil
}

// Start starts DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Start() error {
	c.logger.Info("Starting Kaspa Data Availability Layer Client.")
	c.ctx, c.cancel = context.WithCancel(context.Background())
	// other client has already been set
	if c.client != nil {
		c.logger.Info("Kaspa client already set.")
		return nil
	}

	mnemonic := os.Getenv(c.config.MnemonicEnv)
	if mnemonic == "" {
		return fmt.Errorf("mnemonic environment variable %s is not set or empty", c.config.MnemonicEnv)
	}

	client, err := client.NewClient(c.ctx, &c.config, mnemonic)
	if err != nil {
		return fmt.Errorf("error while establishing connection to DA layer: %w", err)
	}
	c.client = client
	return nil
}

// Stop stops DataAvailabilityLayerClient.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.logger.Info("Stopping Kaspa Data Availability Layer Client.")
	err := c.client.Stop()
	if err != nil {
		c.logger.Error("Stopping Kaspa client", "err", err)
	}
	return nil
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Kaspa
}

// SubmitBatch submits batch to DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	blob, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}

	c.logger.Debug("Submitting to da batch with size", "size", len(blob))
	return c.submitBatchLoop(blob)
}

// submitBatchLoop tries submitting the batch, keep trying indefinitely.
func (c *DataAvailabilityLayerClient) submitBatchLoop(dataBlob []byte) da.ResultSubmitBatch {
	backoff := c.config.Backoff.Backoff()

	for {
		select {
		case <-c.ctx.Done():
			return da.ResultSubmitBatch{}
		default:

			// TODO: differentiate between unrecoverable and recoverable errors.(https://github.com/dymensionxyz/dymint/issues/1416)
			txHash, blobHash, err := c.client.SubmitBlob(dataBlob)
			if err != nil {
				metrics.RollappConsecutiveFailedDASubmission.Inc()
				c.logger.Error("submit blob", "err", err)
				backoff.Sleep()
				continue
			}
			metrics.RollappConsecutiveFailedDASubmission.Set(0)
			submitMetadata := &SubmitMetaData{
				TxHash:   txHash,
				BlobHash: blobHash,
			}

			err = retry.Do(
				func() error {
					result := c.checkBatchAvailability(submitMetadata)
					return result.Error
				},
				retry.Context(c.ctx),
				retry.LastErrorOnly(true),
				retry.Delay(c.batchRetryDelay),
				retry.DelayType(retry.FixedDelay),
				retry.Attempts(c.batchRetryAttempts),
			)

			if err != nil {
				c.logger.Error("Check batch availability: submitted batch but did not get availability success status.", "error", err)
				metrics.RollappConsecutiveFailedDASubmission.Inc()
				backoff.Sleep()
				continue
			}

			c.logger.Debug("Successfully submitted batch.")
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "success",
				},
				SubmitMetaData: &da.DASubmitMetaData{
					DAPath: submitMetadata.ToPath(),
					Client: da.Kaspa,
				},
			}
		}
	}
}

// RetrieveBatches retrieves batch from DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) RetrieveBatches(daPath string) da.ResultRetrieveBatch {
	daMetaData := &SubmitMetaData{}
	daMetaData, err := daMetaData.FromPath(daPath)
	if err != nil {
		return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: "wrong da path", Error: err}}
	}

	blob, err := c.client.GetBlob(daMetaData.TxHash)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Blob not found",
				Error:   err,
			},
		}
	}

	batch := &types.Batch{}
	err = batch.UnmarshalBinary(blob)
	if err != nil {
		c.logger.Error("Unmarshaling blob", "error", err)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Error unmarshaling batch: %s", err),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code: da.StatusSuccess,
		},
		Batches: []*types.Batch{batch},
	}
}

// CheckBatchAvailability checks if a batch is available on BNB by verifying the transaction and commitment exists onchain
func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daPath string) da.ResultCheckBatch {
	// Parse the DA path to get the transaction hash
	daMetaData := &SubmitMetaData{}
	daMetaData, err := daMetaData.FromPath(daPath)
	if err != nil {
		return da.ResultCheckBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: "wrong da path", Error: err}}
	}

	var result da.ResultCheckBatch
	err = retry.Do(
		func() error {
			result = c.checkBatchAvailability(daMetaData)
			if result.Error == da.ErrBlobNotFound {
				return retry.Unrecoverable(result.Error)
			}
			return result.Error
		},
		retry.Attempts(c.batchRetryAttempts), //nolint:gosec // RetryAttempts should be always positive
		retry.DelayType(retry.FixedDelay),
		retry.Delay(c.batchRetryDelay),
	)
	if err != nil {
		c.logger.Error("CheckBatchAvailability", "hash", daMetaData.TxHash, "error", err)
	}
	return result

}

// GetSignerBalance returns the balance for a specific address. //TODO: implement balance refresh func.(https://github.com/dymensionxyz/dymint/issues/1415)
func (d *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	balance, err := d.client.GetBalance()
	if err != nil {
		return da.Balance{}, err
	}
	return da.Balance{
		Amount: math.NewIntFromUint64(balance),
		Denom:  "Sompi",
	}, nil
}

// GetMaxBlobSizeBytes returns the maximum allowed blob size in the DA, used to check the max batch size configured
func (d *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return client.MaxBlobSizeBytes
}

func (c *DataAvailabilityLayerClient) checkBatchAvailability(daMetaData *SubmitMetaData) da.ResultCheckBatch {
	var currentDelay time.Duration = c.batchRetryDelay

	err := retry.Do(
		func() error {
			// First check if transactions are mature enough
			err := c.client.CheckTransactionMaturity(daMetaData.TxHash)
			if err != nil {
				// Check if this is a maturity error with missing confirmations info
				var maturityErr da.ErrMaturityNotReached
				if errors.As(err, &maturityErr) {
					// Calculate dynamic delay based on missing confirmations. Kaspa has 10 blocks per second, so we estimate delay accordingly
					currentDelay += time.Duration(float64(maturityErr.MissingConfirmations)*0.1) * time.Second

					c.logger.Debug("Transaction not mature yet, retrying with dynamic delay",
						"txHash", daMetaData.TxHash,
						"missingConfirmations", maturityErr.MissingConfirmations,
						"retryDelay", currentDelay)
				}
				return err // Return error to trigger retry
			}
			return nil // Success
		},
		retry.Attempts(c.batchRetryAttempts), //nolint:gosec // RetryAttempts should be always positive
		retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
			// Use dynamic delay for maturity errors
			return currentDelay
		}),
	)

	if err != nil {
		c.logger.Error("checkBatchAvailability", "hash", daMetaData.TxHash, "error", err)
		var maturityErr da.ErrMaturityNotReached
		if errors.As(err, &maturityErr) {
			return da.ResultCheckBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: fmt.Sprintf("Transaction not mature enough after retries, missing %d confirmations", maturityErr.MissingConfirmations),
					Error:   err,
				},
			}
		} else {
			return da.ResultCheckBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: fmt.Sprintf("All attempts failed: %v", err),
					Error:   err,
				},
			}
		}
	}

	// Check if the transaction exists by trying to fetch it
	blob, err := c.client.GetBlob(daMetaData.TxHash)
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Blob not found: %v", err),
				Error:   err,
			},
		}
	}

	// calculate blobhash
	h := sha256.New()
	h.Write(blob)
	blobHash := h.Sum(nil)

	if hex.EncodeToString(blobHash) != daMetaData.BlobHash {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Blob not found: %v", err),
				Error:   da.ErrBlobNotFound,
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
