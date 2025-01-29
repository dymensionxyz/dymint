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

			submitMetadata := &DASubmitMetaData{Height: resp.Result.DataLayerHeight}

			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusCode(resp.Result.Code),
					Message: resp.Result.Message,
				},
				SubmitMetaData: &da.DASubmitMetaData{
					Client: da.Grpc,
					DAPath: submitMetadata.ToPath(),
				},
			}
		}
	}
}

// CheckBatchAvailability proxies CheckBatchAvailability request to gRPC server.
func (d *DataAvailabilityLayerClient) CheckBatchAvailability(daPath string) da.ResultCheckBatch {
	backoff := d.getBackoff()

	submitMetadata := &DASubmitMetaData{}
	daMetaData, err := submitMetadata.FromPath(daPath)
	if err != nil {
		return da.ResultCheckBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err}}
	}
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
func (d *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return maxBlobSize
}

// RetrieveBatches proxies RetrieveBlocks request to gRPC server.
func (d *DataAvailabilityLayerClient) RetrieveBatches(daPath string) da.ResultRetrieveBatch {
	backoff := d.getBackoff()

	daMetaData := &DASubmitMetaData{}
	daMetaData, err := daMetaData.FromPath(daPath)
	if err != nil {
		return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err}}
	}
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

// DAMetaData contains meta data about a batch on the Data Availability Layer.
type DASubmitMetaData struct {
	// Height is the height of the block in the da layer
	Height uint64
}

// ToPath converts a DAMetaData to a path.
func (d *DASubmitMetaData) ToPath() string {
	return strconv.FormatUint(d.Height, 10)
}

// FromPath parses a path to a DAMetaData.
func (d *DASubmitMetaData) FromPath(path string) (*DASubmitMetaData, error) {

	height, err := strconv.ParseUint(path, 10, 64)
	if err != nil {
		return nil, err
	}

	submitData := &DASubmitMetaData{
		Height: height,
	}

	return submitData, nil
}
