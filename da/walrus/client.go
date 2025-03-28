package walrus

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/walrus/client"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const (
	maxBlobSizeBytes = 10 * 1024 * 1024 // 1MB
)

var _ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}

type DataAvailabilityLayerClient struct {
	config       Config
	client       *client.Client
	logger       types.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	pubsubServer *pubsub.Server
}

func (d *DataAvailabilityLayerClient) RetrieveBatches(daPath string) da.ResultRetrieveBatch {
	// Parse the DA path to get the blob ID
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
	blobID := daMetaData.DAPath

	d.logger.Debug("Getting blob from Walrus DA.", "blob_id", blobID)

	var resultRetrieveBatch da.ResultRetrieveBatch
	err = retry.Do(
		func() error {
			resultRetrieveBatch = d.retrieveBatches(daMetaData)
			return resultRetrieveBatch.Error
		},
		retry.Attempts(uint(*d.config.RetryAttempts)), //nolint:gosec // RetryAttempts should be always positive
		retry.DelayType(retry.FixedDelay),
		retry.Delay(d.config.RetryDelay),
	)
	if err != nil {
		d.logger.Error("Retrieve batch", "blob_id", blobID, "error", err)
	}
	return resultRetrieveBatch
}

// retrieveBatches downloads a batch from Walrus and returns the batch included
func (d *DataAvailabilityLayerClient) retrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	ctx, cancel := context.WithTimeout(d.ctx, d.config.Timeout)
	defer cancel()

	data, err := d.client.GetBlob(ctx, daMetaData.DAPath)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Failed to retrieve blob: %v", err),
				Error:   fmt.Errorf("failed to retrieve blob: %v", err),
			},
		}
	}

	if len(data) == 0 {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "No batch data found in blob",
				Error:   da.ErrBlobNotFound,
			},
		}
	}

	// Parse batch
	batch := new(types.Batch)
	if err := batch.UnmarshalBinary(data); err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Failed to unmarshal batch: %v", err),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}

	d.logger.Debug("Blob retrieved successfully from Walrus DA.", "blob_id", daMetaData.DAPath)

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Batch retrieval successful",
		},
		Batches: []*types.Batch{batch},
	}
}

func (d *DataAvailabilityLayerClient) CheckBatchAvailability(daPath string) da.ResultCheckBatch {
	// Parse the DA path to get the blob ID
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
	blobID := daMetaData.DAPath

	var result da.ResultCheckBatch
	err = retry.Do(
		func() error {
			result = d.checkBatchAvailability(daMetaData)
			return result.Error
		},
		retry.Attempts(uint(*d.config.RetryAttempts)), //nolint:gosec // RetryAttempts should be always positive
		retry.DelayType(retry.FixedDelay),
		retry.Delay(d.config.RetryDelay),
	)
	if err != nil {
		d.logger.Error("CheckBatchAvailability", "blob_id", blobID, "error", err)
	}
	return result
}

// checkBatchAvailability checks if a batch is available on Walrus
func (d *DataAvailabilityLayerClient) checkBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	ctx, cancel := context.WithTimeout(d.ctx, d.config.Timeout)
	defer cancel()

	err := d.client.CheckBlobAvailability(ctx, daMetaData.DAPath)
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Check batch availability: blob not found: %v", err),
				Error:   fmt.Errorf("check batch availability: blob not found: %w", err),
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

func (d *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
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

	backoff := d.config.Backoff.Backoff()

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Debug("Context cancelled.")
			return da.ResultSubmitBatch{}
		default:
			daMetaData, err := d.submit(data)
			if err != nil {
				d.logger.Error("Submit blob.", "error", err)
				backoff.Sleep()
				continue
			}

			result := d.checkBatchAvailability(daMetaData)
			if result.Error != nil {
				d.logger.Error("Check batch availability: submitted batch but did not get availability success status.", "error", result.Error)
				backoff.Sleep()
				continue
			}

			d.logger.Debug("Submitted blob to Walrus DA successfully.", "blob_size", len(data), "blob_id", daMetaData.DAPath)

			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "Batch submitted successfully to Walrus",
				},
				SubmitMetaData: &da.DASubmitMetaData{
					Client: da.Walrus,
					DAPath: daMetaData.DAPath,
				},
			}
		}
	}
}

// submit submits a blob to Walrus
func (d *DataAvailabilityLayerClient) submit(data []byte) (*da.DASubmitMetaData, error) {
	if len(data) > maxBlobSizeBytes {
		return nil, fmt.Errorf("batch does not fit into blob: %d bytes: limit: %d bytes", len(data), maxBlobSizeBytes)
	}

	ctx, cancel := context.WithTimeout(d.ctx, d.config.Timeout)
	defer cancel()

	blobID, err := d.client.SubmitBlob(ctx, data, d.config.StoreDurationEpochs, d.config.BlobOwnerAddr)
	if err != nil {
		return nil, fmt.Errorf("submit blob: %v", err)
	}

	if blobID == "" {
		return nil, fmt.Errorf("empty blob ID")
	}

	return &da.DASubmitMetaData{
		Client: da.Walrus,
		DAPath: blobID,
	}, nil
}

func (d *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
	d.logger = logger
	var err error
	if d.config, err = createConfig(config); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.pubsubServer = pubsubServer

	// Apply options
	for _, apply := range options {
		apply(d)
	}

	return nil
}

// Start prepares the Walrus client to work
func (d *DataAvailabilityLayerClient) Start() error {
	d.logger.Info("Starting Walrus Data Availability Layer Client.")

	// Create Walrus client
	d.client = client.NewClient(d.config.PublisherUrl, d.config.AggregatorUrl)

	d.logger.Info("Walrus client initialized successfully")
	return nil
}

// Stop stops the Walrus Data Availability Layer Client
func (d *DataAvailabilityLayerClient) Stop() error {
	d.logger.Info("Stopping Walrus Data Availability Layer Client.")
	if d.cancel != nil {
		d.cancel()
	}
	return nil
}

func (d *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Walrus
}

func (d *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return maxBlobSizeBytes
}

// GetSignerBalance cannot return any balance for Walrus DA. In theory, it should return the balance of the
// publisher, but it doesn't make sense to return the balance of public publisher.
func (d *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	return da.Balance{}, nil
}

func (d *DataAvailabilityLayerClient) RollappId() string {
	return d.config.BlobOwnerAddr
}
