package eth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/ethutils"
	"github.com/dymensionxyz/dymint/da/stub"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/metrics"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const (
	defaultTxInclusionTimeout = 100 * time.Second
	defaultBatchRetryDelay    = 10 * time.Second
	defaultBatchRetryAttempts = 10
	ArchivePoolAddress        = "0x0000000000000000000000000000000000000000" // the data settling address
)

// ToPath converts a SubmitMetaData to a path.
func (d *SubmitMetaData) ToPath() string {
	path := []string{
		string(d.Commitment),
		string(d.Proof),
		d.Slot,
	}
	for i, part := range path {
		path[i] = strings.Trim(part, da.PathSeparator)
	}
	return strings.Join(path, da.PathSeparator)
}

// FromPath parses a path to a SubmitMetaData.
func (d *SubmitMetaData) FromPath(path string) (*SubmitMetaData, error) {
	pathParts := strings.FieldsFunc(path, func(r rune) bool { return r == rune(da.PathSeparator[0]) })
	if len(pathParts) != 3 {
		return nil, fmt.Errorf("invalid DA path")
	}

	submitData := &SubmitMetaData{Commitment: []byte(pathParts[0]), Proof: []byte(pathParts[1]), Slot: pathParts[2]}
	return submitData, nil
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

type DataAvailabilityLayerClient struct {
	stub.Layer
	client             EthClient
	pubsubServer       *pubsub.Server
	config             EthConfig
	logger             types.Logger
	ctx                context.Context
	cancel             context.CancelFunc
	txInclusionTimeout time.Duration
	batchRetryDelay    time.Duration
	batchRetryAttempts uint
}

// SubmitMetaData contains meta data about a batch on the Data Availability Layer.
type SubmitMetaData struct {
	Commitment []byte
	Proof      []byte
	Slot       string
}

// WithRPCClient sets rpc client.
func WithRPCClient(client EthClient) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).client = client
	}
}

// WithTxInclusionTimeout is an option which sets the timeout for waiting for transaction inclusion.
func WithTxInclusionTimeout(timeout time.Duration) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).txInclusionTimeout = timeout
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

	c.ctx, c.cancel = context.WithCancel(context.Background())

	ethconfig, err := createConfig(config)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	c.config = ethconfig

	c.pubsubServer = pubsubServer

	c.txInclusionTimeout = defaultTxInclusionTimeout
	c.batchRetryDelay = defaultBatchRetryDelay
	c.batchRetryAttempts = defaultBatchRetryAttempts

	// Apply options
	for _, apply := range options {
		apply(c)
	}

	metrics.RollappConsecutiveFailedDASubmission.Set(0)
	return nil
}

// Start starts DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Start() error {
	c.logger.Info("Starting Ethereum Data Availability Layer Client.")
	c.ctx, c.cancel = context.WithCancel(context.Background())
	// other client has already been set
	if c.client != nil {
		c.logger.Info("Ethereum DA client already set.")
		return nil
	}

	client, err := NewClient(c.ctx, &c.config)
	if err != nil {
		return fmt.Errorf("error while establishing connection to DA layer: %w", err)
	}
	c.client = client
	return nil
}

// Stop stops DataAvailabilityLayerClient.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.logger.Info("Stopping Ethereum Data Availability Layer Client.")
	c.cancel()
	return nil
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Ethereum
}

// SubmitBatch submits batch to DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	blob, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   err,
			},
		}
	}

	c.logger.Debug("Submitting to da batch with size", "size", len(blob))
	return c.submitBatchLoop(blob)
}

// submitBatchLoop tries submitting the batch. In case we get a configuration error we would like to stop trying,
// otherwise, for network error we keep trying indefinitely.
func (c *DataAvailabilityLayerClient) submitBatchLoop(dataBlob []byte) da.ResultSubmitBatch {
	backoff := c.config.Backoff.Backoff()

	for {
		select {
		case <-c.ctx.Done():
			return da.ResultSubmitBatch{}
		default:
			var commitment []byte
			var proof []byte
			var slot string
			err := retry.Do(
				func() error {
					var err error
					commitment, proof, slot, err = c.client.SubmitBlob(dataBlob)
					if err != nil {
						metrics.RollappConsecutiveFailedDASubmission.Inc()
						c.logger.Error("broadcasting batch", "error", err)
						if errors.Is(err, ethutils.ErrBlobInputTooLarge) {
							err = retry.Unrecoverable(err)
						}
						return err
					}
					return nil
				},
				retry.Context(c.ctx),
				retry.LastErrorOnly(true),
				retry.Delay(c.batchRetryDelay),
				retry.DelayType(retry.FixedDelay),
				retry.Attempts(c.batchRetryAttempts),
			)
			if err != nil {
				err = fmt.Errorf("broadcast data blob: %w", err)

				if errors.Is(err, ethutils.ErrBlobInputTooLarge) {
					return da.ResultSubmitBatch{
						BaseResult: da.BaseResult{
							Code:    da.StatusError,
							Message: err.Error(),
							Error:   err,
						},
					}
				}

				c.logger.Error(err.Error())
				continue
			}

			submitMetadata := &SubmitMetaData{
				Commitment: commitment,
				Proof:      proof,
				Slot:       slot,
			}

			err = retry.Do(
				func() error {
					availResult := c.checkBatchAvailability(submitMetadata)
					return availResult.Error
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

			metrics.RollappConsecutiveFailedDASubmission.Set(0)

			c.logger.Debug("Successfully submitted batch.")
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "success",
				},
				SubmitMetaData: &da.DASubmitMetaData{
					DAPath: submitMetadata.ToPath(),
					Client: da.Ethereum,
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

	blob, err := c.client.GetBlob(daMetaData.Slot, daMetaData.Commitment)
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
				Message: "Err Unmarshal",
				Error:   err,
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

// CheckBatchAvailability checks if a batch is available on Ethereum by verifying the transaction and commitment exists onchain
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
		c.logger.Error("CheckBatchAvailability", "slot", daMetaData.Slot, "commitment", daMetaData.Commitment, "error", err)
	}
	return result
}

// GetSignerBalance returns the balance for a specific address
func (d *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	balance, err := d.client.GetSignerBalance()
	if err != nil {
		return da.Balance{}, err
	}
	return da.Balance{
		Amount: math.NewIntFromBigInt(balance),
		Denom:  "ETH",
	}, nil
}

// GetMaxBlobSizeBytes returns the maximum allowed blob size in the DA, used to check the max batch size configured
func (d *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return ethutils.MaxBlobDataSize
}

func (c *DataAvailabilityLayerClient) checkBatchAvailability(daMetaData *SubmitMetaData) da.ResultCheckBatch {
	// Check if the blob exists in the specified slot and matches the kzg commitment, and the commitment matches the blob data using the proof.
	err := c.client.ValidateInclusion(daMetaData.Slot, daMetaData.Commitment, daMetaData.Proof)
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Blob not found: %v", err),
				Error:   err,
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
