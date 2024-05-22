package mockserv

import (
	"context"
	"encoding/binary"

	"google.golang.org/grpc"

	"github.com/dymensionxyz/dymint/settlement"
	slmock "github.com/dymensionxyz/dymint/settlement/grpc/mockserv/proto"
	"github.com/dymensionxyz/dymint/store"
)

var (
	settlementKVPrefix = []byte{0}
	slStateIndexKey    = []byte("slStateIndex")
)

type server struct {
	slmock.UnimplementedMockSLServer
	kv store.KV
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

	return &slmock.SLGetIndexReply{Index: slStateIndex}, nil
}

func (s *server) SetIndex(ctx context.Context, in *slmock.SLSetIndexRequest) (*slmock.SLSetIndexResult, error) {
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
	b, err := s.kv.Get(getKey(in.GetIndex()))
	if err != nil {
		return nil, err
	}
	return &slmock.SLGetBatchReply{Index: in.GetIndex(), Batch: b}, nil
}

func (s *server) SetBatch(ctx context.Context, in *slmock.SLSetBatchRequest) (*slmock.SLSetBatchReply, error) {
	err := s.kv.Set(getKey(in.GetIndex()), in.GetBatch())
	if err != nil {
		return nil, err
	}
	return &slmock.SLSetBatchReply{Result: in.GetIndex()}, nil
}

func GetServer(conf settlement.GrpcConfig) *grpc.Server {
	srv := grpc.NewServer()

	slstore := store.NewDefaultKVStore(".", "db", "settlement")
	kv := store.NewPrefixKV(slstore, settlementKVPrefix)

	mockImpl := &server{kv: kv}

	slmock.RegisterMockSLServer(srv, mockImpl)
	return srv
}
