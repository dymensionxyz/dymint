package interchain

import (
	"github.com/tendermint/tendermint/libs/pubsub"
	"google.golang.org/grpc"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

// DataAvailabilityLayerClient is a client for DA-provider blockchains supporting the interchain-da module.
type DataAvailabilityLayerClient struct {
	logger types.Logger

	pubsubServer *pubsub.Server

	chainConfig    ChainConfig
	grpc           grpc.ClientConn // grpc connection to the DA chain
	encodingConfig interface{}     // The DA chain's encoding config TODO: come up with the proper type

	synced chan struct{}
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

func (c *DataAvailabilityLayerClient) Init(config []byte, _ *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
	panic("implement me")
}

func (c *DataAvailabilityLayerClient) Start() error {
	panic("implement me")
}

func (c *DataAvailabilityLayerClient) Stop() error {
	panic("implement me")
}

// Synced returns channel for on sync event
func (c *DataAvailabilityLayerClient) Synced() <-chan struct{} {
	return c.synced
}

func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Interchain
}

func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	panic("implement me")
}

func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	panic("implement me")
}

func (c *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	panic("implement me")
}
