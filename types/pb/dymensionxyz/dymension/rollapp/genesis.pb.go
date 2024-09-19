// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: types/dymensionxyz/dymension/rollapp/genesis.proto

package rollapp

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
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

// GenesisState defines the rollapp module's genesis state.
type GenesisState struct {
	Params                             Params                           `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
	RollappList                        []Rollapp                        `protobuf:"bytes,2,rep,name=rollappList,proto3" json:"rollappList"`
	StateInfoList                      []StateInfo                      `protobuf:"bytes,3,rep,name=stateInfoList,proto3" json:"stateInfoList"`
	LatestStateInfoIndexList           []StateInfoIndex                 `protobuf:"bytes,4,rep,name=latestStateInfoIndexList,proto3" json:"latestStateInfoIndexList"`
	LatestFinalizedStateIndexList      []StateInfoIndex                 `protobuf:"bytes,5,rep,name=latestFinalizedStateIndexList,proto3" json:"latestFinalizedStateIndexList"`
	BlockHeightToFinalizationQueueList []BlockHeightToFinalizationQueue `protobuf:"bytes,6,rep,name=blockHeightToFinalizationQueueList,proto3" json:"blockHeightToFinalizationQueueList"`
	// LivenessEvents are scheduled upcoming liveness events
	LivenessEvents []LivenessEvent `protobuf:"bytes,7,rep,name=livenessEvents,proto3" json:"livenessEvents"`
}

func (m *GenesisState) Reset()         { *m = GenesisState{} }
func (m *GenesisState) String() string { return proto.CompactTextString(m) }
func (*GenesisState) ProtoMessage()    {}
func (*GenesisState) Descriptor() ([]byte, []int) {
	return fileDescriptor_cbb9c396fa138f1f, []int{0}
}
func (m *GenesisState) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GenesisState) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GenesisState.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GenesisState) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenesisState.Merge(m, src)
}
func (m *GenesisState) XXX_Size() int {
	return m.Size()
}
func (m *GenesisState) XXX_DiscardUnknown() {
	xxx_messageInfo_GenesisState.DiscardUnknown(m)
}

var xxx_messageInfo_GenesisState proto.InternalMessageInfo

func (m *GenesisState) GetParams() Params {
	if m != nil {
		return m.Params
	}
	return Params{}
}

func (m *GenesisState) GetRollappList() []Rollapp {
	if m != nil {
		return m.RollappList
	}
	return nil
}

func (m *GenesisState) GetStateInfoList() []StateInfo {
	if m != nil {
		return m.StateInfoList
	}
	return nil
}

func (m *GenesisState) GetLatestStateInfoIndexList() []StateInfoIndex {
	if m != nil {
		return m.LatestStateInfoIndexList
	}
	return nil
}

func (m *GenesisState) GetLatestFinalizedStateIndexList() []StateInfoIndex {
	if m != nil {
		return m.LatestFinalizedStateIndexList
	}
	return nil
}

func (m *GenesisState) GetBlockHeightToFinalizationQueueList() []BlockHeightToFinalizationQueue {
	if m != nil {
		return m.BlockHeightToFinalizationQueueList
	}
	return nil
}

func (m *GenesisState) GetLivenessEvents() []LivenessEvent {
	if m != nil {
		return m.LivenessEvents
	}
	return nil
}

func init() {
	proto.RegisterType((*GenesisState)(nil), "dymensionxyz.dymension.rollapp.GenesisState")
}

func init() {
	proto.RegisterFile("types/dymensionxyz/dymension/rollapp/genesis.proto", fileDescriptor_cbb9c396fa138f1f)
}

var fileDescriptor_cbb9c396fa138f1f = []byte{
	// 430 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x93, 0xcf, 0x6f, 0xda, 0x30,
	0x1c, 0xc5, 0x93, 0xf1, 0x63, 0x92, 0xd9, 0x76, 0xb0, 0x76, 0x88, 0x90, 0x96, 0x21, 0x0e, 0x1b,
	0x3b, 0x2c, 0xd1, 0x40, 0x3b, 0x55, 0xea, 0x01, 0xb5, 0xb4, 0x48, 0x48, 0x6d, 0xa1, 0xbd, 0xb4,
	0x07, 0xea, 0x80, 0x09, 0x56, 0x83, 0x1d, 0xc5, 0x06, 0x01, 0x7f, 0x45, 0x0f, 0xfd, 0xa3, 0x38,
	0x72, 0xec, 0x09, 0x55, 0xf0, 0x8f, 0x54, 0x38, 0x0e, 0xa5, 0xad, 0x4a, 0x52, 0xf5, 0xe4, 0x58,
	0x7e, 0xef, 0xf3, 0x9e, 0xad, 0x7c, 0x41, 0x59, 0x4c, 0x7c, 0xcc, 0xed, 0xee, 0x64, 0x80, 0x29,
	0x27, 0x8c, 0x8e, 0x27, 0xd3, 0xa7, 0x8d, 0x1d, 0x30, 0xcf, 0x43, 0xbe, 0x6f, 0xbb, 0x98, 0x62,
	0x4e, 0xb8, 0xe5, 0x07, 0x4c, 0x30, 0x68, 0x6e, 0xab, 0xad, 0xcd, 0xc6, 0x52, 0xea, 0xfc, 0x77,
	0x97, 0xb9, 0x4c, 0x4a, 0xed, 0xf5, 0x57, 0xe8, 0xca, 0xff, 0x4b, 0x94, 0xe4, 0xa3, 0x00, 0x0d,
	0x54, 0x50, 0x3e, 0x59, 0x39, 0xb5, 0x2a, 0xcf, 0xff, 0x44, 0x1e, 0x2e, 0x90, 0xc0, 0x6d, 0x42,
	0x7b, 0x51, 0xbb, 0xbd, 0xf7, 0xbc, 0x43, 0x5b, 0x04, 0x88, 0xf2, 0x1e, 0x0e, 0x94, 0xb9, 0x92,
	0xc8, 0xec, 0x91, 0xd1, 0xda, 0xae, 0x2e, 0x57, 0x5c, 0x64, 0xc0, 0x97, 0xa3, 0x90, 0xd7, 0x5a,
	0xb7, 0x81, 0x07, 0x20, 0x1b, 0xde, 0xde, 0xd0, 0x0b, 0x7a, 0x29, 0x57, 0xfe, 0x65, 0xed, 0x7e,
	0x67, 0xeb, 0x54, 0xaa, 0xab, 0xe9, 0xd9, 0xe2, 0xa7, 0xd6, 0x54, 0x5e, 0x78, 0x02, 0x72, 0xea,
	0xbc, 0x41, 0xb8, 0x30, 0x3e, 0x15, 0x52, 0xa5, 0x5c, 0xf9, 0x77, 0x1c, 0xaa, 0x19, 0xae, 0x8a,
	0xb5, 0x4d, 0x80, 0x17, 0xe0, 0xab, 0x7c, 0xad, 0x3a, 0xed, 0x31, 0x89, 0x4c, 0x49, 0xe4, 0x9f,
	0x38, 0x64, 0x2b, 0x32, 0x29, 0xe8, 0x73, 0x0a, 0xf4, 0x81, 0xe1, 0x21, 0x81, 0xb9, 0xd8, 0xe8,
	0xea, 0xb4, 0x8b, 0xc7, 0x32, 0x21, 0x2d, 0x13, 0xac, 0xc4, 0x09, 0xd2, 0xa9, 0x62, 0xde, 0xa4,
	0xc2, 0x29, 0xf8, 0x11, 0x9e, 0xd5, 0x08, 0x45, 0x1e, 0x99, 0xe2, 0xae, 0x12, 0x45, 0xb1, 0x99,
	0x0f, 0xc4, 0xee, 0x46, 0xc3, 0x3b, 0x1d, 0x14, 0x1d, 0x8f, 0x75, 0x6e, 0x8e, 0x31, 0x71, 0xfb,
	0xe2, 0x9c, 0x29, 0x21, 0x12, 0x84, 0xd1, 0xb3, 0x21, 0x1e, 0x62, 0xd9, 0x20, 0x2b, 0x1b, 0xec,
	0xc7, 0x35, 0xa8, 0xee, 0x24, 0xa9, 0x46, 0x09, 0xf2, 0xe0, 0x15, 0xf8, 0x16, 0xfd, 0x95, 0x87,
	0x23, 0x4c, 0x05, 0x37, 0x3e, 0xcb, 0x06, 0x7f, 0xe3, 0x1a, 0x34, 0xb6, 0x5d, 0x2a, 0xf0, 0x05,
	0xaa, 0x7a, 0x3d, 0x5b, 0x9a, 0xfa, 0x7c, 0x69, 0xea, 0x0f, 0x4b, 0x53, 0xbf, 0x5d, 0x99, 0xda,
	0x7c, 0x65, 0x6a, 0xf7, 0x2b, 0x53, 0xbb, 0xac, 0xb9, 0x44, 0xf4, 0x87, 0x8e, 0xd5, 0x61, 0x83,
	0x57, 0x43, 0x43, 0xa8, 0xb0, 0xc3, 0x71, 0xf2, 0x9d, 0x98, 0x89, 0x72, 0xb2, 0x72, 0x92, 0x2a,
	0x8f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x44, 0x7a, 0x7f, 0x11, 0xc5, 0x04, 0x00, 0x00,
}

func (m *GenesisState) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GenesisState) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GenesisState) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.LivenessEvents) > 0 {
		for iNdEx := len(m.LivenessEvents) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.LivenessEvents[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x3a
		}
	}
	if len(m.BlockHeightToFinalizationQueueList) > 0 {
		for iNdEx := len(m.BlockHeightToFinalizationQueueList) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.BlockHeightToFinalizationQueueList[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x32
		}
	}
	if len(m.LatestFinalizedStateIndexList) > 0 {
		for iNdEx := len(m.LatestFinalizedStateIndexList) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.LatestFinalizedStateIndexList[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x2a
		}
	}
	if len(m.LatestStateInfoIndexList) > 0 {
		for iNdEx := len(m.LatestStateInfoIndexList) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.LatestStateInfoIndexList[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.StateInfoList) > 0 {
		for iNdEx := len(m.StateInfoList) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.StateInfoList[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.RollappList) > 0 {
		for iNdEx := len(m.RollappList) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.RollappList[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenesis(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintGenesis(dAtA []byte, offset int, v uint64) int {
	offset -= sovGenesis(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GenesisState) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Params.Size()
	n += 1 + l + sovGenesis(uint64(l))
	if len(m.RollappList) > 0 {
		for _, e := range m.RollappList {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.StateInfoList) > 0 {
		for _, e := range m.StateInfoList {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.LatestStateInfoIndexList) > 0 {
		for _, e := range m.LatestStateInfoIndexList {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.LatestFinalizedStateIndexList) > 0 {
		for _, e := range m.LatestFinalizedStateIndexList {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.BlockHeightToFinalizationQueueList) > 0 {
		for _, e := range m.BlockHeightToFinalizationQueueList {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.LivenessEvents) > 0 {
		for _, e := range m.LivenessEvents {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	return n
}

func sovGenesis(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGenesis(x uint64) (n int) {
	return sovGenesis(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GenesisState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
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
			return fmt.Errorf("proto: GenesisState: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GenesisState: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RollappList", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RollappList = append(m.RollappList, Rollapp{})
			if err := m.RollappList[len(m.RollappList)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field StateInfoList", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.StateInfoList = append(m.StateInfoList, StateInfo{})
			if err := m.StateInfoList[len(m.StateInfoList)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LatestStateInfoIndexList", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LatestStateInfoIndexList = append(m.LatestStateInfoIndexList, StateInfoIndex{})
			if err := m.LatestStateInfoIndexList[len(m.LatestStateInfoIndexList)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LatestFinalizedStateIndexList", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LatestFinalizedStateIndexList = append(m.LatestFinalizedStateIndexList, StateInfoIndex{})
			if err := m.LatestFinalizedStateIndexList[len(m.LatestFinalizedStateIndexList)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockHeightToFinalizationQueueList", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BlockHeightToFinalizationQueueList = append(m.BlockHeightToFinalizationQueueList, BlockHeightToFinalizationQueue{})
			if err := m.BlockHeightToFinalizationQueueList[len(m.BlockHeightToFinalizationQueueList)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LivenessEvents", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LivenessEvents = append(m.LivenessEvents, LivenessEvent{})
			if err := m.LivenessEvents[len(m.LivenessEvents)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
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
func skipGenesis(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
				return 0, ErrInvalidLengthGenesis
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGenesis
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGenesis
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGenesis        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenesis          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGenesis = fmt.Errorf("proto: unexpected end of group")
)
