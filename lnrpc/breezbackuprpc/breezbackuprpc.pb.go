// Code generated by protoc-gen-go. DO NOT EDIT.
// source: breezbackuprpc/breezbackuprpc.proto

package breezbackuprpc

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type GetBackupRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetBackupRequest) Reset()         { *m = GetBackupRequest{} }
func (m *GetBackupRequest) String() string { return proto.CompactTextString(m) }
func (*GetBackupRequest) ProtoMessage()    {}
func (*GetBackupRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e730e5e074a28cd0, []int{0}
}

func (m *GetBackupRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetBackupRequest.Unmarshal(m, b)
}
func (m *GetBackupRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetBackupRequest.Marshal(b, m, deterministic)
}
func (m *GetBackupRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetBackupRequest.Merge(m, src)
}
func (m *GetBackupRequest) XXX_Size() int {
	return xxx_messageInfo_GetBackupRequest.Size(m)
}
func (m *GetBackupRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetBackupRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetBackupRequest proto.InternalMessageInfo

type GetBackupResponse struct {
	Files                []string `protobuf:"bytes,1,rep,name=files,proto3" json:"files,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetBackupResponse) Reset()         { *m = GetBackupResponse{} }
func (m *GetBackupResponse) String() string { return proto.CompactTextString(m) }
func (*GetBackupResponse) ProtoMessage()    {}
func (*GetBackupResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_e730e5e074a28cd0, []int{1}
}

func (m *GetBackupResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetBackupResponse.Unmarshal(m, b)
}
func (m *GetBackupResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetBackupResponse.Marshal(b, m, deterministic)
}
func (m *GetBackupResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetBackupResponse.Merge(m, src)
}
func (m *GetBackupResponse) XXX_Size() int {
	return xxx_messageInfo_GetBackupResponse.Size(m)
}
func (m *GetBackupResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetBackupResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetBackupResponse proto.InternalMessageInfo

func (m *GetBackupResponse) GetFiles() []string {
	if m != nil {
		return m.Files
	}
	return nil
}

func init() {
	proto.RegisterType((*GetBackupRequest)(nil), "breezbackuprpc.GetBackupRequest")
	proto.RegisterType((*GetBackupResponse)(nil), "breezbackuprpc.GetBackupResponse")
}

func init() {
	proto.RegisterFile("breezbackuprpc/breezbackuprpc.proto", fileDescriptor_e730e5e074a28cd0)
}

var fileDescriptor_e730e5e074a28cd0 = []byte{
	// 178 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x4e, 0x2a, 0x4a, 0x4d,
	0xad, 0x4a, 0x4a, 0x4c, 0xce, 0x2e, 0x2d, 0x28, 0x2a, 0x48, 0xd6, 0x47, 0xe5, 0xea, 0x15, 0x14,
	0xe5, 0x97, 0xe4, 0x0b, 0xf1, 0xa1, 0x8a, 0x2a, 0x09, 0x71, 0x09, 0xb8, 0xa7, 0x96, 0x38, 0x81,
	0xf9, 0x41, 0xa9, 0x85, 0xa5, 0xa9, 0xc5, 0x25, 0x4a, 0x9a, 0x5c, 0x82, 0x48, 0x62, 0xc5, 0x05,
	0xf9, 0x79, 0xc5, 0xa9, 0x42, 0x22, 0x5c, 0xac, 0x69, 0x99, 0x39, 0xa9, 0xc5, 0x12, 0x8c, 0x0a,
	0xcc, 0x1a, 0x9c, 0x41, 0x10, 0x8e, 0x51, 0x32, 0x17, 0xaf, 0x13, 0xc8, 0x40, 0x88, 0xe2, 0xd4,
	0x22, 0xa1, 0x20, 0x2e, 0x4e, 0xb8, 0x5e, 0x21, 0x05, 0x3d, 0x34, 0x37, 0xa0, 0x5b, 0x25, 0xa5,
	0x88, 0x47, 0x05, 0xc4, 0x62, 0x25, 0x06, 0x27, 0xb3, 0x28, 0x93, 0xf4, 0xcc, 0x92, 0x8c, 0xd2,
	0x24, 0xbd, 0xe4, 0xfc, 0x5c, 0xfd, 0x9c, 0xcc, 0xf4, 0x8c, 0x92, 0xbc, 0xcc, 0xbc, 0xf4, 0xbc,
	0xd4, 0x92, 0xf2, 0xfc, 0xa2, 0x6c, 0xfd, 0x9c, 0xbc, 0x14, 0xfd, 0x9c, 0x3c, 0x4c, 0x1f, 0x27,
	0xb1, 0x81, 0xbd, 0x6c, 0x0c, 0x08, 0x00, 0x00, 0xff, 0xff, 0x86, 0xde, 0xa8, 0x6d, 0x19, 0x01,
	0x00, 0x00,
}
