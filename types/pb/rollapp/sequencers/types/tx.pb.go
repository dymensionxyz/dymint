// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: types/rollapp/sequencers/types/tx.proto

package sequencers

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types/msgservice"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	types "github.com/gogo/protobuf/types"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type MsgCreateSequencer struct {
	// Operator is the bech32-encoded address of the actor sending the update - must be val addr
	Operator string `protobuf:"bytes,1,opt,name=operator,proto3" json:"operator,omitempty"`
	// PubKey is a tendermint consensus pub key
	PubKey *types.Any `protobuf:"bytes,2,opt,name=pub_key,json=pubKey,proto3" json:"pub_key,omitempty"`
	// Signature is operator signed with the private key of pub_key
	Signature []byte `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (m *MsgCreateSequencer) Reset()         { *m = MsgCreateSequencer{} }
func (m *MsgCreateSequencer) String() string { return proto.CompactTextString(m) }
func (*MsgCreateSequencer) ProtoMessage()    {}
func (*MsgCreateSequencer) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb27942d1c1a6ff0, []int{0}
}
func (m *MsgCreateSequencer) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgCreateSequencer) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgCreateSequencer.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgCreateSequencer) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgCreateSequencer.Merge(m, src)
}
func (m *MsgCreateSequencer) XXX_Size() int {
	return m.Size()
}
func (m *MsgCreateSequencer) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgCreateSequencer.DiscardUnknown(m)
}

var xxx_messageInfo_MsgCreateSequencer proto.InternalMessageInfo

func (m *MsgCreateSequencer) GetOperator() string {
	if m != nil {
		return m.Operator
	}
	return ""
}

func (m *MsgCreateSequencer) GetPubKey() *types.Any {
	if m != nil {
		return m.PubKey
	}
	return nil
}

func (m *MsgCreateSequencer) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type MsgCreateSequencerResponse struct {
}

func (m *MsgCreateSequencerResponse) Reset()         { *m = MsgCreateSequencerResponse{} }
func (m *MsgCreateSequencerResponse) String() string { return proto.CompactTextString(m) }
func (*MsgCreateSequencerResponse) ProtoMessage()    {}
func (*MsgCreateSequencerResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb27942d1c1a6ff0, []int{1}
}
func (m *MsgCreateSequencerResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgCreateSequencerResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgCreateSequencerResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgCreateSequencerResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgCreateSequencerResponse.Merge(m, src)
}
func (m *MsgCreateSequencerResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgCreateSequencerResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgCreateSequencerResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgCreateSequencerResponse proto.InternalMessageInfo

type MsgUpdateSequencer struct {
	// Operator is the bech32-encoded address of the actor sending the update - must be val addr
	Operator string `protobuf:"bytes,1,opt,name=operator,proto3" json:"operator,omitempty"`
	// RewardAddr is a bech32 encoded sdk acc address
	RewardAddr string `protobuf:"bytes,3,opt,name=reward_addr,json=rewardAddr,proto3" json:"reward_addr,omitempty"`
}

func (m *MsgUpdateSequencer) Reset()         { *m = MsgUpdateSequencer{} }
func (m *MsgUpdateSequencer) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateSequencer) ProtoMessage()    {}
func (*MsgUpdateSequencer) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb27942d1c1a6ff0, []int{2}
}
func (m *MsgUpdateSequencer) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateSequencer) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgUpdateSequencer.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgUpdateSequencer) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgUpdateSequencer.Merge(m, src)
}
func (m *MsgUpdateSequencer) XXX_Size() int {
	return m.Size()
}
func (m *MsgUpdateSequencer) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgUpdateSequencer.DiscardUnknown(m)
}

var xxx_messageInfo_MsgUpdateSequencer proto.InternalMessageInfo

func (m *MsgUpdateSequencer) GetOperator() string {
	if m != nil {
		return m.Operator
	}
	return ""
}

func (m *MsgUpdateSequencer) GetRewardAddr() string {
	if m != nil {
		return m.RewardAddr
	}
	return ""
}

type MsgUpdateSequencerResponse struct {
}

func (m *MsgUpdateSequencerResponse) Reset()         { *m = MsgUpdateSequencerResponse{} }
func (m *MsgUpdateSequencerResponse) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateSequencerResponse) ProtoMessage()    {}
func (*MsgUpdateSequencerResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb27942d1c1a6ff0, []int{3}
}
func (m *MsgUpdateSequencerResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateSequencerResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgUpdateSequencerResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgUpdateSequencerResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgUpdateSequencerResponse.Merge(m, src)
}
func (m *MsgUpdateSequencerResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgUpdateSequencerResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgUpdateSequencerResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgUpdateSequencerResponse proto.InternalMessageInfo

// ConsensusMsgUpsertSequencer is a consensus message to upsert the sequencer.
type ConsensusMsgUpsertSequencer struct {
	// Operator is the bech32-encoded address of the actor sending the update
	Operator string `protobuf:"bytes,1,opt,name=operator,proto3" json:"operator,omitempty"`
	// ConsPubKey is a tendermint consensus pub key
	ConsPubKey *types.Any `protobuf:"bytes,2,opt,name=cons_pub_key,json=consPubKey,proto3" json:"cons_pub_key,omitempty"`
	// RewardAddr is the bech32-encoded sequencer's reward address
	RewardAddr string `protobuf:"bytes,3,opt,name=reward_addr,json=rewardAddr,proto3" json:"reward_addr,omitempty"`
}

func (m *ConsensusMsgUpsertSequencer) Reset()         { *m = ConsensusMsgUpsertSequencer{} }
func (m *ConsensusMsgUpsertSequencer) String() string { return proto.CompactTextString(m) }
func (*ConsensusMsgUpsertSequencer) ProtoMessage()    {}
func (*ConsensusMsgUpsertSequencer) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb27942d1c1a6ff0, []int{4}
}
func (m *ConsensusMsgUpsertSequencer) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ConsensusMsgUpsertSequencer) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ConsensusMsgUpsertSequencer.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ConsensusMsgUpsertSequencer) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ConsensusMsgUpsertSequencer.Merge(m, src)
}
func (m *ConsensusMsgUpsertSequencer) XXX_Size() int {
	return m.Size()
}
func (m *ConsensusMsgUpsertSequencer) XXX_DiscardUnknown() {
	xxx_messageInfo_ConsensusMsgUpsertSequencer.DiscardUnknown(m)
}

var xxx_messageInfo_ConsensusMsgUpsertSequencer proto.InternalMessageInfo

func (m *ConsensusMsgUpsertSequencer) GetOperator() string {
	if m != nil {
		return m.Operator
	}
	return ""
}

func (m *ConsensusMsgUpsertSequencer) GetConsPubKey() *types.Any {
	if m != nil {
		return m.ConsPubKey
	}
	return nil
}

func (m *ConsensusMsgUpsertSequencer) GetRewardAddr() string {
	if m != nil {
		return m.RewardAddr
	}
	return ""
}

type ConsensusMsgUpsertSequencerResponse struct {
}

func (m *ConsensusMsgUpsertSequencerResponse) Reset()         { *m = ConsensusMsgUpsertSequencerResponse{} }
func (m *ConsensusMsgUpsertSequencerResponse) String() string { return proto.CompactTextString(m) }
func (*ConsensusMsgUpsertSequencerResponse) ProtoMessage()    {}
func (*ConsensusMsgUpsertSequencerResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb27942d1c1a6ff0, []int{5}
}
func (m *ConsensusMsgUpsertSequencerResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ConsensusMsgUpsertSequencerResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ConsensusMsgUpsertSequencerResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ConsensusMsgUpsertSequencerResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ConsensusMsgUpsertSequencerResponse.Merge(m, src)
}
func (m *ConsensusMsgUpsertSequencerResponse) XXX_Size() int {
	return m.Size()
}
func (m *ConsensusMsgUpsertSequencerResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ConsensusMsgUpsertSequencerResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ConsensusMsgUpsertSequencerResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MsgCreateSequencer)(nil), "rollapp.sequencers.types.MsgCreateSequencer")
	proto.RegisterType((*MsgCreateSequencerResponse)(nil), "rollapp.sequencers.types.MsgCreateSequencerResponse")
	proto.RegisterType((*MsgUpdateSequencer)(nil), "rollapp.sequencers.types.MsgUpdateSequencer")
	proto.RegisterType((*MsgUpdateSequencerResponse)(nil), "rollapp.sequencers.types.MsgUpdateSequencerResponse")
	proto.RegisterType((*ConsensusMsgUpsertSequencer)(nil), "rollapp.sequencers.types.ConsensusMsgUpsertSequencer")
	proto.RegisterType((*ConsensusMsgUpsertSequencerResponse)(nil), "rollapp.sequencers.types.ConsensusMsgUpsertSequencerResponse")
}

func init() {
	proto.RegisterFile("types/rollapp/sequencers/types/tx.proto", fileDescriptor_bb27942d1c1a6ff0)
}

var fileDescriptor_bb27942d1c1a6ff0 = []byte{
	// 406 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0xbd, 0x8e, 0xd3, 0x40,
	0x10, 0xc7, 0xb3, 0x09, 0x0a, 0xc9, 0x26, 0x48, 0xc8, 0x4a, 0x61, 0x4c, 0x30, 0x91, 0x11, 0x22,
	0x42, 0xc2, 0x2b, 0x40, 0xa2, 0x48, 0x17, 0xd2, 0x81, 0x90, 0x90, 0x81, 0x86, 0x26, 0xf2, 0xc7,
	0x64, 0xb1, 0x88, 0x77, 0x97, 0xdd, 0x35, 0xc4, 0x94, 0x57, 0x5f, 0x71, 0xaf, 0x70, 0x6f, 0x70,
	0x8f, 0x71, 0x65, 0xca, 0x2b, 0x4f, 0x49, 0x71, 0xaf, 0x71, 0xf2, 0x47, 0x12, 0x29, 0x77, 0x3a,
	0xe5, 0x2a, 0xef, 0xcc, 0xfc, 0x3d, 0xff, 0xdf, 0x8c, 0x06, 0xbf, 0xd2, 0x99, 0x00, 0x45, 0x24,
	0x9f, 0xcf, 0x7d, 0x21, 0x88, 0x82, 0x3f, 0x29, 0xb0, 0x10, 0xa4, 0x22, 0x65, 0x41, 0x2f, 0x5c,
	0x21, 0xb9, 0xe6, 0x86, 0x59, 0x49, 0xdc, 0x9d, 0xc4, 0x2d, 0x24, 0x56, 0x8f, 0x72, 0xca, 0x0b,
	0x11, 0xc9, 0x5f, 0xa5, 0xde, 0x7a, 0x56, 0xfe, 0x1f, 0x72, 0x95, 0x70, 0x45, 0x12, 0x45, 0xc9,
	0xdf, 0xb7, 0xf9, 0xa7, 0x2a, 0x3f, 0xa1, 0x9c, 0xd3, 0x39, 0x90, 0x22, 0x0a, 0xd2, 0x19, 0xf1,
	0x59, 0x56, 0x96, 0x9c, 0x63, 0x84, 0x8d, 0x2f, 0x8a, 0x4e, 0x24, 0xf8, 0x1a, 0xbe, 0x6d, 0xdc,
	0x0c, 0x0b, 0xb7, 0xb8, 0x00, 0xe9, 0x6b, 0x2e, 0x4d, 0x34, 0x40, 0xc3, 0xb6, 0xb7, 0x8d, 0x8d,
	0x37, 0xf8, 0xa1, 0x48, 0x83, 0xe9, 0x6f, 0xc8, 0xcc, 0xfa, 0x00, 0x0d, 0x3b, 0xef, 0x7a, 0x6e,
	0xd9, 0xdf, 0xdd, 0xf4, 0x77, 0xc7, 0x2c, 0xf3, 0x9a, 0x22, 0x0d, 0x3e, 0x43, 0x66, 0xf4, 0x71,
	0x5b, 0xc5, 0x94, 0xf9, 0x3a, 0x95, 0x60, 0x36, 0x06, 0x68, 0xd8, 0xf5, 0x76, 0x89, 0xd1, 0xa3,
	0xa3, 0xab, 0xb3, 0xd7, 0xdb, 0xde, 0x4e, 0x1f, 0x5b, 0x37, 0x69, 0x3c, 0x50, 0x82, 0x33, 0x05,
	0xce, 0xac, 0x60, 0xfd, 0x21, 0xa2, 0x83, 0x59, 0x9f, 0xe3, 0x8e, 0x84, 0x7f, 0xbe, 0x8c, 0xa6,
	0x7e, 0x14, 0xc9, 0xc2, 0xbe, 0xed, 0xe1, 0x32, 0x35, 0x8e, 0x22, 0xb9, 0xe7, 0xff, 0xe9, 0x41,
	0xab, 0xfe, 0xb8, 0x51, 0x51, 0xec, 0xf9, 0x6c, 0x29, 0x4e, 0x11, 0x7e, 0x3a, 0xc9, 0x5f, 0x4c,
	0xa5, 0xaa, 0xd0, 0x29, 0x90, 0xfa, 0x30, 0x9e, 0x0f, 0xb8, 0x1b, 0x72, 0xa6, 0xa6, 0x87, 0x2c,
	0x10, 0xe7, 0xca, 0xaf, 0xe5, 0x12, 0xef, 0x39, 0x87, 0xf3, 0x12, 0xbf, 0xb8, 0x03, 0x71, 0x33,
	0xca, 0xc7, 0xef, 0xe7, 0x2b, 0x1b, 0x2d, 0x57, 0x36, 0xba, 0x5c, 0xd9, 0xe8, 0x64, 0x6d, 0xd7,
	0x96, 0x6b, 0xbb, 0x76, 0xb1, 0xb6, 0x6b, 0x3f, 0x47, 0x34, 0xd6, 0xbf, 0xd2, 0xc0, 0x0d, 0x79,
	0x42, 0xa2, 0x2c, 0x01, 0xa6, 0x62, 0xce, 0x16, 0xd9, 0xff, 0x3c, 0x88, 0x99, 0xae, 0x0e, 0x56,
	0x04, 0xb7, 0x1c, 0x73, 0xd0, 0x2c, 0xc6, 0x78, 0x7f, 0x1d, 0x00, 0x00, 0xff, 0xff, 0x38, 0x7c,
	0xd4, 0x90, 0xef, 0x02, 0x00, 0x00,
}

func (m *MsgCreateSequencer) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgCreateSequencer) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgCreateSequencer) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Signature) > 0 {
		i -= len(m.Signature)
		copy(dAtA[i:], m.Signature)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Signature)))
		i--
		dAtA[i] = 0x1a
	}
	if m.PubKey != nil {
		{
			size, err := m.PubKey.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTx(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if len(m.Operator) > 0 {
		i -= len(m.Operator)
		copy(dAtA[i:], m.Operator)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Operator)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgCreateSequencerResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgCreateSequencerResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgCreateSequencerResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgUpdateSequencer) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateSequencer) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateSequencer) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.RewardAddr) > 0 {
		i -= len(m.RewardAddr)
		copy(dAtA[i:], m.RewardAddr)
		i = encodeVarintTx(dAtA, i, uint64(len(m.RewardAddr)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Operator) > 0 {
		i -= len(m.Operator)
		copy(dAtA[i:], m.Operator)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Operator)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgUpdateSequencerResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateSequencerResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateSequencerResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *ConsensusMsgUpsertSequencer) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ConsensusMsgUpsertSequencer) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ConsensusMsgUpsertSequencer) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.RewardAddr) > 0 {
		i -= len(m.RewardAddr)
		copy(dAtA[i:], m.RewardAddr)
		i = encodeVarintTx(dAtA, i, uint64(len(m.RewardAddr)))
		i--
		dAtA[i] = 0x1a
	}
	if m.ConsPubKey != nil {
		{
			size, err := m.ConsPubKey.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTx(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if len(m.Operator) > 0 {
		i -= len(m.Operator)
		copy(dAtA[i:], m.Operator)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Operator)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ConsensusMsgUpsertSequencerResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ConsensusMsgUpsertSequencerResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ConsensusMsgUpsertSequencerResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgCreateSequencer) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Operator)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	if m.PubKey != nil {
		l = m.PubKey.Size()
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Signature)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *MsgCreateSequencerResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgUpdateSequencer) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Operator)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.RewardAddr)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *MsgUpdateSequencerResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *ConsensusMsgUpsertSequencer) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Operator)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	if m.ConsPubKey != nil {
		l = m.ConsPubKey.Size()
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.RewardAddr)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *ConsensusMsgUpsertSequencerResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgCreateSequencer) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgCreateSequencer: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgCreateSequencer: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Operator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Operator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PubKey", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.PubKey == nil {
				m.PubKey = &types.Any{}
			}
			if err := m.PubKey.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signature", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signature = append(m.Signature[:0], dAtA[iNdEx:postIndex]...)
			if m.Signature == nil {
				m.Signature = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *MsgCreateSequencerResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgCreateSequencerResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgCreateSequencerResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *MsgUpdateSequencer) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgUpdateSequencer: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgUpdateSequencer: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Operator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Operator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RewardAddr", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RewardAddr = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *MsgUpdateSequencerResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgUpdateSequencerResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgUpdateSequencerResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *ConsensusMsgUpsertSequencer) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: ConsensusMsgUpsertSequencer: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ConsensusMsgUpsertSequencer: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Operator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Operator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ConsPubKey", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.ConsPubKey == nil {
				m.ConsPubKey = &types.Any{}
			}
			if err := m.ConsPubKey.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RewardAddr", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RewardAddr = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *ConsensusMsgUpsertSequencerResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: ConsensusMsgUpsertSequencerResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ConsensusMsgUpsertSequencerResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
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
					return 0, ErrIntOverflowTx
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
					return 0, ErrIntOverflowTx
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
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)
