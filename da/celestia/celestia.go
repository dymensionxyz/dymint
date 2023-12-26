package celestia

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"cosmossdk.io/math"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/pubsub"

	openrpc "github.com/rollkit/celestia-openrpc"

	"github.com/rollkit/celestia-openrpc/types/blob"
	"github.com/rollkit/celestia-openrpc/types/share"

	"github.com/dymensionxyz/dymint/da"
	celtypes "github.com/dymensionxyz/dymint/da/celestia/types"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
)

// DataAvailabilityLayerClient use celestia-node public API.
type DataAvailabilityLayerClient struct {
	rpc celtypes.CelestiaRPCClient

	pubsubServer        *pubsub.Server
	config              Config
	logger              log.Logger
	ctx                 context.Context
	cancel              context.CancelFunc
	txPollingRetryDelay time.Duration
	txPollingAttempts   int
	submitRetryDelay    time.Duration
}

var _ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
var _ da.BatchRetriever = &DataAvailabilityLayerClient{}

// WithRPCClient sets rpc client.
func WithRPCClient(rpc celtypes.CelestiaRPCClient) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).rpc = rpc
	}
}

// WithTxPollingRetryDelay sets tx polling retry delay.
func WithTxPollingRetryDelay(delay time.Duration) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).txPollingRetryDelay = delay
	}
}

// WithTxPollingAttempts sets tx polling retry delay.
func WithTxPollingAttempts(attempts int) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).txPollingAttempts = attempts
	}
}

// WithSubmitRetryDelay sets submit retry delay.
func WithSubmitRetryDelay(delay time.Duration) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).submitRetryDelay = delay
	}
}

// Init initializes DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KVStore, logger log.Logger, options ...da.Option) error {
	c.logger = logger

	if len(config) <= 0 {
		return errors.New("config is empty")
	}
	err := json.Unmarshal(config, &c.config)
	if err != nil {
		return err
	}
	err = (&c.config).InitNamespaceID()
	if err != nil {
		return err
	}

	if c.config.GasPrices != 0 && c.config.Fee != 0 {
		return errors.New("can't set both gas prices and fee")
	}

	if c.config.Fee == 0 && c.config.GasPrices == 0 {
		return errors.New("fee or gas prices must be set")
	}

	if c.config.GasAdjustment == 0 {
		c.config.GasAdjustment = defaultGasAdjustment
	}

	c.pubsubServer = pubsubServer
	// Set defaults
	c.txPollingRetryDelay = defaultTxPollingRetryDelay
	c.txPollingAttempts = defaultTxPollingAttempts
	c.submitRetryDelay = defaultSubmitRetryDelay

	c.ctx, c.cancel = context.WithCancel(context.Background())

	// Apply options
	for _, apply := range options {
		apply(c)
	}

	return nil
}

// Start prepares DataAvailabilityLayerClient to work.
func (c *DataAvailabilityLayerClient) Start() (err error) {
	c.logger.Info("starting Celestia Data Availability Layer Client")

	// other client has already been set
	if c.rpc != nil {
		return nil
	}

	rpc, err := openrpc.NewClient(c.ctx, c.config.BaseURL, c.config.AuthToken)
	if err != nil {
		return err
	}
	state, err := rpc.Header.SyncState(c.ctx)
	if err != nil {
		return err
	}
	if !state.Finished() {
		c.logger.Info("waiting for celestia-node to finish syncing", "height", state.Height, "target", state.ToHeight)
		err := rpc.Header.SyncWait(c.ctx)
		if err != nil {
			return err
		}
	}

	c.rpc = NewOpenRPC(rpc)
	return nil
}

// Stop stops DataAvailabilityLayerClient.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.logger.Info("stopping Celestia Data Availability Layer Client")
	err := c.pubsubServer.Stop()
	if err != nil {
		return err
	}
	c.cancel()
	return nil
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Celestia
}

// SubmitBatch submits a batch to the DA layer.
func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	data, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}

	blockBlob, err := blob.NewBlobV0(c.config.NamespaceID.Bytes(), data)
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}
	blobs := []*blob.Blob{blockBlob}

	estimatedGas := DefaultEstimateGas(uint32(len(data)))
	gasWanted := uint64(float64(estimatedGas) * c.config.GasAdjustment)
	fees := c.calculateFees(gasWanted)
	c.logger.Debug("Submitting to da blob with size", "size", len(blockBlob.Data), "estimatedGas", estimatedGas, "gasAdjusted", gasWanted, "fees", fees)

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled")
			return da.ResultSubmitBatch{}
		default:
			txResponse, err := c.rpc.SubmitPayForBlob(c.ctx, math.NewInt(fees), gasWanted, blobs)
			if err != nil {
				c.logger.Error("Failed to submit DA batch. Emitting health event and trying again", "error", err)
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
				continue
			}

			if txResponse == nil {
				c.logger.Error("Failed to submit DA batch. Emitting health event and trying again", "error", "txResponse is nil")
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, errors.New("txResponse is nil"))
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
				continue
			}

			if txResponse.Code != 0 {
				c.logger.Error("Failed to submit DA batch. Emitting health event and trying again", "txResponse", txResponse.RawLog, "code", txResponse.Code)
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, errors.New(txResponse.RawLog))
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
				continue
			}

			c.logger.Info("Successfully submitted DA batch", "txHash", txResponse.TxHash, "daHeight", txResponse.Height, "gasWanted", txResponse.GasWanted, "gasUsed", txResponse.GasUsed)
			res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, true, nil)
			if err != nil {
				return res
			}
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:     da.StatusSuccess,
					Message:  "tx hash: " + txResponse.TxHash,
					DAHeight: uint64(txResponse.Height),
				},
			}
		}
	}
}

// RetrieveBatches gets a batch of blocks from DA layer.
func (c *DataAvailabilityLayerClient) RetrieveBatches(dataLayerHeight uint64) da.ResultRetrieveBatch {
	blobs, err := c.rpc.GetAll(c.ctx, dataLayerHeight, []share.Namespace{c.config.NamespaceID.Bytes()})
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}

	var batches []*types.Batch
	for i, blob := range blobs {
		var batch pb.Batch
		err = proto.Unmarshal(blob.Data, &batch)
		if err != nil {
			c.logger.Error("failed to unmarshal block", "daHeight", dataLayerHeight, "position", i, "error", err)
			continue
		}
		parsedBatch := new(types.Batch)
		err := parsedBatch.FromProto(&batch)
		if err != nil {
			return da.ResultRetrieveBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: err.Error(),
				},
			}
		}
		batches = append(batches, parsedBatch)
	}

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:     da.StatusSuccess,
			DAHeight: dataLayerHeight,
		},
		Batches: batches,
	}
}
