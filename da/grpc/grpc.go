package grpc

import (
	"context"
	"encoding/json"
	"strconv"

	"cosmossdk.io/math"
	"github.com/dymensionxyz/dymint/da/stub"
	uretry "github.com/dymensionxyz/dymint/utils/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dalc"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const maxBlobSize = 2097152 // 2MB (equivalent to avail or celestia)

// DataAvailabilityLayerClient is a generic client that proxies all DA requests via gRPC.
type DataAvailabilityLayerClient struct {
	stub.Layer
	config Config

	conn   *grpc.ClientConn
	client dalc.DALCServiceClient
	synced chan struct{}
	logger types.Logger

	// FIXME: ought to refactor so ctx is given to us by the object creator (same pattern should apply to celestia, avail clients)
	ctx    context.Context
	cancel context.CancelFunc
}

// Config contains configuration options for DataAvailabilityLayerClient.
type Config struct {
	// TODO(tzdybal): add more options!
	Host string `json:"host"`
	Port int    `json:"port"`
}

// DefaultConfig defines default values for DataAvailabilityLayerClient configuration.
var DefaultConfig = Config{
	Host: "127.0.0.1",
	Port: 7980,
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

// Init sets the configuration options.
func (d *DataAvailabilityLayerClient) Init(config []byte, _ *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
	d.logger = logger
	d.synced = make(chan struct{}, 1)
	d.ctx, d.cancel = context.WithCancel(context.Background())
	if len(config) == 0 {
		d.config = DefaultConfig
		return nil
	}
	return json.Unmarshal(config, &d.config)
}

// Start creates connection to gRPC server and instantiates gRPC client.
func (d *DataAvailabilityLayerClient) Start() error {
	d.logger.Info("starting GRPC DALC", "host", d.config.Host, "port", d.config.Port)
	d.synced <- struct{}{}

	var err error
	var opts []grpc.DialOption
	// TODO(tzdybal): add more options
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	d.conn, err = grpc.Dial(d.config.Host+":"+strconv.Itoa(d.config.Port), opts...)
	if err != nil {
		return err
	}

	d.client = dalc.NewDALCServiceClient(d.conn)
	return nil
}

// Stop closes connection to gRPC server.
func (d *DataAvailabilityLayerClient) Stop() error {
	d.cancel()
	return d.conn.Close()
}

// WaitForSyncing is used to check when the DA light client finished syncing
func (m *DataAvailabilityLayerClient) WaitForSyncing() {
	<-m.synced
}

// GetClientType returns client type.
func (d *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Grpc
}

// SubmitBatch proxies SubmitBatch request to gRPC server.
func (d *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	backoff := d.getBackoff()

	for {
		select {
		case <-d.ctx.Done():
			return da.ResultSubmitBatch{}
		default:
			resp, err := d.client.SubmitBatch(context.TODO(), &dalc.SubmitBatchRequest{Batch: batch.ToProto()})
			if err != nil {
				if !errorIsRetryable(err) {
					return da.ResultSubmitBatch{
						BaseResult: da.BaseResult{
							Code:    da.StatusError,
							Message: err.Error(),
							Error:   err,
						},
					}
				}
			}
			if err != nil {
				d.logger.Error("Submit blob.", "error", err)
				backoff.Sleep()
				continue
			}
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusCode(resp.Result.Code),
					Message: resp.Result.Message,
				},
				SubmitMetaData: &da.DASubmitMetaData{
					Client: da.Grpc,
					Height: resp.Result.DataLayerHeight,
				},
			}
		}
	}
}

// CheckBatchAvailability proxies CheckBatchAvailability request to gRPC server.
func (d *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	backoff := d.getBackoff()

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Debug("Context cancelled")
			return da.ResultCheckBatch{}
		default:
			resp, err := d.client.CheckBatchAvailability(context.TODO(), &dalc.CheckBatchAvailabilityRequest{DataLayerHeight: daMetaData.Height})
			if err != nil {
				if !errorIsRetryable(err) {
					return da.ResultCheckBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err}}
				}
				d.logger.Error("Check blob availability.", "error", err)
				backoff.Sleep()
				continue
			}
			return da.ResultCheckBatch{
				BaseResult: da.BaseResult{Code: da.StatusCode(resp.Result.Code), Message: resp.Result.Message},
			}
		}
	}
}

// GetMaxBlobSizeBytes returns the maximum allowed blob size in the DA, used to check the max batch size configured
func (d *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint32 {
	return maxBlobSize
}

// RetrieveBatches proxies RetrieveBlocks request to gRPC server.
func (d *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	backoff := d.getBackoff()

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Debug("Context cancelled")
			return da.ResultRetrieveBatch{}
		default:
			resp, err := d.client.RetrieveBatches(context.TODO(), &dalc.RetrieveBatchesRequest{DataLayerHeight: daMetaData.Height})
			if err != nil {
				if !errorIsRetryable(err) {
					return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err}}
				}
				d.logger.Error("Retrieve batches", "error", err)
				backoff.Sleep()
				continue
			}

			batches := make([]*types.Batch, len(resp.Batches))
			for i, batch := range resp.Batches {
				var b types.Batch
				err = b.FromProto(batch)
				if err != nil {
					return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err}}
				}
				batches[i] = &b
			}
			return da.ResultRetrieveBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusCode(resp.Result.Code),
					Message: resp.Result.Message,
				},
				CheckMetaData: &da.DACheckMetaData{
					Height: daMetaData.Height,
				},
				Batches: batches,
			}
		}
	}
}

func (d *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	return da.Balance{
		Amount: math.ZeroInt(),
		Denom:  "adym",
	}, nil
}

func (d *DataAvailabilityLayerClient) getBackoff() uretry.Backoff {
	return uretry.NewBackoffConfig().Backoff()
}

func errorIsRetryable(err error) bool {
	// kept it simple for now
	return true
}
