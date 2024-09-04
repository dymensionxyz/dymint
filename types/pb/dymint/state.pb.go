// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: types/dymint/state.proto

package dymint

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	github_com_gogo_protobuf_types "github.com/gogo/protobuf/types"
	_ "github.com/tendermint/tendermint/abci/types"
	state "github.com/tendermint/tendermint/proto/tendermint/state"
	types "github.com/tendermint/tendermint/proto/tendermint/types"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	io "io"
	math "math"
	math_bits "math/bits"
	time "time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type State struct {
	Version                          *state.Version         `protobuf:"bytes,1,opt,name=version,proto3" json:"version,omitempty"`
	ChainId                          string                 `protobuf:"bytes,2,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	InitialHeight                    int64                  `protobuf:"varint,3,opt,name=initial_height,json=initialHeight,proto3" json:"initial_height,omitempty"`
	LastBlockHeight                  int64                  `protobuf:"varint,4,opt,name=last_block_height,json=lastBlockHeight,proto3" json:"last_block_height,omitempty"`
	LastBlockID                      types.BlockID          `protobuf:"bytes,5,opt,name=last_block_id,json=lastBlockId,proto3" json:"last_block_id"`
	LastBlockTime                    time.Time              `protobuf:"bytes,6,opt,name=last_block_time,json=lastBlockTime,proto3,stdtime" json:"last_block_time"`
	Validators                       *types.ValidatorSet    `protobuf:"bytes,9,opt,name=validators,proto3" json:"validators,omitempty"` // Deprecated: Do not use.
	LastHeightValidatorsChanged      int64                  `protobuf:"varint,11,opt,name=last_height_validators_changed,json=lastHeightValidatorsChanged,proto3" json:"last_height_validators_changed,omitempty"`
	LastHeightConsensusParamsChanged int64                  `protobuf:"varint,13,opt,name=last_height_consensus_params_changed,json=lastHeightConsensusParamsChanged,proto3" json:"last_height_consensus_params_changed,omitempty"`
	LastResultsHash                  []byte                 `protobuf:"bytes,14,opt,name=last_results_hash,json=lastResultsHash,proto3" json:"last_results_hash,omitempty"`
	AppHash                          []byte                 `protobuf:"bytes,15,opt,name=app_hash,json=appHash,proto3" json:"app_hash,omitempty"`
	LastStoreHeight                  uint64                 `protobuf:"varint,16,opt,name=last_store_height,json=lastStoreHeight,proto3" json:"last_store_height,omitempty"`
	BaseHeight                       uint64                 `protobuf:"varint,17,opt,name=base_height,json=baseHeight,proto3" json:"base_height,omitempty"`
	SequencerSet                     SequencerSet           `protobuf:"bytes,18,opt,name=sequencerSet,proto3" json:"sequencerSet"`
	ConsensusParams                  RollappConsensusParams `protobuf:"bytes,19,opt,name=consensus_params,json=consensusParams,proto3" json:"consensus_params"`
	LastSubmittedHeight              uint64                 `protobuf:"varint,20,opt,name=last_submitted_height,json=lastSubmittedHeight,proto3" json:"last_submitted_height,omitempty"`
}

func (m *State) Reset()         { *m = State{} }
func (m *State) String() string { return proto.CompactTextString(m) }
func (*State) ProtoMessage()    {}
func (*State) Descriptor() ([]byte, []int) {
	return fileDescriptor_4b679420add07272, []int{0}
}
func (m *State) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *State) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_State.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *State) XXX_Merge(src proto.Message) {
	xxx_messageInfo_State.Merge(m, src)
}
func (m *State) XXX_Size() int {
	return m.Size()
}
func (m *State) XXX_DiscardUnknown() {
	xxx_messageInfo_State.DiscardUnknown(m)
}

var xxx_messageInfo_State proto.InternalMessageInfo

func (m *State) GetVersion() *state.Version {
	if m != nil {
		return m.Version
	}
	return nil
}

func (m *State) GetChainId() string {
	if m != nil {
		return m.ChainId
	}
	return ""
}

func (m *State) GetInitialHeight() int64 {
	if m != nil {
		return m.InitialHeight
	}
	return 0
}

func (m *State) GetLastBlockHeight() int64 {
	if m != nil {
		return m.LastBlockHeight
	}
	return 0
}

func (m *State) GetLastBlockID() types.BlockID {
	if m != nil {
		return m.LastBlockID
	}
	return types.BlockID{}
}

func (m *State) GetLastBlockTime() time.Time {
	if m != nil {
		return m.LastBlockTime
	}
	return time.Time{}
}

// Deprecated: Do not use.
func (m *State) GetValidators() *types.ValidatorSet {
	if m != nil {
		return m.Validators
	}
	return nil
}

func (m *State) GetLastHeightValidatorsChanged() int64 {
	if m != nil {
		return m.LastHeightValidatorsChanged
	}
	return 0
}

func (m *State) GetLastHeightConsensusParamsChanged() int64 {
	if m != nil {
		return m.LastHeightConsensusParamsChanged
	}
	return 0
}

func (m *State) GetLastResultsHash() []byte {
	if m != nil {
		return m.LastResultsHash
	}
	return nil
}

func (m *State) GetAppHash() []byte {
	if m != nil {
		return m.AppHash
	}
	return nil
}

func (m *State) GetLastStoreHeight() uint64 {
	if m != nil {
		return m.LastStoreHeight
	}
	return 0
}

func (m *State) GetBaseHeight() uint64 {
	if m != nil {
		return m.BaseHeight
	}
	return 0
}

func (m *State) GetSequencerSet() SequencerSet {
	if m != nil {
		return m.SequencerSet
	}
	return SequencerSet{}
}

func (m *State) GetConsensusParams() RollappConsensusParams {
	if m != nil {
		return m.ConsensusParams
	}
	return RollappConsensusParams{}
}

func (m *State) GetLastSubmittedHeight() uint64 {
	if m != nil {
		return m.LastSubmittedHeight
	}
	return 0
}

//rollapp params defined in genesis and updated via gov proposal
type RollappConsensusParams struct {
	//maximum amount of gas that all transactions included in a block can use
	Blockmaxgas int64 `protobuf:"varint,1,opt,name=blockmaxgas,proto3" json:"blockmaxgas,omitempty"`
	//maximum allowed block size
	Blockmaxsize int64 `protobuf:"varint,2,opt,name=blockmaxsize,proto3" json:"blockmaxsize,omitempty"`
	//data availability type (e.g. celestia) used in the rollapp
	Da string `protobuf:"bytes,3,opt,name=da,proto3" json:"da,omitempty"`
	//commit used for the rollapp executable
	Commit string `protobuf:"bytes,4,opt,name=commit,proto3" json:"commit,omitempty"`
}

func (m *RollappConsensusParams) Reset()         { *m = RollappConsensusParams{} }
func (m *RollappConsensusParams) String() string { return proto.CompactTextString(m) }
func (*RollappConsensusParams) ProtoMessage()    {}
func (*RollappConsensusParams) Descriptor() ([]byte, []int) {
	return fileDescriptor_4b679420add07272, []int{1}
}
func (m *RollappConsensusParams) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RollappConsensusParams) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RollappConsensusParams.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *RollappConsensusParams) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RollappConsensusParams.Merge(m, src)
}
func (m *RollappConsensusParams) XXX_Size() int {
	return m.Size()
}
func (m *RollappConsensusParams) XXX_DiscardUnknown() {
	xxx_messageInfo_RollappConsensusParams.DiscardUnknown(m)
}

var xxx_messageInfo_RollappConsensusParams proto.InternalMessageInfo

func (m *RollappConsensusParams) GetBlockmaxgas() int64 {
	if m != nil {
		return m.Blockmaxgas
	}
	return 0
}

func (m *RollappConsensusParams) GetBlockmaxsize() int64 {
	if m != nil {
		return m.Blockmaxsize
	}
	return 0
}

func (m *RollappConsensusParams) GetDa() string {
	if m != nil {
		return m.Da
	}
	return ""
}

func (m *RollappConsensusParams) GetCommit() string {
	if m != nil {
		return m.Commit
	}
	return ""
}

func init() {
	proto.RegisterType((*State)(nil), "dymint.State")
	proto.RegisterType((*RollappConsensusParams)(nil), "dymint.RollappConsensusParams")
}

func init() { proto.RegisterFile("types/dymint/state.proto", fileDescriptor_4b679420add07272) }

var fileDescriptor_4b679420add07272 = []byte{
	// 729 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x94, 0xcd, 0x6e, 0xda, 0x4a,
	0x14, 0xc7, 0x31, 0x10, 0x30, 0x03, 0x04, 0x32, 0xe4, 0x46, 0x4e, 0xae, 0x64, 0xb8, 0xdc, 0x7b,
	0x2b, 0xd4, 0x85, 0x91, 0x92, 0x7d, 0x2b, 0x39, 0x59, 0x04, 0x14, 0xb5, 0x95, 0xa9, 0xb2, 0xe8,
	0x06, 0x8d, 0xed, 0xa9, 0x3d, 0xaa, 0xbf, 0xca, 0x0c, 0x51, 0x92, 0x07, 0xe8, 0x3a, 0xef, 0xd1,
	0x17, 0xc9, 0x32, 0xcb, 0xae, 0xd2, 0x8a, 0xbc, 0x48, 0x35, 0x1f, 0x26, 0x26, 0x69, 0x56, 0x30,
	0xff, 0xf3, 0x9b, 0xbf, 0xcf, 0x9c, 0x73, 0x66, 0x80, 0xc1, 0xae, 0x32, 0x4c, 0xc7, 0xfe, 0x55,
	0x4c, 0x12, 0x36, 0xa6, 0x0c, 0x31, 0x6c, 0x65, 0x8b, 0x94, 0xa5, 0xb0, 0x26, 0xb5, 0x83, 0xdd,
	0x20, 0x0d, 0x52, 0x21, 0x8d, 0xf9, 0x3f, 0x19, 0x3d, 0xe8, 0x07, 0x69, 0x1a, 0x44, 0x78, 0x2c,
	0x56, 0xee, 0xf2, 0xf3, 0x98, 0x91, 0x18, 0x53, 0x86, 0xe2, 0x4c, 0x01, 0xff, 0x48, 0x63, 0x86,
	0x13, 0x1f, 0x2f, 0x84, 0x39, 0x72, 0x3d, 0x32, 0x16, 0xaa, 0x42, 0x86, 0xcf, 0x10, 0x25, 0x14,
	0x98, 0x57, 0x2f, 0x30, 0x17, 0x28, 0x22, 0x3e, 0x62, 0xe9, 0x42, 0x71, 0xff, 0xbe, 0xc0, 0x65,
	0x68, 0x81, 0xe2, 0x97, 0x3f, 0x28, 0x0e, 0xbc, 0xf1, 0xc1, 0xfd, 0x8d, 0x82, 0xc8, 0x1f, 0x19,
	0x1a, 0x7e, 0xaf, 0x83, 0xad, 0x19, 0xdf, 0x00, 0x8f, 0x40, 0xfd, 0x02, 0x2f, 0x28, 0x49, 0x13,
	0x43, 0x1b, 0x68, 0xa3, 0xe6, 0xe1, 0xbe, 0xf5, 0x68, 0x6a, 0xc9, 0x2a, 0x9e, 0x4b, 0xc0, 0xc9,
	0x49, 0xb8, 0x0f, 0x74, 0x2f, 0x44, 0x24, 0x99, 0x13, 0xdf, 0x28, 0x0f, 0xb4, 0x51, 0xc3, 0xa9,
	0x8b, 0xf5, 0xc4, 0x87, 0xff, 0x83, 0x6d, 0x92, 0x10, 0x46, 0x50, 0x34, 0x0f, 0x31, 0x09, 0x42,
	0x66, 0x54, 0x06, 0xda, 0xa8, 0xe2, 0xb4, 0x95, 0x7a, 0x2a, 0x44, 0xf8, 0x1a, 0xec, 0x44, 0x88,
	0xb2, 0xb9, 0x1b, 0xa5, 0xde, 0x97, 0x9c, 0xac, 0x0a, 0xb2, 0xc3, 0x03, 0x36, 0xd7, 0x15, 0xeb,
	0x80, 0x76, 0x81, 0x25, 0xbe, 0xb1, 0xf5, 0x3c, 0x51, 0x79, 0x6e, 0xb1, 0x6b, 0x72, 0x62, 0xf7,
	0x6e, 0xef, 0xfb, 0xa5, 0xd5, 0x7d, 0xbf, 0x79, 0x96, 0x5b, 0x4d, 0x4e, 0x9c, 0xe6, 0xda, 0x77,
	0xe2, 0xc3, 0x33, 0xd0, 0x29, 0x78, 0xf2, 0x8e, 0x1b, 0x35, 0xe1, 0x7a, 0x60, 0xc9, 0x71, 0xb0,
	0xf2, 0x71, 0xb0, 0x3e, 0xe6, 0xe3, 0x60, 0xeb, 0xdc, 0xf6, 0xe6, 0x67, 0x5f, 0x73, 0xda, 0x6b,
	0x2f, 0x1e, 0x85, 0x36, 0x00, 0xeb, 0x2e, 0x52, 0xa3, 0x21, 0x8c, 0xcc, 0xe7, 0xe9, 0x9d, 0xe7,
	0xcc, 0x0c, 0x33, 0xbb, 0x6c, 0x68, 0x4e, 0x61, 0x17, 0x3c, 0x06, 0xa6, 0xc8, 0x48, 0xd6, 0x62,
	0xfe, 0x18, 0x99, 0x7b, 0x21, 0x4a, 0x02, 0xec, 0x1b, 0x4d, 0x51, 0x9e, 0xbf, 0x39, 0x25, 0x2b,
	0xb3, 0xf6, 0xa3, 0xc7, 0x12, 0x81, 0xef, 0xc0, 0x7f, 0x45, 0x13, 0x2f, 0x4d, 0x28, 0x4e, 0xe8,
	0x92, 0xce, 0xe5, 0xf0, 0xac, 0xad, 0xda, 0xc2, 0x6a, 0xf0, 0x68, 0x75, 0x9c, 0x93, 0x1f, 0x04,
	0x98, 0xfb, 0xe5, 0x6d, 0x5a, 0x60, 0xba, 0x8c, 0x18, 0x9d, 0x87, 0x88, 0x86, 0xc6, 0xf6, 0x40,
	0x1b, 0xb5, 0x64, 0x9b, 0x1c, 0xa9, 0x9f, 0x22, 0x1a, 0xf2, 0xa1, 0x40, 0x59, 0x26, 0x91, 0x8e,
	0x40, 0xea, 0x28, 0xcb, 0x44, 0xe8, 0xad, 0xb2, 0xa1, 0x2c, 0x5d, 0xe0, 0xbc, 0xdb, 0xdd, 0x81,
	0x36, 0xaa, 0xda, 0xbd, 0xd5, 0x7d, 0xbf, 0xc3, 0xdb, 0x34, 0xe3, 0x31, 0x99, 0x8c, 0xf4, 0x2e,
	0x08, 0xb0, 0x0f, 0x9a, 0x2e, 0xa2, 0xeb, 0xad, 0x3b, 0x7c, 0xab, 0x03, 0xb8, 0xa4, 0x80, 0x37,
	0xa0, 0x45, 0xf1, 0xd7, 0x25, 0x4e, 0x3c, 0xcc, 0xab, 0x6b, 0x40, 0xd1, 0x83, 0x5d, 0x4b, 0x4d,
	0xfd, 0xac, 0x10, 0xb3, 0xab, 0xbc, 0x8d, 0xce, 0x06, 0x0f, 0xdf, 0x83, 0xee, 0xd3, 0x62, 0x19,
	0x3d, 0xd5, 0x47, 0xe5, 0xe1, 0xa4, 0x51, 0x84, 0xb2, 0xec, 0x49, 0xa5, 0x94, 0x5b, 0xc7, 0xdb,
	0x94, 0xe1, 0x21, 0xf8, 0x4b, 0x1e, 0x79, 0xe9, 0xc6, 0x84, 0x31, 0xec, 0xe7, 0xb9, 0xef, 0x8a,
	0xdc, 0x7b, 0xe2, 0x84, 0x79, 0x4c, 0x1e, 0x62, 0x5a, 0xd5, 0xeb, 0x5d, 0x7d, 0x5a, 0xd5, 0xf5,
	0x6e, 0x63, 0x5a, 0xd5, 0x41, 0xb7, 0x39, 0xad, 0xea, 0xad, 0x6e, 0x7b, 0xf8, 0x4d, 0x03, 0x7b,
	0x7f, 0xfe, 0x3a, 0x1c, 0x80, 0xa6, 0x18, 0xe1, 0x18, 0x5d, 0x06, 0x88, 0x8a, 0x2b, 0x5c, 0x71,
	0x8a, 0x12, 0x1c, 0x82, 0x56, 0xbe, 0xa4, 0xe4, 0x1a, 0x8b, 0xfb, 0x5a, 0x71, 0x36, 0x34, 0xb8,
	0x0d, 0xca, 0x3e, 0x12, 0x17, 0xb5, 0xe1, 0x94, 0x7d, 0x04, 0xf7, 0x40, 0xcd, 0x4b, 0xe3, 0x98,
	0xc8, 0x2b, 0xd9, 0x70, 0xd4, 0xca, 0x3e, 0xbd, 0x5d, 0x99, 0xda, 0xdd, 0xca, 0xd4, 0x7e, 0xad,
	0x4c, 0xed, 0xe6, 0xc1, 0x2c, 0xdd, 0x3d, 0x98, 0xa5, 0x1f, 0x0f, 0x66, 0xe9, 0x93, 0x15, 0x10,
	0x16, 0x2e, 0x5d, 0xcb, 0x4b, 0x63, 0xfe, 0xd2, 0xe0, 0x84, 0xbf, 0x13, 0x97, 0x57, 0xd7, 0xf9,
	0xeb, 0xa3, 0x9e, 0x30, 0x57, 0xad, 0xdd, 0x9a, 0xb8, 0x5e, 0x47, 0xbf, 0x03, 0x00, 0x00, 0xff,
	0xff, 0x89, 0x09, 0x4d, 0x10, 0xb5, 0x05, 0x00, 0x00,
}

func (m *State) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *State) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *State) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.LastSubmittedHeight != 0 {
		i = encodeVarintState(dAtA, i, uint64(m.LastSubmittedHeight))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xa0
	}
	{
		size, err := m.ConsensusParams.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintState(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x9a
	{
		size, err := m.SequencerSet.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintState(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1
	i--
	dAtA[i] = 0x92
	if m.BaseHeight != 0 {
		i = encodeVarintState(dAtA, i, uint64(m.BaseHeight))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x88
	}
	if m.LastStoreHeight != 0 {
		i = encodeVarintState(dAtA, i, uint64(m.LastStoreHeight))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x80
	}
	if len(m.AppHash) > 0 {
		i -= len(m.AppHash)
		copy(dAtA[i:], m.AppHash)
		i = encodeVarintState(dAtA, i, uint64(len(m.AppHash)))
		i--
		dAtA[i] = 0x7a
	}
	if len(m.LastResultsHash) > 0 {
		i -= len(m.LastResultsHash)
		copy(dAtA[i:], m.LastResultsHash)
		i = encodeVarintState(dAtA, i, uint64(len(m.LastResultsHash)))
		i--
		dAtA[i] = 0x72
	}
	if m.LastHeightConsensusParamsChanged != 0 {
		i = encodeVarintState(dAtA, i, uint64(m.LastHeightConsensusParamsChanged))
		i--
		dAtA[i] = 0x68
	}
	if m.LastHeightValidatorsChanged != 0 {
		i = encodeVarintState(dAtA, i, uint64(m.LastHeightValidatorsChanged))
		i--
		dAtA[i] = 0x58
	}
	if m.Validators != nil {
		{
			size, err := m.Validators.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintState(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x4a
	}
	n4, err4 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.LastBlockTime, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.LastBlockTime):])
	if err4 != nil {
		return 0, err4
	}
	i -= n4
	i = encodeVarintState(dAtA, i, uint64(n4))
	i--
	dAtA[i] = 0x32
	{
		size, err := m.LastBlockID.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintState(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x2a
	if m.LastBlockHeight != 0 {
		i = encodeVarintState(dAtA, i, uint64(m.LastBlockHeight))
		i--
		dAtA[i] = 0x20
	}
	if m.InitialHeight != 0 {
		i = encodeVarintState(dAtA, i, uint64(m.InitialHeight))
		i--
		dAtA[i] = 0x18
	}
	if len(m.ChainId) > 0 {
		i -= len(m.ChainId)
		copy(dAtA[i:], m.ChainId)
		i = encodeVarintState(dAtA, i, uint64(len(m.ChainId)))
		i--
		dAtA[i] = 0x12
	}
	if m.Version != nil {
		{
			size, err := m.Version.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintState(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *RollappConsensusParams) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RollappConsensusParams) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *RollappConsensusParams) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Commit) > 0 {
		i -= len(m.Commit)
		copy(dAtA[i:], m.Commit)
		i = encodeVarintState(dAtA, i, uint64(len(m.Commit)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Da) > 0 {
		i -= len(m.Da)
		copy(dAtA[i:], m.Da)
		i = encodeVarintState(dAtA, i, uint64(len(m.Da)))
		i--
		dAtA[i] = 0x1a
	}
	if m.Blockmaxsize != 0 {
		i = encodeVarintState(dAtA, i, uint64(m.Blockmaxsize))
		i--
		dAtA[i] = 0x10
	}
	if m.Blockmaxgas != 0 {
		i = encodeVarintState(dAtA, i, uint64(m.Blockmaxgas))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintState(dAtA []byte, offset int, v uint64) int {
	offset -= sovState(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *State) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Version != nil {
		l = m.Version.Size()
		n += 1 + l + sovState(uint64(l))
	}
	l = len(m.ChainId)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	if m.InitialHeight != 0 {
		n += 1 + sovState(uint64(m.InitialHeight))
	}
	if m.LastBlockHeight != 0 {
		n += 1 + sovState(uint64(m.LastBlockHeight))
	}
	l = m.LastBlockID.Size()
	n += 1 + l + sovState(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.LastBlockTime)
	n += 1 + l + sovState(uint64(l))
	if m.Validators != nil {
		l = m.Validators.Size()
		n += 1 + l + sovState(uint64(l))
	}
	if m.LastHeightValidatorsChanged != 0 {
		n += 1 + sovState(uint64(m.LastHeightValidatorsChanged))
	}
	if m.LastHeightConsensusParamsChanged != 0 {
		n += 1 + sovState(uint64(m.LastHeightConsensusParamsChanged))
	}
	l = len(m.LastResultsHash)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	l = len(m.AppHash)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	if m.LastStoreHeight != 0 {
		n += 2 + sovState(uint64(m.LastStoreHeight))
	}
	if m.BaseHeight != 0 {
		n += 2 + sovState(uint64(m.BaseHeight))
	}
	l = m.SequencerSet.Size()
	n += 2 + l + sovState(uint64(l))
	l = m.ConsensusParams.Size()
	n += 2 + l + sovState(uint64(l))
	if m.LastSubmittedHeight != 0 {
		n += 2 + sovState(uint64(m.LastSubmittedHeight))
	}
	return n
}

func (m *RollappConsensusParams) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Blockmaxgas != 0 {
		n += 1 + sovState(uint64(m.Blockmaxgas))
	}
	if m.Blockmaxsize != 0 {
		n += 1 + sovState(uint64(m.Blockmaxsize))
	}
	l = len(m.Da)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	l = len(m.Commit)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	return n
}

func sovState(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozState(x uint64) (n int) {
	return sovState(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *State) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowState
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: State: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: State: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Version == nil {
				m.Version = &state.Version{}
			}
			if err := m.Version.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChainId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ChainId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field InitialHeight", wireType)
			}
			m.InitialHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.InitialHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastBlockHeight", wireType)
			}
			m.LastBlockHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LastBlockHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastBlockID", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.LastBlockID.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastBlockTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.LastBlockTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Validators", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Validators == nil {
				m.Validators = &types.ValidatorSet{}
			}
			if err := m.Validators.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 11:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastHeightValidatorsChanged", wireType)
			}
			m.LastHeightValidatorsChanged = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LastHeightValidatorsChanged |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 13:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastHeightConsensusParamsChanged", wireType)
			}
			m.LastHeightConsensusParamsChanged = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LastHeightConsensusParamsChanged |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 14:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastResultsHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LastResultsHash = append(m.LastResultsHash[:0], dAtA[iNdEx:postIndex]...)
			if m.LastResultsHash == nil {
				m.LastResultsHash = []byte{}
			}
			iNdEx = postIndex
		case 15:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AppHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AppHash = append(m.AppHash[:0], dAtA[iNdEx:postIndex]...)
			if m.AppHash == nil {
				m.AppHash = []byte{}
			}
			iNdEx = postIndex
		case 16:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastStoreHeight", wireType)
			}
			m.LastStoreHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LastStoreHeight |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 17:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field BaseHeight", wireType)
			}
			m.BaseHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.BaseHeight |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 18:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SequencerSet", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.SequencerSet.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 19:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ConsensusParams", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ConsensusParams.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 20:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastSubmittedHeight", wireType)
			}
			m.LastSubmittedHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LastSubmittedHeight |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipState(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthState
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *RollappConsensusParams) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowState
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RollappConsensusParams: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RollappConsensusParams: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Blockmaxgas", wireType)
			}
			m.Blockmaxgas = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Blockmaxgas |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Blockmaxsize", wireType)
			}
			m.Blockmaxsize = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Blockmaxsize |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Da", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Da = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Commit", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Commit = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipState(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthState
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipState(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowState
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowState
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowState
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthState
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupState
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthState
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthState        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowState          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupState = fmt.Errorf("proto: unexpected end of group")
)
