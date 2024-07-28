// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: dymensionxyz/dymension/sequencer/params.proto

package types

import (
	fmt "fmt"
	types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	github_com_cosmos_gogoproto_types "github.com/cosmos/gogoproto/types"
	_ "google.golang.org/protobuf/types/known/durationpb"
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

// Params defines the parameters for the module.
type Params struct {
	MinBond types.Coin `protobuf:"bytes,1,opt,name=min_bond,json=minBond,proto3" json:"min_bond,omitempty"`
	// unbonding_time is the time duration of unbonding.
	UnbondingTime time.Duration `protobuf:"bytes,2,opt,name=unbonding_time,json=unbondingTime,proto3,stdduration" json:"unbonding_time"`
	// notice_period is the time duration of notice period.
	NoticePeriod time.Duration `protobuf:"bytes,3,opt,name=notice_period,json=noticePeriod,proto3,stdduration" json:"notice_period"`
}

func (m *Params) Reset()      { *m = Params{} }
func (*Params) ProtoMessage() {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_599b0eefba99ee26, []int{0}
}
func (m *Params) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Params) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Params) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}
func (m *Params) XXX_Size() int {
	return m.Size()
}
func (m *Params) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func (m *Params) GetMinBond() types.Coin {
	if m != nil {
		return m.MinBond
	}
	return types.Coin{}
}

func (m *Params) GetUnbondingTime() time.Duration {
	if m != nil {
		return m.UnbondingTime
	}
	return 0
}

func (m *Params) GetNoticePeriod() time.Duration {
	if m != nil {
		return m.NoticePeriod
	}
	return 0
}

func init() {
	proto.RegisterType((*Params)(nil), "dymensionxyz.dymension.sequencer.Params")
}

func init() {
	proto.RegisterFile("dymensionxyz/dymension/sequencer/params.proto", fileDescriptor_599b0eefba99ee26)
}

var fileDescriptor_599b0eefba99ee26 = []byte{
	// 342 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x91, 0xbf, 0x4b, 0xc3, 0x40,
	0x14, 0xc7, 0x73, 0x0a, 0xb5, 0x44, 0xeb, 0x10, 0x1c, 0x6a, 0x87, 0xa4, 0x38, 0x39, 0xe8, 0x1d,
	0xb5, 0xe0, 0xe0, 0x18, 0x1d, 0xc4, 0x29, 0x14, 0x27, 0x97, 0x92, 0x1f, 0xcf, 0x78, 0xe0, 0xdd,
	0x8b, 0xb9, 0x4b, 0x69, 0xfc, 0x2b, 0x1c, 0x3b, 0xf6, 0xcf, 0xe9, 0xd8, 0xd1, 0xa9, 0x4a, 0xbb,
	0x88, 0x7f, 0x82, 0x93, 0x24, 0x69, 0x43, 0x17, 0xc1, 0xed, 0xde, 0xbd, 0xef, 0xf7, 0xc3, 0x07,
	0x9e, 0x79, 0x1e, 0xe5, 0x02, 0xa4, 0xe2, 0x28, 0xc7, 0xf9, 0x2b, 0xab, 0x07, 0xa6, 0xe0, 0x25,
	0x03, 0x19, 0x42, 0xca, 0x12, 0x3f, 0xf5, 0x85, 0xa2, 0x49, 0x8a, 0x1a, 0xad, 0xee, 0x76, 0x9c,
	0xd6, 0x03, 0xad, 0xe3, 0x9d, 0xa3, 0x18, 0x63, 0x2c, 0xc3, 0xac, 0x78, 0x55, 0xbd, 0x8e, 0x1d,
	0xa2, 0x12, 0xa8, 0x58, 0xe0, 0x2b, 0x60, 0xa3, 0x5e, 0x00, 0xda, 0xef, 0xb1, 0x10, 0xb9, 0xdc,
	0xec, 0x63, 0xc4, 0xf8, 0x19, 0x58, 0x39, 0x05, 0xd9, 0x23, 0x8b, 0xb2, 0xd4, 0xd7, 0x05, 0xb9,
	0xfc, 0x39, 0xf9, 0x21, 0x66, 0xc3, 0x2b, 0x45, 0x2c, 0xcf, 0x6c, 0x0a, 0x2e, 0x87, 0x01, 0xca,
	0xa8, 0x4d, 0xba, 0xe4, 0x74, 0xff, 0xe2, 0x98, 0x56, 0x74, 0x5a, 0xd0, 0xe9, 0x9a, 0x4e, 0xaf,
	0x91, 0x4b, 0xb7, 0x33, 0x5b, 0x38, 0xc6, 0xf7, 0xc2, 0xb1, 0x36, 0x95, 0x33, 0x14, 0x5c, 0x83,
	0x48, 0x74, 0x3e, 0xd8, 0x13, 0x5c, 0xba, 0x28, 0x23, 0xeb, 0xce, 0x3c, 0xcc, 0x64, 0xb1, 0xe4,
	0x32, 0x1e, 0x6a, 0x2e, 0xa0, 0xbd, 0xb3, 0xe6, 0x56, 0x56, 0x74, 0x63, 0x45, 0x6f, 0xd6, 0x56,
	0x6e, 0xb3, 0xe0, 0x4e, 0x3e, 0x1c, 0x32, 0x68, 0xd5, 0xd5, 0x7b, 0x2e, 0xc0, 0xba, 0x35, 0x5b,
	0x12, 0x35, 0x0f, 0x61, 0x98, 0x40, 0xca, 0x31, 0x6a, 0xef, 0xfe, 0x1f, 0x75, 0x50, 0x35, 0xbd,
	0xb2, 0x78, 0xd5, 0x9c, 0x4c, 0x1d, 0xe3, 0x6b, 0xea, 0x10, 0xd7, 0x9b, 0x2d, 0x6d, 0x32, 0x5f,
	0xda, 0xe4, 0x73, 0x69, 0x93, 0xb7, 0x95, 0x6d, 0xcc, 0x57, 0xb6, 0xf1, 0xbe, 0xb2, 0x8d, 0x87,
	0xcb, 0x98, 0xeb, 0xa7, 0x2c, 0xa0, 0x21, 0x0a, 0xf6, 0xc7, 0x21, 0x47, 0x7d, 0x36, 0xde, 0xba,
	0xa6, 0xce, 0x13, 0x50, 0x41, 0xa3, 0xd4, 0xe8, 0xff, 0x06, 0x00, 0x00, 0xff, 0xff, 0xfd, 0x81,
	0x25, 0x58, 0xfe, 0x01, 0x00, 0x00,
}

