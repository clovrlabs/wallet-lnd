// Code generated by protoc-gen-go. DO NOT EDIT.
// source: backuprpc/backup.proto

package backuprpc

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type BackupEventSubscription struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BackupEventSubscription) Reset()         { *m = BackupEventSubscription{} }
func (m *BackupEventSubscription) String() string { return proto.CompactTextString(m) }
func (*BackupEventSubscription) ProtoMessage()    {}
func (*BackupEventSubscription) Descriptor() ([]byte, []int) {
	return fileDescriptor_938215dbc7c25baf, []int{0}
}

func (m *BackupEventSubscription) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BackupEventSubscription.Unmarshal(m, b)
}
func (m *BackupEventSubscription) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BackupEventSubscription.Marshal(b, m, deterministic)
}
func (m *BackupEventSubscription) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BackupEventSubscription.Merge(m, src)
}
func (m *BackupEventSubscription) XXX_Size() int {
	return xxx_messageInfo_BackupEventSubscription.Size(m)
}
func (m *BackupEventSubscription) XXX_DiscardUnknown() {
	xxx_messageInfo_BackupEventSubscription.DiscardUnknown(m)
}

var xxx_messageInfo_BackupEventSubscription proto.InternalMessageInfo

type BackupEventUpdate struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BackupEventUpdate) Reset()         { *m = BackupEventUpdate{} }
func (m *BackupEventUpdate) String() string { return proto.CompactTextString(m) }
func (*BackupEventUpdate) ProtoMessage()    {}
func (*BackupEventUpdate) Descriptor() ([]byte, []int) {
	return fileDescriptor_938215dbc7c25baf, []int{1}
}

func (m *BackupEventUpdate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BackupEventUpdate.Unmarshal(m, b)
}
func (m *BackupEventUpdate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BackupEventUpdate.Marshal(b, m, deterministic)
}
func (m *BackupEventUpdate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BackupEventUpdate.Merge(m, src)
}
func (m *BackupEventUpdate) XXX_Size() int {
	return xxx_messageInfo_BackupEventUpdate.Size(m)
}
func (m *BackupEventUpdate) XXX_DiscardUnknown() {
	xxx_messageInfo_BackupEventUpdate.DiscardUnknown(m)
}

var xxx_messageInfo_BackupEventUpdate proto.InternalMessageInfo

func init() {
	proto.RegisterType((*BackupEventSubscription)(nil), "backuprpc.BackupEventSubscription")
	proto.RegisterType((*BackupEventUpdate)(nil), "backuprpc.BackupEventUpdate")
}

func init() { proto.RegisterFile("backuprpc/backup.proto", fileDescriptor_938215dbc7c25baf) }

var fileDescriptor_938215dbc7c25baf = []byte{
	// 123 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x4b, 0x4a, 0x4c, 0xce,
	0x2e, 0x2d, 0x28, 0x2a, 0x48, 0xd6, 0x87, 0xb0, 0xf4, 0x0a, 0x8a, 0xf2, 0x4b, 0xf2, 0x85, 0x38,
	0xe1, 0xe2, 0x4a, 0x92, 0x5c, 0xe2, 0x4e, 0x60, 0x8e, 0x6b, 0x59, 0x6a, 0x5e, 0x49, 0x70, 0x69,
	0x52, 0x71, 0x72, 0x51, 0x66, 0x41, 0x49, 0x66, 0x7e, 0x9e, 0x92, 0x30, 0x97, 0x20, 0x92, 0x54,
	0x68, 0x41, 0x4a, 0x62, 0x49, 0xaa, 0x51, 0x2a, 0x17, 0x1b, 0x44, 0x50, 0x28, 0x9a, 0x4b, 0x14,
	0xaa, 0x3c, 0x29, 0x15, 0x49, 0x5d, 0xb1, 0x90, 0x92, 0x1e, 0xdc, 0x78, 0x3d, 0x1c, 0x66, 0x4b,
	0xc9, 0x60, 0x57, 0x03, 0xb1, 0xc4, 0x80, 0x31, 0x89, 0x0d, 0xec, 0x50, 0x63, 0x40, 0x00, 0x00,
	0x00, 0xff, 0xff, 0xe1, 0xd6, 0xb3, 0x27, 0xc2, 0x00, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// BackupClient is the client API for Backup service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type BackupClient interface {
	SubscribeBackupEvents(ctx context.Context, in *BackupEventSubscription, opts ...grpc.CallOption) (Backup_SubscribeBackupEventsClient, error)
}

type backupClient struct {
	cc *grpc.ClientConn
}

func NewBackupClient(cc *grpc.ClientConn) BackupClient {
	return &backupClient{cc}
}

func (c *backupClient) SubscribeBackupEvents(ctx context.Context, in *BackupEventSubscription, opts ...grpc.CallOption) (Backup_SubscribeBackupEventsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Backup_serviceDesc.Streams[0], "/backuprpc.Backup/SubscribeBackupEvents", opts...)
	if err != nil {
		return nil, err
	}
	x := &backupSubscribeBackupEventsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Backup_SubscribeBackupEventsClient interface {
	Recv() (*BackupEventUpdate, error)
	grpc.ClientStream
}

type backupSubscribeBackupEventsClient struct {
	grpc.ClientStream
}

func (x *backupSubscribeBackupEventsClient) Recv() (*BackupEventUpdate, error) {
	m := new(BackupEventUpdate)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// BackupServer is the server API for Backup service.
type BackupServer interface {
	SubscribeBackupEvents(*BackupEventSubscription, Backup_SubscribeBackupEventsServer) error
}

// UnimplementedBackupServer can be embedded to have forward compatible implementations.
type UnimplementedBackupServer struct {
}

func (*UnimplementedBackupServer) SubscribeBackupEvents(req *BackupEventSubscription, srv Backup_SubscribeBackupEventsServer) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeBackupEvents not implemented")
}

func RegisterBackupServer(s *grpc.Server, srv BackupServer) {
	s.RegisterService(&_Backup_serviceDesc, srv)
}

func _Backup_SubscribeBackupEvents_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(BackupEventSubscription)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BackupServer).SubscribeBackupEvents(m, &backupSubscribeBackupEventsServer{stream})
}

type Backup_SubscribeBackupEventsServer interface {
	Send(*BackupEventUpdate) error
	grpc.ServerStream
}

type backupSubscribeBackupEventsServer struct {
	grpc.ServerStream
}

func (x *backupSubscribeBackupEventsServer) Send(m *BackupEventUpdate) error {
	return x.ServerStream.SendMsg(m)
}

var _Backup_serviceDesc = grpc.ServiceDesc{
	ServiceName: "backuprpc.Backup",
	HandlerType: (*BackupServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SubscribeBackupEvents",
			Handler:       _Backup_SubscribeBackupEvents_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "backuprpc/backup.proto",
}
