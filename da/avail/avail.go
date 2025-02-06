package avail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/availproject/avail-go-sdk/metadata"
	availgo "github.com/availproject/avail-go-sdk/sdk"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/stub"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/metrics"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/vedhavyas/go-subkey/v2"
)

const (
	keyringNetworkID          uint8 = 42
	defaultTxInclusionTimeout       = 100 * time.Second
	defaultBatchRetryDelay          = 10 * time.Second
	defaultBatchRetryAttempts       = 10
	DataCallSection                 = "DataAvailability"
	DataCallMethod                  = "submit_data"
	DataCallSectionIndex            = 29
	DataCallMethodIndex             = 1
	maxBlobSize                     = 500000 // according to Avail 2MB is the max block size, but tx limit is 512KB
)

type Config struct {
	Seed        string `json:"seed"`
	RpcEndpoint string `json:"endpoint"`
	AppID       int64  `json:"app_id"`
}

type DataAvailabilityLayerClient struct {
	stub.Layer
	client             availgo.SDK
	account            subkey.KeyPair
	pubsubServer       *pubsub.Server
	config             Config
	logger             types.Logger
	ctx                context.Context
	cancel             context.CancelFunc
	txInclusionTimeout time.Duration
	batchRetryDelay    time.Duration
	batchRetryAttempts uint
	synced             chan struct{}
}

// SubmitMetaData contains meta data about a batch on the Data Availability Layer.
type SubmitMetaData struct {
	// Height is the height of the block in the da layer
	Height uint32
	// Avail App Id
	AppId int64
	// Hash that identifies the blob
	AccountAddress string
}

// ToPath converts a SubmitMetaData to a path.
func (d *SubmitMetaData) ToPath() string {
	path := []string{
		strconv.FormatUint(uint64(d.Height), 10),
		strconv.FormatInt(d.AppId, 10),
		d.AccountAddress,
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

	height, err := strconv.ParseUint(pathParts[0], 10, 32)
	if err != nil {
		return nil, err
	}

	submitData := &SubmitMetaData{
		Height: uint32(height),
	}

	submitData.AppId, err = strconv.ParseInt(pathParts[1], 10, 64)
	if err != nil {
		return nil, err
	}

	submitData.AccountAddress = pathParts[2]
	return submitData, nil
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

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
	c.synced = make(chan struct{}, 1)

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
	c.ctx, c.cancel = context.WithCancel(context.Background())
	sdk, err := availgo.NewSDK(c.config.RpcEndpoint)
	if err != nil {
		return err
	}
	c.client = sdk
	acc, err := availgo.Account.NewKeyPair(c.config.Seed)
	if err != nil {
		return err
	}
	c.account = acc

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
			var daBlockHeight uint32
			err := retry.Do(
				func() error {
					var err error
					tx := c.client.Tx.DataAvailability.SubmitData(dataBlob)
					res, err := tx.ExecuteAndWatchInclusion(c.account, availgo.NewTransactionOptions().WithAppId(uint32(c.config.AppID)))
					if err != nil {
						metrics.RollappConsecutiveFailedDASubmission.Inc()
						c.logger.Error("broadcasting batch", "error", err)
						if errors.Is(err, da.ErrTxBroadcastConfigError) {
							err = retry.Unrecoverable(err)
						}
						return err
					}
					daBlockHeight = res.BlockNumber
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
				Height:         daBlockHeight,
				AppId:          c.config.AppID,
				AccountAddress: c.account.SS58Address(42),
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

	block, err := c.getBlock(daMetaData.Height)

	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   errors.Join(da.ErrRetrieval, err),
			},
		}
	}

	accountId, err := metadata.NewAccountIdFromAddress(daMetaData.AccountAddress)

	// Convert the data returned to batches
	var batches []*types.Batch

	// Block Blobs filtered by Signer
	blobs := block.DataSubmissionBySigner(accountId)

	// Printout Block Blobs filtered by Signer
	for _, blob := range blobs {
		batch := &types.Batch{}
		err = batch.UnmarshalBinary(blob.Data)
		if err != nil {
			// try to parse from the next byte on the next iteration
			continue
		}

		// Add the batch to the list
		batches = append(batches, batch)
		// Remove the bytes we just decoded.
		size := batch.ToProto().Size()
		if len(blob.Data) < size {
			// not supposed to happen, additional safety check
			break
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
	//nolint:typecheck
	block, err := c.getBlock(daMetaData.Height)
	if err != nil {
		block, err := c.client.GetBlockLatest()
		if err != nil {
			return da.ResultCheckBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: err.Error(),
					Error:   errors.Join(da.ErrRetrieval, err),
				},
			}
		}
		if uint64(block.Block.Header.Number) < daMetaData.Height {
			return da.ResultCheckBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: "wrong da height",
					Error:   errors.Join(da.ErrBlobNotIncluded, err),
				},
			}
		}
	}
	for _, ext := range block.Block.Extrinsics {
		// these values below are specific indexes only for data submission, differs with each extrinsic
		if ext.Signature.AppID.Int64() == c.config.AppID &&
			ext.Method.CallIndex.SectionIndex == DataCallSectionIndex &&
			ext.Method.CallIndex.MethodIndex == DataCallMethodIndex {
			data := ext.Method.Args

			for 0 < len(data) {
				batch := &types.Batch{}
				err := batch.UnmarshalBinary(data)
				if err != nil {
					// try to parse from the next byte on the next iteration
					data = data[1:]
					continue
				}
				// if blob unmarshaled successfully, we check is the same batch submitted by the sequencer comparing hashes
				if bytes.Equal(crypto.Keccak256Hash(data).Bytes(), daMetaData.BlobHash) {
					return da.ResultCheckBatch{
						BaseResult: da.BaseResult{
							Code:    da.StatusSuccess,
							Message: "Blob available",
						},
					}
				} else {
					continue
				}
			}

		}
	}

	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "not implemented",
			Code:    da.StatusError,
			Message: "Blob not available",
			Error:   da.ErrBlobNotIncluded,
		},
	}
}

// GetMaxBlobSizeBytes returns the maximum allowed blob size in the DA, used to check the max batch size configured
func (d *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return maxBlobSize
}

func (c *DataAvailabilityLayerClient) getBlock(height uint32) (availgo.Block, error) {
	blockHash, err := c.client.Client.BlockHash(height)
	if err != nil {
		return availgo.Block{}, err
	}
	return availgo.NewBlock(c.client.Client, blockHash)
}
