package grpc

import (
	"context"
	"encoding/json"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dalc"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// DataAvailabilityLayerClient is a generic client that proxies all DA requests via gRPC.
type DataAvailabilityLayerClient struct {
	config Config

	conn   *grpc.ClientConn
	client dalc.DALCServiceClient

	logger log.Logger
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

var _ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
var _ da.BatchRetriever = &DataAvailabilityLayerClient{}

// Init sets the configuration options.
func (d *DataAvailabilityLayerClient) Init(config []byte, _ *pubsub.Server, _ store.KVStore, logger log.Logger, options ...da.Option) error {
	d.logger = logger
	if len(config) == 0 {
		logger.Info("DA GRPC Default config")
		d.config = DefaultConfig
		return nil
	}
	err := json.Unmarshal(config, &d.config)
	logger.Info("DA GRPC config", "config", config, "host", d.config.Host, "port", d.config.Port)

	return err
}

// Start creates connection to gRPC server and instantiates gRPC client.
func (d *DataAvailabilityLayerClient) Start() error {
	d.logger.Info("Starting GRPC DALC", "host", d.config.Host, "port", d.config.Port)
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
	d.logger.Info("stopoing GRPC DALC")
	return d.conn.Close()
}

// GetClientType returns client type.
func (d *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.GRPC
}

// SubmitBatch proxies SubmitBatch request to gRPC server.
func (d *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	resp, err := d.client.SubmitBatch(context.TODO(), &dalc.SubmitBatchRequest{Batch: batch.ToProto()})
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()},
		}
	}
	return da.ResultSubmitBatch{
		BaseResult: da.BaseResult{
			Code:     da.StatusCode(resp.Result.Code),
			Message:  resp.Result.Message,
			DAHeight: resp.Result.DataLayerHeight,
		},
	}
}

// CheckBatchAvailability proxies CheckBatchAvailability request to gRPC server.
func (d *DataAvailabilityLayerClient) CheckBatchAvailability(dataLayerHeight uint64) da.ResultCheckBatch {
	resp, err := d.client.CheckBatchAvailability(context.TODO(), &dalc.CheckBatchAvailabilityRequest{DataLayerHeight: dataLayerHeight})
	if err != nil {
		return da.ResultCheckBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
	}
	return da.ResultCheckBatch{
		BaseResult:    da.BaseResult{Code: da.StatusCode(resp.Result.Code), Message: resp.Result.Message},
		DataAvailable: resp.DataAvailable,
	}
}

// RetrieveBatches proxies RetrieveBlocks request to gRPC server.
func (d *DataAvailabilityLayerClient) RetrieveBatches(dataLayerHeight uint64) da.ResultRetrieveBatch {
	resp, err := d.client.RetrieveBatches(context.TODO(), &dalc.RetrieveBatchesRequest{DataLayerHeight: dataLayerHeight})
	if err != nil {
		return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
	}

	batches := make([]*types.Batch, len(resp.Batches))
	for i, batch := range resp.Batches {
		var b types.Batch
		err = b.FromProto(batch)
		if err != nil {
			return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
		}
		batches[i] = &b
	}
	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:     da.StatusCode(resp.Result.Code),
			Message:  resp.Result.Message,
			DAHeight: dataLayerHeight,
		},
		Batches: batches,
	}
}
