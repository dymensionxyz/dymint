// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: types/dymensionxyz/dymension/sequencer/operating_status.proto

package sequencer

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	math "math"
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

// OperatingStatus defines the operating status of a sequencer
type OperatingStatus int32

const (
	// OPERATING_STATUS_UNBONDED defines a sequencer that is not active and won't
	// be scheduled
	Unbonded OperatingStatus = 0
	// UNBONDING defines a sequencer that is currently unbonding.
	Unbonding OperatingStatus = 1
	// OPERATING_STATUS_BONDED defines a sequencer that is bonded and can be
	// scheduled
	Bonded OperatingStatus = 2
)

var OperatingStatus_name = map[int32]string{
	0: "OPERATING_STATUS_UNBONDED",
	1: "OPERATING_STATUS_UNBONDING",
	2: "OPERATING_STATUS_BONDED",
}

var OperatingStatus_value = map[string]int32{
	"OPERATING_STATUS_UNBONDED":  0,
	"OPERATING_STATUS_UNBONDING": 1,
	"OPERATING_STATUS_BONDED":    2,
}

func (x OperatingStatus) String() string {
	return proto.EnumName(OperatingStatus_name, int32(x))
}

func (OperatingStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d854b24b5c08f0d1, []int{0}
}

func init() {
	proto.RegisterEnum("dymensionxyz.dymension.sequencer.OperatingStatus", OperatingStatus_name, OperatingStatus_value)
}

func init() {
	proto.RegisterFile("types/dymensionxyz/dymension/sequencer/operating_status.proto", fileDescriptor_d854b24b5c08f0d1)
}

var fileDescriptor_d854b24b5c08f0d1 = []byte{
	// 263 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xb2, 0x2d, 0xa9, 0x2c, 0x48,
	0x2d, 0xd6, 0x4f, 0xa9, 0xcc, 0x4d, 0xcd, 0x2b, 0xce, 0xcc, 0xcf, 0xab, 0xa8, 0xac, 0x42, 0x70,
	0xf4, 0x8b, 0x53, 0x0b, 0x4b, 0x53, 0xf3, 0x92, 0x53, 0x8b, 0xf4, 0xf3, 0x0b, 0x52, 0x8b, 0x12,
	0x4b, 0x32, 0xf3, 0xd2, 0xe3, 0x8b, 0x4b, 0x12, 0x4b, 0x4a, 0x8b, 0xf5, 0x0a, 0x8a, 0xf2, 0x4b,
	0xf2, 0x85, 0x14, 0x90, 0x35, 0xea, 0xc1, 0x39, 0x7a, 0x70, 0x8d, 0x52, 0x22, 0xe9, 0xf9, 0xe9,
	0xf9, 0x60, 0xc5, 0xfa, 0x20, 0x16, 0x44, 0x9f, 0xd6, 0x1c, 0x46, 0x2e, 0x7e, 0x7f, 0x98, 0x91,
	0xc1, 0x60, 0x13, 0x85, 0xb4, 0xb9, 0x24, 0xfd, 0x03, 0x5c, 0x83, 0x1c, 0x43, 0x3c, 0xfd, 0xdc,
	0xe3, 0x83, 0x43, 0x1c, 0x43, 0x42, 0x83, 0xe3, 0x43, 0xfd, 0x9c, 0xfc, 0xfd, 0x5c, 0x5c, 0x5d,
	0x04, 0x18, 0xa4, 0x78, 0xba, 0xe6, 0x2a, 0x70, 0x84, 0xe6, 0x25, 0xe5, 0xe7, 0xa5, 0xa4, 0xa6,
	0x08, 0xe9, 0x72, 0x49, 0xe1, 0x50, 0xec, 0xe9, 0xe7, 0x2e, 0xc0, 0x28, 0xc5, 0xdb, 0x35, 0x57,
	0x81, 0x13, 0xa2, 0x3a, 0x33, 0x2f, 0x5d, 0x48, 0x9d, 0x4b, 0x1c, 0x43, 0x39, 0xd4, 0x64, 0x26,
	0x29, 0xae, 0xae, 0xb9, 0x0a, 0x6c, 0x4e, 0x60, 0x73, 0xa5, 0x58, 0x3a, 0x16, 0xcb, 0x31, 0x38,
	0x25, 0x9d, 0x78, 0x24, 0xc7, 0x78, 0xe1, 0x91, 0x1c, 0xe3, 0x83, 0x47, 0x72, 0x8c, 0x13, 0x1e,
	0xcb, 0x31, 0x5c, 0x78, 0x2c, 0xc7, 0x70, 0xe3, 0xb1, 0x1c, 0x43, 0x94, 0x47, 0x7a, 0x66, 0x49,
	0x46, 0x69, 0x92, 0x5e, 0x72, 0x7e, 0x2e, 0x46, 0xa0, 0x65, 0xe6, 0x95, 0xe8, 0x43, 0x82, 0xb3,
	0x20, 0x89, 0x60, 0x88, 0x26, 0xb1, 0x81, 0x43, 0xc2, 0x18, 0x10, 0x00, 0x00, 0xff, 0xff, 0x10,
	0x0f, 0x7d, 0xbc, 0x82, 0x01, 0x00, 0x00,
}
