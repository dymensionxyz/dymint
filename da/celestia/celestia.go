package celestia

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/pubsub"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	httprpcclient "github.com/tendermint/tendermint/rpc/client/http"

	cnc "github.com/celestiaorg/go-cnc"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
)

type CNCClientI interface {
	SubmitPFB(ctx context.Context, namespaceID cnc.Namespace, blob []byte, fee int64, gasLimit uint64) (*cnc.TxResponse, error)
	NamespacedShares(ctx context.Context, namespaceID cnc.Namespace, height uint64) ([][]byte, error)
	NamespacedData(ctx context.Context, namespaceID cnc.Namespace, height uint64) ([][]byte, error)
}

// DataAvailabilityLayerClient use celestia-node public API.
type DataAvailabilityLayerClient struct {
	client              CNCClientI
	pubsubServer        *pubsub.Server
	RPCClient           rpcclient.Client
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

// WithCNCClient sets CNC client.
func WithCNCClient(client CNCClientI) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).client = client
	}
}

// WithRPCClient sets rpc client.
func WithRPCClient(rpcClient rpcclient.Client) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).RPCClient = rpcClient
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
	c.RPCClient, err = httprpcclient.New(c.config.AppNodeURL, "")
	if err != nil {
		return err
	}
	c.client, err = cnc.NewClient(c.config.BaseURL, cnc.WithTimeout(c.config.Timeout))
	if err != nil {
		return err
	}

	// Apply options
	for _, apply := range options {
		apply(c)
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	return nil
}

// Start prepares DataAvailabilityLayerClient to work.
func (c *DataAvailabilityLayerClient) Start() error {
	c.logger.Info("starting Celestia Data Availability Layer Client")
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
	blob, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}
	estimatedGas := DefaultEstimateGas(uint32(len(blob)))
	gasWanted := uint64(float64(estimatedGas) * c.config.GasAdjustment)
	fees := c.calculateFees(gasWanted)
	c.logger.Debug("Submitting to da blob with size", "size", len(blob), "estimatedGas", estimatedGas, "gasAdjusted", gasWanted, "fees", fees)

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled")
			return da.ResultSubmitBatch{}
		default:
			//SubmitPFB sets an error if the txResponse has error, so we check check the txResponse for error
			txResponse, err := c.client.SubmitPFB(c.ctx, c.config.NamespaceID, blob, fees, gasWanted)
			if txResponse == nil {
				c.logger.Error("Failed to submit DA batch. Emitting health event and trying again", "error", err)
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
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

			// Here we assume that if txResponse is not nil and also error is not nil it means that the transaction
			// was submitted (not necessarily accepted) and we still didn't get a clear status regarding it (e.g timeout).
			// hence trying to poll for it.
			daHeight := uint64(txResponse.Height)
			if daHeight == 0 {
				c.logger.Debug("Failed to receive DA batch inclusion result. Waiting for inclusion", "txHash", txResponse.TxHash)
				daHeight, err = c.waitForTXInclusion(txResponse.TxHash)
				if err != nil {
					c.logger.Error("Failed to receive DA batch inclusion result. Emitting health event and trying again", "error", err)
					res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
					if err != nil {
						return res
					}
					time.Sleep(c.submitRetryDelay)
					continue
				}
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
					DAHeight: daHeight,
				},
			}
		}
	}
}

// CheckBatchAvailability queries DA layer to check data availability of block at given height.
func (c *DataAvailabilityLayerClient) CheckBatchAvailability(dataLayerHeight uint64) da.ResultCheckBatch {
	shares, err := c.client.NamespacedShares(c.ctx, c.config.NamespaceID, dataLayerHeight)
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}

	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:     da.StatusSuccess,
			DAHeight: dataLayerHeight,
		},
		DataAvailable: len(shares) > 0,
	}
}

// RetrieveBatches gets a batch of blocks from DA layer.
func (c *DataAvailabilityLayerClient) RetrieveBatches(dataLayerHeight uint64) da.ResultRetrieveBatch {
	data, err := c.client.NamespacedData(c.ctx, c.config.NamespaceID, dataLayerHeight)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}

	batches := make([]*types.Batch, len(data))
	for i, msg := range data {
		var batch pb.Batch
		err = proto.Unmarshal(msg, &batch)
		if err != nil {
			c.logger.Error("failed to unmarshal batch", "daHeight", dataLayerHeight, "position", i, "error", err)
			continue
		}
		batches[i] = new(types.Batch)
		err := batches[i].FromProto(&batch)
		if err != nil {
			return da.ResultRetrieveBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: err.Error(),
				},
			}
		}
	}

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:     da.StatusSuccess,
			DAHeight: dataLayerHeight,
		},
		Batches: batches,
	}
}

// FIXME(omritoptix): currently we're relaying on a node without validating it using a light client.
// should be proxied through light client once it's supported (https://github.com/dymensionxyz/dymint/issues/335).
func (c *DataAvailabilityLayerClient) waitForTXInclusion(txHash string) (uint64, error) {

	hashBytes, err := hex.DecodeString(txHash)
	if err != nil {
		return 0, err
	}

	inclusionHeight := uint64(0)

	err = retry.Do(func() error {
		result, err := c.RPCClient.Tx(c.ctx, hashBytes, false)
		if err != nil {
			return err
		}

		if result == nil || err != nil {
			c.logger.Error("couldn't get transaction from node", "err", err)
			return errors.New("transaction not found")
		}

		inclusionHeight = uint64(result.Height)

		return nil
	}, retry.Attempts(uint(c.txPollingAttempts)), retry.DelayType(retry.FixedDelay), retry.Delay(c.txPollingRetryDelay))

	if err != nil {
		return 0, err
	}
	return inclusionHeight, nil
}
