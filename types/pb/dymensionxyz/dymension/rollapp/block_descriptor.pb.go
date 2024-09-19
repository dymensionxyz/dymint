// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: types/dymensionxyz/dymension/rollapp/block_descriptor.proto

package rollapp

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	github_com_gogo_protobuf_types "github.com/gogo/protobuf/types"
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

// BlockDescriptor defines a single rollapp chain block description.
type BlockDescriptor struct {
	// height is the height of the block
	Height uint64 `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	// stateRoot is a 32 byte array of the hash of the block (state root of the block)
	StateRoot []byte `protobuf:"bytes,2,opt,name=stateRoot,proto3" json:"stateRoot,omitempty"`
	// timestamp is the time from the block header
	Timestamp time.Time `protobuf:"bytes,3,opt,name=timestamp,proto3,stdtime" json:"timestamp"`
}

func (m *BlockDescriptor) Reset()         { *m = BlockDescriptor{} }
func (m *BlockDescriptor) String() string { return proto.CompactTextString(m) }
func (*BlockDescriptor) ProtoMessage()    {}
func (*BlockDescriptor) Descriptor() ([]byte, []int) {
	return fileDescriptor_7dcfa105ccca6c3f, []int{0}
}
func (m *BlockDescriptor) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *BlockDescriptor) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_BlockDescriptor.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *BlockDescriptor) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BlockDescriptor.Merge(m, src)
}
func (m *BlockDescriptor) XXX_Size() int {
	return m.Size()
}
func (m *BlockDescriptor) XXX_DiscardUnknown() {
	xxx_messageInfo_BlockDescriptor.DiscardUnknown(m)
}

var xxx_messageInfo_BlockDescriptor proto.InternalMessageInfo

func (m *BlockDescriptor) GetHeight() uint64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *BlockDescriptor) GetStateRoot() []byte {
	if m != nil {
		return m.StateRoot
	}
	return nil
}

func (m *BlockDescriptor) GetTimestamp() time.Time {
	if m != nil {
		return m.Timestamp
	}
	return time.Time{}
}

// BlockDescriptors defines list of BlockDescriptor.
type BlockDescriptors struct {
	BD []BlockDescriptor `protobuf:"bytes,1,rep,name=BD,proto3" json:"BD"`
}

func (m *BlockDescriptors) Reset()         { *m = BlockDescriptors{} }
func (m *BlockDescriptors) String() string { return proto.CompactTextString(m) }
func (*BlockDescriptors) ProtoMessage()    {}
func (*BlockDescriptors) Descriptor() ([]byte, []int) {
	return fileDescriptor_7dcfa105ccca6c3f, []int{1}
}
func (m *BlockDescriptors) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *BlockDescriptors) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_BlockDescriptors.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *BlockDescriptors) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BlockDescriptors.Merge(m, src)
}
func (m *BlockDescriptors) XXX_Size() int {
	return m.Size()
}
func (m *BlockDescriptors) XXX_DiscardUnknown() {
	xxx_messageInfo_BlockDescriptors.DiscardUnknown(m)
}

var xxx_messageInfo_BlockDescriptors proto.InternalMessageInfo

func (m *BlockDescriptors) GetBD() []BlockDescriptor {
	if m != nil {
		return m.BD
	}
	return nil
}

func init() {
	proto.RegisterType((*BlockDescriptor)(nil), "dymensionxyz.dymension.rollapp.BlockDescriptor")
	proto.RegisterType((*BlockDescriptors)(nil), "dymensionxyz.dymension.rollapp.BlockDescriptors")
}

func init() {
	proto.RegisterFile("types/dymensionxyz/dymension/rollapp/block_descriptor.proto", fileDescriptor_7dcfa105ccca6c3f)
}

var fileDescriptor_7dcfa105ccca6c3f = []byte{
	// 306 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x91, 0xc1, 0x4a, 0xc3, 0x30,
	0x18, 0xc7, 0x9b, 0x6d, 0x0c, 0x97, 0x09, 0x4a, 0x11, 0x29, 0x43, 0xb2, 0xb2, 0x53, 0x4f, 0x09,
	0xcc, 0xa3, 0xb7, 0x30, 0x7d, 0x80, 0xe2, 0x45, 0x2f, 0xba, 0x6e, 0x31, 0x0b, 0xb6, 0xfb, 0x42,
	0x93, 0x81, 0xf3, 0x15, 0xbc, 0xec, 0xb1, 0x76, 0xdc, 0xd1, 0x93, 0xca, 0xfa, 0x22, 0xd2, 0x76,
	0x6b, 0x65, 0x82, 0xde, 0xf2, 0x0f, 0xdf, 0x2f, 0xbf, 0x24, 0x7f, 0x7c, 0x65, 0x97, 0x5a, 0x18,
	0x36, 0x5d, 0x26, 0x62, 0x6e, 0x14, 0xcc, 0x5f, 0x96, 0xaf, 0x75, 0x60, 0x29, 0xc4, 0xf1, 0x58,
	0x6b, 0x16, 0xc5, 0x30, 0x79, 0x7e, 0x98, 0x0a, 0x33, 0x49, 0x95, 0xb6, 0x90, 0x52, 0x9d, 0x82,
	0x05, 0x97, 0xfc, 0xc4, 0x68, 0x15, 0xe8, 0x0e, 0xeb, 0x9d, 0x49, 0x90, 0x50, 0x8c, 0xb2, 0x7c,
	0x55, 0x52, 0xbd, 0xbe, 0x04, 0x90, 0xb1, 0x60, 0x45, 0x8a, 0x16, 0x4f, 0xcc, 0xaa, 0x44, 0x18,
	0x3b, 0x4e, 0x74, 0x39, 0x30, 0x78, 0x43, 0xf8, 0x84, 0xe7, 0xc6, 0x51, 0x25, 0x74, 0xcf, 0x71,
	0x7b, 0x26, 0x94, 0x9c, 0x59, 0x0f, 0xf9, 0x28, 0x68, 0x85, 0xbb, 0xe4, 0x5e, 0xe0, 0x8e, 0xb1,
	0x63, 0x2b, 0x42, 0x00, 0xeb, 0x35, 0x7c, 0x14, 0x1c, 0x87, 0xf5, 0x86, 0xcb, 0x71, 0xa7, 0x3a,
	0xdc, 0x6b, 0xfa, 0x28, 0xe8, 0x0e, 0x7b, 0xb4, 0xd4, 0xd3, 0xbd, 0x9e, 0xde, 0xee, 0x27, 0xf8,
	0xd1, 0xfa, 0xa3, 0xef, 0xac, 0x3e, 0xfb, 0x28, 0xac, 0xb1, 0xc1, 0x1d, 0x3e, 0x3d, 0xb8, 0x8c,
	0x71, 0xaf, 0x71, 0x83, 0x8f, 0x3c, 0xe4, 0x37, 0x83, 0xee, 0x90, 0xd1, 0xbf, 0x7f, 0x81, 0x1e,
	0xd0, 0xbc, 0x95, 0x5b, 0xc2, 0x06, 0x1f, 0xf1, 0xc7, 0xf5, 0x96, 0xa0, 0xcd, 0x96, 0xa0, 0xaf,
	0x2d, 0x41, 0xab, 0x8c, 0x38, 0x9b, 0x8c, 0x38, 0xef, 0x19, 0x71, 0xee, 0x6f, 0xa4, 0xb2, 0xb3,
	0x45, 0x44, 0x27, 0x90, 0xfc, 0xea, 0x46, 0xcd, 0x2d, 0x2b, 0x5b, 0xd3, 0xd1, 0x3f, 0xc5, 0x45,
	0xed, 0xe2, 0x95, 0x97, 0xdf, 0x01, 0x00, 0x00, 0xff, 0xff, 0x0f, 0x6e, 0x09, 0x97, 0xe7, 0x01,
	0x00, 0x00,
}

func (m *BlockDescriptor) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *BlockDescriptor) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *BlockDescriptor) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n1, err1 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Timestamp, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Timestamp):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintBlockDescriptor(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x1a
	if len(m.StateRoot) > 0 {
		i -= len(m.StateRoot)
		copy(dAtA[i:], m.StateRoot)
		i = encodeVarintBlockDescriptor(dAtA, i, uint64(len(m.StateRoot)))
		i--
		dAtA[i] = 0x12
	}
	if m.Height != 0 {
		i = encodeVarintBlockDescriptor(dAtA, i, uint64(m.Height))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *BlockDescriptors) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *BlockDescriptors) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *BlockDescriptors) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.BD) > 0 {
		for iNdEx := len(m.BD) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.BD[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintBlockDescriptor(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintBlockDescriptor(dAtA []byte, offset int, v uint64) int {
	offset -= sovBlockDescriptor(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *BlockDescriptor) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Height != 0 {
		n += 1 + sovBlockDescriptor(uint64(m.Height))
	}
	l = len(m.StateRoot)
	if l > 0 {
		n += 1 + l + sovBlockDescriptor(uint64(l))
	}
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Timestamp)
	n += 1 + l + sovBlockDescriptor(uint64(l))
	return n
}

func (m *BlockDescriptors) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.BD) > 0 {
		for _, e := range m.BD {
			l = e.Size()
			n += 1 + l + sovBlockDescriptor(uint64(l))
		}
	}
	return n
}

func sovBlockDescriptor(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozBlockDescriptor(x uint64) (n int) {
	return sovBlockDescriptor(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *BlockDescriptor) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBlockDescriptor
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
			return fmt.Errorf("proto: BlockDescriptor: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: BlockDescriptor: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Height", wireType)
			}
			m.Height = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlockDescriptor
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Height |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field StateRoot", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlockDescriptor
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
				return ErrInvalidLengthBlockDescriptor
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthBlockDescriptor
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.StateRoot = append(m.StateRoot[:0], dAtA[iNdEx:postIndex]...)
			if m.StateRoot == nil {
				m.StateRoot = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlockDescriptor
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
				return ErrInvalidLengthBlockDescriptor
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthBlockDescriptor
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.Timestamp, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipBlockDescriptor(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBlockDescriptor
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
func (m *BlockDescriptors) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBlockDescriptor
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
			return fmt.Errorf("proto: BlockDescriptors: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: BlockDescriptors: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BD", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlockDescriptor
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
				return ErrInvalidLengthBlockDescriptor
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthBlockDescriptor
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BD = append(m.BD, BlockDescriptor{})
			if err := m.BD[len(m.BD)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipBlockDescriptor(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBlockDescriptor
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
func skipBlockDescriptor(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowBlockDescriptor
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
					return 0, ErrIntOverflowBlockDescriptor
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
					return 0, ErrIntOverflowBlockDescriptor
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
				return 0, ErrInvalidLengthBlockDescriptor
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupBlockDescriptor
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthBlockDescriptor
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthBlockDescriptor        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowBlockDescriptor          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupBlockDescriptor = fmt.Errorf("proto: unexpected end of group")
)
