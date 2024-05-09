package mockserv

import (
	"context"
	"os"
	"time"

	"github.com/dymensionxyz/dymint/store"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"google.golang.org/grpc"

	"github.com/dymensionxyz/dymint/da"
	grpcda "github.com/dymensionxyz/dymint/da/grpc"
	"github.com/dymensionxyz/dymint/da/local"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dalc"
	"github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// GetServer creates and returns gRPC server instance.
func GetServer(kv store.KVStore, conf grpcda.Config, mockConfig []byte) *grpc.Server {
	logger := tmlog.NewTMLogger(os.Stdout)

	srv := grpc.NewServer()
	mockImpl := &mockImpl{}
	err := mockImpl.da.Init(mockConfig, pubsub.NewServer(), kv, logger)
	if err != nil {
		logger.Error("initialize mock DALC", "error", err)
		panic(err)
	}
	err = mockImpl.da.Start()
	if err != nil {
		logger.Error("start mock DALC", "error", err)
		panic(err)
	}
	dalc.RegisterDALCServiceServer(srv, mockImpl)
	return srv
}

type mockImpl struct {
	da local.DataAvailabilityLayerClient
}

func (m *mockImpl) SubmitBatch(_ context.Context, request *dalc.SubmitBatchRequest) (*dalc.SubmitBatchResponse, error) {
	var b types.Batch
	err := b.FromProto(request.Batch)
	if err != nil {
		return nil, err
	}
	resp := m.da.SubmitBatch(&b)
	return &dalc.SubmitBatchResponse{
		Result: &dalc.DAResponse{
			Code:            dalc.StatusCode(resp.Code),
			Message:         resp.Message,
			DataLayerHeight: resp.SubmitMetaData.Height,
		},
	}, nil
}

func (m *mockImpl) CheckBatchAvailability(_ context.Context, request *dalc.CheckBatchAvailabilityRequest) (*dalc.CheckBatchAvailabilityResponse, error) {
	daMetaData := &da.DASubmitMetaData{
		Height: request.DataLayerHeight,
	}
	resp := m.da.CheckBatchAvailability(daMetaData)
	return &dalc.CheckBatchAvailabilityResponse{
		Result: &dalc.DAResponse{
			Code:    dalc.StatusCode(resp.Code),
			Message: resp.Message,
		},
	}, nil
}

func (m *mockImpl) RetrieveBatches(context context.Context, request *dalc.RetrieveBatchesRequest) (*dalc.RetrieveBatchesResponse, error) {
	dataMetaData := &da.DASubmitMetaData{
		Height: request.DataLayerHeight,
	}
	resp := m.da.RetrieveBatches(dataMetaData)
	batches := make([]*dymint.Batch, len(resp.Batches))
	for i := range resp.Batches {
		batches[i] = resp.Batches[i].ToProto()
	}
	return &dalc.RetrieveBatchesResponse{
		Result: &dalc.DAResponse{
			Code:    dalc.StatusCode(resp.Code),
			Message: resp.Message,
		},
		Batches: batches,
	}, nil
}
