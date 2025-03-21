package bnb

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/stub"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/metrics"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const (
	maxBlobSize               = 100000
	defaultTxInclusionTimeout = 100 * time.Second
	defaultBatchRetryDelay    = 10 * time.Second
	defaultBatchRetryAttempts = 10
)

type BNBConfig struct {
	Account               string  `json:"account"`
	Network               string  `json:"network"`
	Endpoint              string  `json:"endpoint"`
	Contract              string  `json:"contract"`
	PrivateKey            string  `json:"key"`
	FeeLimitMultiplier    uint64  `json:"fee_limit_multiplier"`
	FeeLimitThresholdGwei float64 `json:"fee_limit_threshold_gwei"`
	BlobGasPriceLimitGwei float64 `json:"blob_gas_price_limit_gwei"`
	MinBaseFeeGwei        float64 `json:"min_base_fee_gwei"`
	MinTipCapGwei         float64 `json:"min_tip_cap_gwei"`
}

// ToPath converts a SubmitMetaData to a path.
func (d *SubmitMetaData) ToPath() string {
	path := []string{
		hex.EncodeToString(d.FrameRef),
	}
	for i, part := range path {
		path[i] = strings.Trim(part, da.PathSeparator)
	}
	return strings.Join(path, da.PathSeparator)
}

// FromPath parses a path to a SubmitMetaData.
func (d *SubmitMetaData) FromPath(path string) (*SubmitMetaData, error) {
	pathParts := strings.FieldsFunc(path, func(r rune) bool { return r == rune(da.PathSeparator[0]) })
	if len(pathParts) != 1 {
		return nil, fmt.Errorf("invalid DA path")
	}

	ref, err := hex.DecodeString(pathParts[0])
	if err != nil {
		return nil, err
	}
	submitData := &SubmitMetaData{FrameRef: ref}
	return submitData, nil
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

type DataAvailabilityLayerClient struct {
	stub.Layer
	client             BNBClient
	pubsubServer       *pubsub.Server
	config             BNBConfig
	logger             types.Logger
	ctx                context.Context
	cancel             context.CancelFunc
	txInclusionTimeout time.Duration
	batchRetryDelay    time.Duration
	batchRetryAttempts uint
}

// SubmitMetaData contains meta data about a batch on the Data Availability Layer.
type SubmitMetaData struct {
	// Height is the height of the block in the da layer
	FrameRef []byte
}

// WithRPCClient sets rpc client.
func WithRPCClient(client BNBClient) da.Option {
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

	if len(config) > 0 {
		err := json.Unmarshal(config, &c.config)
		if err != nil {
			return err
		}
	}

	// Set defaults
	c.pubsubServer = pubsubServer

	// TODO: Make configurable
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
	c.logger.Info("Starting Avail Data Availability Layer Client.")

	// other client has already been set
	if c.client != nil {
		c.logger.Info("Avail client already set.")
		return nil
	}

	client, err := NewClient(c.ctx, c.config)
	if err != nil {
		return fmt.Errorf("error while establishing connection to DA layer: %w", err)
	}
	c.client = client
	return nil
}

// Stop stops DataAvailabilityLayerClient.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.logger.Info("Stopping Avail Data Availability Layer Client.")
	c.cancel()
	return nil
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Avail
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
	for {
		select {
		case <-c.ctx.Done():
			return da.ResultSubmitBatch{}
		default:
			var frameBytes []byte
			err := retry.Do(
				func() error {
					var err error
					frameBytes, err = c.client.SubmitBlob(dataBlob)
					if err != nil {
						metrics.RollappConsecutiveFailedDASubmission.Inc()
						c.logger.Error("broadcasting batch", "error", err)
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

				if !retry.IsRecoverable(err) {
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
			metrics.RollappConsecutiveFailedDASubmission.Set(0)
			submitMetadata := &SubmitMetaData{
				FrameRef: frameBytes,
			}

			c.logger.Debug("Successfully submitted batch.")
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "success",
				},
				SubmitMetaData: &da.DASubmitMetaData{
					DAPath: submitMetadata.ToPath(),
					Client: da.Avail,
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

	blob, err := c.client.GetBlob(daMetaData.FrameRef)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Blob not found",
				Error:   err,
			},
		}
	}

	var batches []*types.Batch
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

	// Add the batch to the list
	batches = append(batches, batch)
	// Remove the bytes we just decoded.
	size := batch.ToProto().Size()
	if len(blob) < size {
		c.logger.Error("Batch size does not match blob data length", "blob data length", len(blob), "batch size", size)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Wrong size",
				Error:   fmt.Errorf("wrong size"),
			},
		}
	}

	// if no batches, return error
	if len(batches) == 0 {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Blob not found",
				Error:   da.ErrBlobNotFound,
			},
		}
	}

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code: da.StatusSuccess,
		},
		Batches: batches,
	}
}

// CheckBatchAvailability checks batch availability in DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daPath string) da.ResultCheckBatch {
	daMetaData := &SubmitMetaData{}
	daMetaData, err := daMetaData.FromPath(daPath)
	if err != nil {
		return da.ResultCheckBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: "wrong da path", Error: err}}
	}
	return da.ResultCheckBatch{}
}

// GetSignerBalance returns the balance for a specific address
// TODO implement get balance for avail client
func (d *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	return da.Balance{
		Amount: math.ZeroInt(),
		Denom:  "avail",
	}, nil
}

// GetMaxBlobSizeBytes returns the maximum allowed blob size in the DA, used to check the max batch size configured
func (d *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return maxBlobSize
}