func (this *Params) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Params)
	if !ok {
		that2, ok := that.(Params)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if !this.MinBond.Equal(&that1.MinBond) {
		return false
	}
	if this.UnbondingTime != that1.UnbondingTime {
		return false
	}
	if this.NoticePeriod != that1.NoticePeriod {
		return false
	}
	return true
}
func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n1, err1 := github_com_cosmos_gogoproto_types.StdDurationMarshalTo(m.NoticePeriod, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.NoticePeriod):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintParams(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x1a
	n2, err2 := github_com_cosmos_gogoproto_types.StdDurationMarshalTo(m.UnbondingTime, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.UnbondingTime):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintParams(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0x12
	{
		size, err := m.MinBond.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintParams(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintParams(dAtA []byte, offset int, v uint64) int {
	offset -= sovParams(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.MinBond.Size()
	n += 1 + l + sovParams(uint64(l))
	l = github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.UnbondingTime)
	n += 1 + l + sovParams(uint64(l))
	l = github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.NoticePeriod)
	n += 1 + l + sovParams(uint64(l))
	return n
}

func sovParams(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozParams(x uint64) (n int) {
	return sovParams(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowParams
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
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MinBond", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowParams
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
				return ErrInvalidLengthParams
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthParams
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MinBond.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field UnbondingTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowParams
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
				return ErrInvalidLengthParams
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthParams
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdDurationUnmarshal(&m.UnbondingTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NoticePeriod", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowParams
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
				return ErrInvalidLengthParams
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthParams
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdDurationUnmarshal(&m.NoticePeriod, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipParams(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthParams
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
func skipParams(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowParams
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
					return 0, ErrIntOverflowParams
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
					return 0, ErrIntOverflowParams
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
				return 0, ErrInvalidLengthParams
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupParams
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthParams
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthParams        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowParams          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupParams = fmt.Errorf("proto: unexpected end of group")
)
