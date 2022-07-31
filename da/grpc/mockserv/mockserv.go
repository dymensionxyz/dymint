package mockserv

import (
	"context"
	"os"

	tmlog "github.com/tendermint/tendermint/libs/log"
	"google.golang.org/grpc"

	grpcda "github.com/celestiaorg/optimint/da/grpc"
	"github.com/celestiaorg/optimint/da/mock"
	"github.com/celestiaorg/optimint/store"
	"github.com/celestiaorg/optimint/types"
	"github.com/celestiaorg/optimint/types/pb/dalc"
	"github.com/celestiaorg/optimint/types/pb/optimint"
)

// GetServer creates and returns gRPC server instance.
func GetServer(kv store.KVStore, conf grpcda.Config, mockConfig []byte) *grpc.Server {
	logger := tmlog.NewTMLogger(os.Stdout)

	srv := grpc.NewServer()
	mockImpl := &mockImpl{}
	err := mockImpl.mock.Init(mockConfig, kv, logger)
	if err != nil {
		logger.Error("failed to initialize mock DALC", "error", err)
		panic(err)
	}
	err = mockImpl.mock.Start()
	if err != nil {
		logger.Error("failed to start mock DALC", "error", err)
		panic(err)
	}
	dalc.RegisterDALCServiceServer(srv, mockImpl)
	return srv
}

type mockImpl struct {
	mock mock.DataAvailabilityLayerClient
}

func (m *mockImpl) SubmitBatch(_ context.Context, request *dalc.SubmitBatchRequest) (*dalc.SubmitBatchResponse, error) {
	var b types.Block
	err := b.FromProto(request.Block)
	if err != nil {
		return nil, err
	}
	resp := m.mock.SubmitBatch(&b)
	return &dalc.SubmitBatchResponse{
		Result: &dalc.DAResponse{
			Code:            dalc.StatusCode(resp.Code),
			Message:         resp.Message,
			DataLayerHeight: resp.DAHeight,
		},
	}, nil
}

func (m *mockImpl) CheckBatchAvailability(_ context.Context, request *dalc.CheckBatchAvailabilityRequest) (*dalc.CheckBatchAvailabilityResponse, error) {
	resp := m.mock.CheckBatchAvailability(request.DataLayerHeight)
	return &dalc.CheckBatchAvailabilityResponse{
		Result: &dalc.DAResponse{
			Code:    dalc.StatusCode(resp.Code),
			Message: resp.Message,
		},
		DataAvailable: resp.DataAvailable,
	}, nil
}

func (m *mockImpl) RetrieveBatches(context context.Context, request *dalc.RetrieveBatchesRequest) (*dalc.RetrieveBatchesResponse, error) {
	resp := m.mock.RetrieveBatches(request.DataLayerHeight)
	blocks := make([]*optimint.Block, len(resp.Blocks))
	for i := range resp.Blocks {
		blocks[i] = resp.Blocks[i].ToProto()
	}
	return &dalc.RetrieveBatchesResponse{
		Result: &dalc.DAResponse{
			Code:    dalc.StatusCode(resp.Code),
			Message: resp.Message,
		},
		Blocks: blocks,
	}, nil
}
