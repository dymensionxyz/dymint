package celestia

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/avast/retry-go"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/pubsub"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	httprpcclient "github.com/tendermint/tendermint/rpc/client/http"

	"github.com/celestiaorg/go-cnc"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
)

const (
	defaultTxPollingRetryDelay = 20 * time.Second
	defaultSubmitRetryDelay    = 10 * time.Second
	defaultTxPollingAttempts   = 5
)

type CNCClientI interface {
	SubmitPFD(ctx context.Context, namespaceID [8]byte, blob []byte, fee int64, gasLimit uint64) (*cnc.TxResponse, error)
	NamespacedShares(ctx context.Context, namespaceID [8]byte, height uint64) ([][]byte, error)
	NamespacedData(ctx context.Context, namespaceID [8]byte, height uint64) ([][]byte, error)
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

// Config stores Celestia DALC configuration parameters.
type Config struct {
	BaseURL     string        `json:"base_url"`
	AppNodeURL  string        `json:"app_node_url"`
	Timeout     time.Duration `json:"timeout"`
	Fee         int64         `json:"fee"`
	GasLimit    uint64        `json:"gas_limit"`
	NamespaceID [8]byte       `json:"namespace_id"`
}

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

	if len(config) > 0 {
		err := json.Unmarshal(config, &c.config)
		if err != nil {
			return err
		}
	}

	c.pubsubServer = pubsubServer
	// Set defaults
	var err error
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
	c.logger.Debug("Submitting to da blob with size", "size", len(blob))
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled")
			return da.ResultSubmitBatch{}
		default:
			txResponse, err := c.client.SubmitPFD(c.ctx, c.config.NamespaceID, blob, c.config.Fee, c.config.GasLimit)
			if txResponse != nil {
				if txResponse.Code != 0 {
					c.logger.Debug("Failed to submit DA batch. Emitting health event and trying again", "txResponse", txResponse, "error", err)
					// Publish an health event. Only if we failed to emit the event we return an error.
					res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, errors.New(txResponse.RawLog))
					if err != nil {
						return res
					}
				} else if err != nil {
					// Here we assume that if txResponse is not nil and also error is not nil it means that the transaction
					// was submitted (not necessarily accepted) and we still didn't get a clear status regarding it (e.g timeout).
					// hence trying to poll for it.
					c.logger.Debug("Failed to receive DA batch inclusion result. Waiting for inclusion", "txResponse", txResponse, "error", err)
					inclusionHeight, err := c.waitForTXInclusion(txResponse.TxHash)
					if err == nil {
						res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, true, nil)
						if err != nil {
							return res
						} else {
							return da.ResultSubmitBatch{
								BaseResult: da.BaseResult{
									Code:     da.StatusSuccess,
									Message:  "tx hash: " + txResponse.TxHash,
									DAHeight: inclusionHeight,
								},
							}
						}
					} else {
						c.logger.Debug("Failed to receive DA batch inclusion result. Emitting health event and trying again", "error", err)
						res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
						if err != nil {
							return res
						}
					}

				} else {
					c.logger.Debug("Successfully submitted DA batch", "txResponse", txResponse)
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
			} else {
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, errors.New("DA txResponse is nil"))
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
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
