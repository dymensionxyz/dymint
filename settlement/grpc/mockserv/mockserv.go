package mockserv

import (
	"context"
	"encoding/binary"
	"log"

	"google.golang.org/grpc"

	"github.com/dymensionxyz/dymint/settlement"
	slmock "github.com/dymensionxyz/dymint/settlement/grpc/mockserv/proto"
	"github.com/dymensionxyz/dymint/store"
)

var settlementKVPrefix = []byte{0}
var slStateIndexKey = []byte("slStateIndex")

type server struct {
	slmock.UnimplementedMockSLServer
	kv store.KVStore
}

func getKey(key uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, key)
	return b
}

func (s *server) GetIndex(ctx context.Context, in *slmock.SLGetIndexRequest) (*slmock.SLGetIndexReply, error) {
	b, err := s.kv.Get(slStateIndexKey)
	if err != nil {
		return nil, err
	}
	slStateIndex := binary.BigEndian.Uint64(b)
	log.Printf("Getting index %d", slStateIndex)

	return &slmock.SLGetIndexReply{Index: slStateIndex}, nil
}

func (s *server) SetIndex(ctx context.Context, in *slmock.SLSetIndexRequest) (*slmock.SLSetIndexResult, error) {
	log.Printf("Setting index to: %v", in.GetIndex())
	slStateIndex := in.GetIndex()
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, slStateIndex)
	err := s.kv.Set(slStateIndexKey, b)
	if err != nil {
		return nil, err
	}
	return &slmock.SLSetIndexResult{Index: binary.BigEndian.Uint64(b)}, nil
}

func (s *server) GetBatch(ctx context.Context, in *slmock.SLGetBatchRequest) (*slmock.SLGetBatchReply, error) {
	log.Printf("Getting batch for index: %v", in.GetIndex())
	b, err := s.kv.Get(getKey(in.GetIndex()))
	log.Printf("Retrieving batch from settlement layer SL state index %d", in.GetIndex())
	if err != nil {
		return nil, err
	}
	return &slmock.SLGetBatchReply{Index: in.GetIndex(), Batch: b}, nil
}

func (s *server) SetBatch(ctx context.Context, in *slmock.SLSetBatchRequest) (*slmock.SLSetBatchReply, error) {
	log.Printf("Setting batch for index: %v", in.GetIndex())
	err := s.kv.Set(getKey(in.GetIndex()), in.GetBatch())
	if err != nil {
		return nil, err
	}
	return &slmock.SLSetBatchReply{Result: in.GetIndex()}, nil
}

// GetServer creates and returns gRPC server instance.
func GetServer(conf settlement.GrpcConfig) *grpc.Server {
	//logger := tmlog.NewTMLogger(os.Stdout)

	srv := grpc.NewServer()

	slstore := store.NewDefaultKVStore(".", "db", "settlement")
	kv := store.NewPrefixKV(slstore, settlementKVPrefix)

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
