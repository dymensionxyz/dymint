// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: dymension/sequencer/sequencer.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	types "github.com/cosmos/cosmos-sdk/codec/types"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	types1 "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/cosmos-sdk/types/msgservice"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/gogo/protobuf/types"
	github_com_gogo_protobuf_types "github.com/gogo/protobuf/types"
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

// Sequencer defines a sequencer identified by its' address (sequencerAddress).
// The sequencer could be attached to only one rollapp (rollappId).
type Sequencer struct {
	// address is the bech32-encoded address of the sequencer account which is the account that the message was sent from.
	Address string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	// pubkey is the public key of the sequencers' dymint client, as a Protobuf Any.
	DymintPubKey *types.Any `protobuf:"bytes,2,opt,name=dymintPubKey,proto3" json:"dymintPubKey,omitempty"`
	// rollappId defines the rollapp to which the sequencer belongs.
	RollappId string `protobuf:"bytes,3,opt,name=rollappId,proto3" json:"rollappId,omitempty"`
	// metadata defines the extra information for the sequencer.
	Metadata SequencerMetadata `protobuf:"bytes,4,opt,name=metadata,proto3" json:"metadata"`
	// jailed defined whether the sequencer has been jailed from bonded status or not.
	Jailed bool `protobuf:"varint,5,opt,name=jailed,proto3" json:"jailed,omitempty"`
	// status is the sequencer status (bonded/unbonding/unbonded).
	Status OperatingStatus `protobuf:"varint,7,opt,name=status,proto3,enum=dymensionxyz.dymension.sequencer.OperatingStatus" json:"status,omitempty"`
	// tokens define the delegated tokens (incl. self-delegation).
	Tokens github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,8,rep,name=tokens,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"tokens"`
	// unbond_request_height stores the height at which this sequencer has
	// requested to unbond.
	UnbondRequestHeight int64 `protobuf:"varint,9,opt,name=unbond_request_height,json=unbondRequestHeight,proto3" json:"unbond_request_height,omitempty"`
	// unbond_time defines, if unbonding, the min time for the sequencer to
	// complete unbonding.
	UnbondTime time.Time `protobuf:"bytes,10,opt,name=unbond_time,json=unbondTime,proto3,stdtime" json:"unbond_time"`
	// notice_period_time defines the time when the sequencer will finish it's notice period if started
	NoticePeriodTime time.Time `protobuf:"bytes,11,opt,name=notice_period_time,json=noticePeriodTime,proto3,stdtime" json:"notice_period_time"`
}

func (m *Sequencer) Reset()         { *m = Sequencer{} }
func (m *Sequencer) String() string { return proto.CompactTextString(m) }
func (*Sequencer) ProtoMessage()    {}
func (*Sequencer) Descriptor() ([]byte, []int) {
	return fileDescriptor_17d99b644bf09274, []int{0}
}
func (m *Sequencer) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Sequencer) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Sequencer.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Sequencer) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Sequencer.Merge(m, src)
}
func (m *Sequencer) XXX_Size() int {
	return m.Size()
}
func (m *Sequencer) XXX_DiscardUnknown() {
	xxx_messageInfo_Sequencer.DiscardUnknown(m)
}

var xxx_messageInfo_Sequencer proto.InternalMessageInfo

func (m *Sequencer) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *Sequencer) GetDymintPubKey() *types.Any {
	if m != nil {
		return m.DymintPubKey
	}
	return nil
}

func (m *Sequencer) GetRollappId() string {
	if m != nil {
		return m.RollappId
	}
	return ""
}

func (m *Sequencer) GetMetadata() SequencerMetadata {
	if m != nil {
		return m.Metadata
	}
	return SequencerMetadata{}
}

func (m *Sequencer) GetJailed() bool {
	if m != nil {
		return m.Jailed
	}
	return false
}

func (m *Sequencer) GetStatus() OperatingStatus {
	if m != nil {
		return m.Status
	}
	return Unbonded
}

func (m *Sequencer) GetTokens() github_com_cosmos_cosmos_sdk_types.Coins {
	if m != nil {
		return m.Tokens
	}
	return nil
}

func (m *Sequencer) GetUnbondRequestHeight() int64 {
	if m != nil {
		return m.UnbondRequestHeight
	}
	return 0
}

func (m *Sequencer) GetUnbondTime() time.Time {
	if m != nil {
		return m.UnbondTime
	}
	return time.Time{}
}

func (m *Sequencer) GetNoticePeriodTime() time.Time {
	if m != nil {
		return m.NoticePeriodTime
	}
	return time.Time{}
}

// BondReduction defines an object which holds the information about the sequencer and its queued unbonding amount
type BondReduction struct {
	// sequencer_address is the bech32-encoded address of the sequencer account which is the account that the message was sent from.
	SequencerAddress string `protobuf:"bytes,1,opt,name=sequencer_address,json=sequencerAddress,proto3" json:"sequencer_address,omitempty"`
	// decrease_bond_amount is the amount of tokens to be unbonded.
	DecreaseBondAmount types1.Coin `protobuf:"bytes,2,opt,name=decrease_bond_amount,json=decreaseBondAmount,proto3" json:"decrease_bond_amount"`
	// decrease_bond_time defines, if unbonding, the min time for the sequencer to complete unbonding.
	DecreaseBondTime time.Time `protobuf:"bytes,3,opt,name=decrease_bond_time,json=decreaseBondTime,proto3,stdtime" json:"decrease_bond_time"`
}

func (m *BondReduction) Reset()         { *m = BondReduction{} }
func (m *BondReduction) String() string { return proto.CompactTextString(m) }
func (*BondReduction) ProtoMessage()    {}
func (*BondReduction) Descriptor() ([]byte, []int) {
	return fileDescriptor_17d99b644bf09274, []int{1}
}
func (m *BondReduction) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *BondReduction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_BondReduction.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *BondReduction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BondReduction.Merge(m, src)
}
func (m *BondReduction) XXX_Size() int {
	return m.Size()
}
func (m *BondReduction) XXX_DiscardUnknown() {
	xxx_messageInfo_BondReduction.DiscardUnknown(m)
}

var xxx_messageInfo_BondReduction proto.InternalMessageInfo

func (m *BondReduction) GetSequencerAddress() string {
	if m != nil {
		return m.SequencerAddress
	}
	return ""
}

func (m *BondReduction) GetDecreaseBondAmount() types1.Coin {
	if m != nil {
		return m.DecreaseBondAmount
	}
	return types1.Coin{}
}

func (m *BondReduction) GetDecreaseBondTime() time.Time {
	if m != nil {
		return m.DecreaseBondTime
	}
	return time.Time{}
}

func init() {
	proto.RegisterType((*Sequencer)(nil), "dymensionxyz.dymension.sequencer.Sequencer")
	proto.RegisterType((*BondReduction)(nil), "dymensionxyz.dymension.sequencer.BondReduction")
}

func init() {
	proto.RegisterFile("dymension/sequencer/sequencer.proto", fileDescriptor_17d99b644bf09274)
}

var fileDescriptor_17d99b644bf09274 = []byte{
	// 635 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x53, 0xc1, 0x6e, 0xd3, 0x4a,
	0x14, 0x8d, 0x5f, 0xda, 0x34, 0x99, 0x3c, 0x50, 0x19, 0x02, 0xb8, 0x15, 0x72, 0xac, 0xb2, 0xb1,
	0x40, 0x1d, 0x93, 0x54, 0x62, 0xdf, 0x20, 0x24, 0x0a, 0x42, 0x14, 0x17, 0x36, 0x6c, 0xac, 0xb1,
	0x3d, 0xb8, 0xa6, 0xf1, 0x8c, 0xf1, 0x8c, 0xab, 0x9a, 0x5f, 0x60, 0xd3, 0xef, 0x60, 0xcd, 0x47,
	0x54, 0xac, 0xba, 0x64, 0x45, 0x51, 0xf3, 0x23, 0xc8, 0x33, 0x63, 0x37, 0x81, 0x8a, 0xaa, 0x2b,
	0xfb, 0xfa, 0x9e, 0x73, 0xe6, 0x9e, 0x3b, 0xc7, 0xe0, 0x41, 0x54, 0xa6, 0x84, 0xf2, 0x84, 0x51,
	0x97, 0x93, 0x4f, 0x05, 0xa1, 0x21, 0xc9, 0x2f, 0xde, 0x50, 0x96, 0x33, 0xc1, 0xa0, 0xdd, 0x80,
	0x8e, 0xca, 0xcf, 0xa8, 0x29, 0x50, 0x83, 0x5b, 0x5f, 0x0b, 0x19, 0x4f, 0x19, 0xf7, 0x25, 0xde,
	0x55, 0x85, 0x22, 0xaf, 0xaf, 0xc5, 0x8c, 0xc5, 0x53, 0xe2, 0xca, 0x2a, 0x28, 0x3e, 0xb8, 0x98,
	0x96, 0xba, 0x35, 0x88, 0x59, 0xcc, 0x14, 0xa5, 0x7a, 0xd3, 0x5f, 0x87, 0x7f, 0x12, 0x44, 0x92,
	0x12, 0x2e, 0x70, 0x9a, 0x69, 0x80, 0xa5, 0xf4, 0xdd, 0x00, 0x73, 0xe2, 0x1e, 0x8e, 0x02, 0x22,
	0xf0, 0xc8, 0x0d, 0x59, 0x42, 0x75, 0xff, 0x9e, 0xee, 0xa7, 0x3c, 0x76, 0x0f, 0x47, 0xd5, 0x43,
	0x37, 0x36, 0x2e, 0x33, 0x9b, 0x12, 0x81, 0x23, 0x2c, 0xb0, 0xc6, 0x3c, 0xbc, 0x0c, 0xc3, 0x32,
	0x92, 0x63, 0x91, 0xd0, 0xd8, 0xe7, 0x02, 0x8b, 0x42, 0x5b, 0xdb, 0xf8, 0xb2, 0x0c, 0x7a, 0x7b,
	0x35, 0x08, 0x9a, 0x60, 0x05, 0x47, 0x51, 0x4e, 0x38, 0x37, 0x0d, 0xdb, 0x70, 0x7a, 0x5e, 0x5d,
	0x42, 0x0f, 0xfc, 0x1f, 0x95, 0x69, 0x42, 0xc5, 0x6e, 0x11, 0xbc, 0x24, 0xa5, 0xf9, 0x9f, 0x6d,
	0x38, 0xfd, 0xf1, 0x00, 0x29, 0xa3, 0xa8, 0x36, 0x8a, 0xb6, 0x69, 0x39, 0x31, 0xbf, 0x7f, 0xdb,
	0x1c, 0xe8, 0x05, 0x86, 0x79, 0x99, 0x09, 0x86, 0x14, 0xcb, 0x5b, 0xd0, 0x80, 0xf7, 0x41, 0x2f,
	0x67, 0xd3, 0x29, 0xce, 0xb2, 0x9d, 0xc8, 0x6c, 0xcb, 0xf3, 0x2e, 0x3e, 0xc0, 0x77, 0xa0, 0x5b,
	0xfb, 0x32, 0x97, 0xe4, 0x69, 0x5b, 0xe8, 0xaa, 0x4b, 0x44, 0x8d, 0x95, 0x57, 0x9a, 0x3a, 0x59,
	0x3a, 0xf9, 0x39, 0x6c, 0x79, 0x8d, 0x14, 0xbc, 0x0b, 0x3a, 0x1f, 0x71, 0x32, 0x25, 0x91, 0xb9,
	0x6c, 0x1b, 0x4e, 0xd7, 0xd3, 0x15, 0xdc, 0x01, 0x1d, 0xb5, 0x18, 0x73, 0xc5, 0x36, 0x9c, 0x9b,
	0xe3, 0xd1, 0xd5, 0x87, 0xbd, 0xae, 0x57, 0xba, 0x27, 0x89, 0x9e, 0x16, 0x80, 0x21, 0xe8, 0x08,
	0x76, 0x40, 0x28, 0x37, 0xbb, 0x76, 0xdb, 0xe9, 0x8f, 0xd7, 0x90, 0x5e, 0x46, 0x75, 0xdb, 0x48,
	0xdf, 0x36, 0x7a, 0xca, 0x12, 0x3a, 0x79, 0x5c, 0x4d, 0xf7, 0xf5, 0x6c, 0xe8, 0xc4, 0x89, 0xd8,
	0x2f, 0x02, 0x14, 0xb2, 0x54, 0x47, 0x4f, 0x3f, 0x36, 0x79, 0x74, 0xe0, 0x8a, 0x32, 0x23, 0x5c,
	0x12, 0xb8, 0xa7, 0xa5, 0xe1, 0x18, 0xdc, 0x29, 0x68, 0xc0, 0x68, 0xe4, 0xe7, 0xd5, 0x40, 0x5c,
	0xf8, 0xfb, 0x24, 0x89, 0xf7, 0x85, 0xd9, 0xb3, 0x0d, 0xa7, 0xed, 0xdd, 0x56, 0x4d, 0x4f, 0xf5,
	0x9e, 0xcb, 0x16, 0x7c, 0x06, 0xfa, 0x9a, 0x53, 0xe5, 0xd1, 0x04, 0x72, 0xab, 0xeb, 0x7f, 0xdd,
	0xe1, 0xdb, 0x3a, 0xac, 0x93, 0x6e, 0x35, 0xde, 0xf1, 0xd9, 0xd0, 0xf0, 0x80, 0x22, 0x56, 0x2d,
	0xe8, 0x01, 0x48, 0x99, 0x48, 0x42, 0xe2, 0x67, 0x24, 0x4f, 0x98, 0x56, 0xeb, 0x5f, 0x43, 0x6d,
	0x55, 0xf1, 0x77, 0x25, 0xbd, 0x02, 0xbc, 0x58, 0xea, 0x76, 0x56, 0x57, 0x36, 0x66, 0x06, 0xb8,
	0x31, 0x91, 0x63, 0x47, 0x45, 0x28, 0x12, 0x46, 0xe1, 0x23, 0x70, 0xab, 0x59, 0xb8, 0xbf, 0x98,
	0xcd, 0xd5, 0xa6, 0xb1, 0xad, 0x43, 0xfa, 0x06, 0x0c, 0x22, 0x12, 0xe6, 0x04, 0x73, 0xe2, 0x4b,
	0x9b, 0x38, 0x65, 0x05, 0x15, 0x3a, 0xac, 0xff, 0xb8, 0x06, 0x15, 0x12, 0x58, 0x93, 0xab, 0x11,
	0xb6, 0x25, 0xb5, 0xf2, 0xba, 0x28, 0x29, 0xbd, 0xb6, 0xaf, 0xe3, 0x75, 0x5e, 0xb5, 0x02, 0x4c,
	0x76, 0x4f, 0xce, 0x2d, 0xe3, 0xf4, 0xdc, 0x32, 0x7e, 0x9d, 0x5b, 0xc6, 0xf1, 0xcc, 0x6a, 0x9d,
	0xce, 0xac, 0xd6, 0x8f, 0x99, 0xd5, 0x7a, 0xff, 0x64, 0x2e, 0x06, 0xf3, 0xf1, 0xbb, 0x28, 0xdc,
	0xc3, 0x2d, 0xf7, 0x68, 0xee, 0xb7, 0x96, 0xd1, 0x08, 0x3a, 0x72, 0x82, 0xad, 0xdf, 0x01, 0x00,
	0x00, 0xff, 0xff, 0xdd, 0x3f, 0x85, 0xd1, 0x0b, 0x05, 0x00, 0x00,
}

func (m *Sequencer) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Sequencer) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Sequencer) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n1, err1 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.NoticePeriodTime, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.NoticePeriodTime):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintSequencer(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x5a
	n2, err2 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.UnbondTime, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.UnbondTime):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintSequencer(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0x52
	if m.UnbondRequestHeight != 0 {
		i = encodeVarintSequencer(dAtA, i, uint64(m.UnbondRequestHeight))
		i--
		dAtA[i] = 0x48
	}
	if len(m.Tokens) > 0 {
		for iNdEx := len(m.Tokens) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Tokens[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintSequencer(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x42
		}
	}
	if m.Status != 0 {
		i = encodeVarintSequencer(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x38
	}
	if m.Jailed {
		i--
		if m.Jailed {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x28
	}
	{
		size, err := m.Metadata.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintSequencer(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x22
	if len(m.RollappId) > 0 {
		i -= len(m.RollappId)
		copy(dAtA[i:], m.RollappId)
		i = encodeVarintSequencer(dAtA, i, uint64(len(m.RollappId)))
		i--
		dAtA[i] = 0x1a
	}
	if m.DymintPubKey != nil {
		{
			size, err := m.DymintPubKey.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintSequencer(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintSequencer(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *BondReduction) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *BondReduction) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *BondReduction) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n5, err5 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.DecreaseBondTime, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.DecreaseBondTime):])
	if err5 != nil {
		return 0, err5
	}
	i -= n5
	i = encodeVarintSequencer(dAtA, i, uint64(n5))
	i--
	dAtA[i] = 0x1a
	{
		size, err := m.DecreaseBondAmount.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintSequencer(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.SequencerAddress) > 0 {
		i -= len(m.SequencerAddress)
		copy(dAtA[i:], m.SequencerAddress)
		i = encodeVarintSequencer(dAtA, i, uint64(len(m.SequencerAddress)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintSequencer(dAtA []byte, offset int, v uint64) int {
	offset -= sovSequencer(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Sequencer) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovSequencer(uint64(l))
	}
	if m.DymintPubKey != nil {
		l = m.DymintPubKey.Size()
		n += 1 + l + sovSequencer(uint64(l))
	}
	l = len(m.RollappId)
	if l > 0 {
		n += 1 + l + sovSequencer(uint64(l))
	}
	l = m.Metadata.Size()
	n += 1 + l + sovSequencer(uint64(l))
	if m.Jailed {
		n += 2
	}
	if m.Status != 0 {
		n += 1 + sovSequencer(uint64(m.Status))
	}
	if len(m.Tokens) > 0 {
		for _, e := range m.Tokens {
			l = e.Size()
			n += 1 + l + sovSequencer(uint64(l))
		}
	}
	if m.UnbondRequestHeight != 0 {
		n += 1 + sovSequencer(uint64(m.UnbondRequestHeight))
	}
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.UnbondTime)
	n += 1 + l + sovSequencer(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.NoticePeriodTime)
	n += 1 + l + sovSequencer(uint64(l))
	return n
}

func (m *BondReduction) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.SequencerAddress)
	if l > 0 {
		n += 1 + l + sovSequencer(uint64(l))
	}
	l = m.DecreaseBondAmount.Size()
	n += 1 + l + sovSequencer(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.DecreaseBondTime)
	n += 1 + l + sovSequencer(uint64(l))
	return n
}

func sovSequencer(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozSequencer(x uint64) (n int) {
	return sovSequencer(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Sequencer) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSequencer
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
			return fmt.Errorf("proto: Sequencer: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Sequencer: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
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
				return ErrInvalidLengthSequencer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSequencer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DymintPubKey", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
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
				return ErrInvalidLengthSequencer
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSequencer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.DymintPubKey == nil {
				m.DymintPubKey = &types.Any{}
			}
			if err := m.DymintPubKey.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RollappId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
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
				return ErrInvalidLengthSequencer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSequencer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RollappId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Metadata", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
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
				return ErrInvalidLengthSequencer
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSequencer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Metadata.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Jailed", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Jailed = bool(v != 0)
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Status |= OperatingStatus(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Tokens", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
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
				return ErrInvalidLengthSequencer
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSequencer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Tokens = append(m.Tokens, types1.Coin{})
			if err := m.Tokens[len(m.Tokens)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field UnbondRequestHeight", wireType)
			}
			m.UnbondRequestHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.UnbondRequestHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field UnbondTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
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
				return ErrInvalidLengthSequencer
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSequencer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.UnbondTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NoticePeriodTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
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
				return ErrInvalidLengthSequencer
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSequencer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.NoticePeriodTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSequencer(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSequencer
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
func (m *BondReduction) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSequencer
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
			return fmt.Errorf("proto: BondReduction: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: BondReduction: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SequencerAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
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
				return ErrInvalidLengthSequencer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSequencer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SequencerAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DecreaseBondAmount", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
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
				return ErrInvalidLengthSequencer
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSequencer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.DecreaseBondAmount.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DecreaseBondTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSequencer
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
				return ErrInvalidLengthSequencer
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSequencer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.DecreaseBondTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSequencer(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSequencer
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
func skipSequencer(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSequencer
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
					return 0, ErrIntOverflowSequencer
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
					return 0, ErrIntOverflowSequencer
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
				return 0, ErrInvalidLengthSequencer
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupSequencer
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthSequencer
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthSequencer        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSequencer          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupSequencer = fmt.Errorf("proto: unexpected end of group")
)
