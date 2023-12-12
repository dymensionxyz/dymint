package mockserv

import (
	"context"
	"log"

	"google.golang.org/grpc"

	grpcsl "github.com/dymensionxyz/dymint/settlement/grpc"
	slmock "github.com/dymensionxyz/dymint/settlement/grpc/mockserv/proto"
	"github.com/dymensionxyz/dymint/store"
)

type server struct {
	slmock.UnimplementedMockSLServer
	kv store.KVStore
}

func (s *server) GetIndex(ctx context.Context, in *slmock.SLIndexRequest) (*slmock.SLIndexReply, error) {
	log.Printf("Received: %v", in.GetKey())
	return &slmock.SLIndexReply{Index: 0}, nil
}

func (s *server) SetIndex(ctx context.Context, in *slmock.SLIndexReply) (*slmock.SLSetIndexResult, error) {
	log.Printf("Received: %v", in.GetIndex())
	return &slmock.SLSetIndexResult{}, nil
}

func (s *server) GetBatch(ctx context.Context, in *slmock.SLGetBatchRequest) (*slmock.SLGetBatchReply, error) {
	log.Printf("Received: %v", in.GetIndex())
	return &slmock.SLGetBatchReply{}, nil
}

func (s *server) SetBatch(ctx context.Context, in *slmock.SLSetBatchRequest) (*slmock.SLSetBatchReply, error) {
	log.Printf("Received: %v", in.GetIndex())
	return &slmock.SLSetBatchReply{}, nil
}

// GetServer creates and returns gRPC server instance.
func GetServer(conf grpcsl.Config) *grpc.Server {
	//logger := tmlog.NewTMLogger(os.Stdout)

	srv := grpc.NewServer()

	kv := store.NewDefaultKVStore(".", "db", "settlement")
	mockImpl := &server{kv: kv}

	/*err := mockImpl.mock.Init(mockConfig, pubsub.NewServer(), kv, logger)
	if err != nil {
		logger.Error("failed to initialize mock SL", "error", err)
		panic(err)
	}
	err = mockImpl.mock.Start()
	if err != nil {
		logger.Error("failed to start mock SL", "error", err)
		panic(err)
	}*/
	slmock.RegisterMockSLServer(srv, mockImpl)
	return srv
}
