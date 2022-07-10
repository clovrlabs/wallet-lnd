// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package breezbackuprpc

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// BreezBackuperClient is the client API for BreezBackuper service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BreezBackuperClient interface {
	GetBackup(ctx context.Context, in *GetBackupRequest, opts ...grpc.CallOption) (*GetBackupResponse, error)
}

type breezBackuperClient struct {
	cc grpc.ClientConnInterface
}

func NewBreezBackuperClient(cc grpc.ClientConnInterface) BreezBackuperClient {
	return &breezBackuperClient{cc}
}

func (c *breezBackuperClient) GetBackup(ctx context.Context, in *GetBackupRequest, opts ...grpc.CallOption) (*GetBackupResponse, error) {
	out := new(GetBackupResponse)
	err := c.cc.Invoke(ctx, "/breezbackuprpc.BreezBackuper/GetBackup", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BreezBackuperServer is the server API for BreezBackuper service.
// All implementations must embed UnimplementedBreezBackuperServer
// for forward compatibility
type BreezBackuperServer interface {
	GetBackup(context.Context, *GetBackupRequest) (*GetBackupResponse, error)
	mustEmbedUnimplementedBreezBackuperServer()
}

// UnimplementedBreezBackuperServer must be embedded to have forward compatible implementations.
type UnimplementedBreezBackuperServer struct {
}

func (UnimplementedBreezBackuperServer) GetBackup(context.Context, *GetBackupRequest) (*GetBackupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBackup not implemented")
}
func (UnimplementedBreezBackuperServer) mustEmbedUnimplementedBreezBackuperServer() {}

// UnsafeBreezBackuperServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BreezBackuperServer will
// result in compilation errors.
type UnsafeBreezBackuperServer interface {
	mustEmbedUnimplementedBreezBackuperServer()
}

func RegisterBreezBackuperServer(s *grpc.Server, srv BreezBackuperServer) {
	s.RegisterService(&_BreezBackuper_serviceDesc, srv)
}

func _BreezBackuper_GetBackup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBackupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BreezBackuperServer).GetBackup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/breezbackuprpc.BreezBackuper/GetBackup",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BreezBackuperServer).GetBackup(ctx, req.(*GetBackupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _BreezBackuper_serviceDesc = grpc.ServiceDesc{
	ServiceName: "breezbackuprpc.BreezBackuper",
	HandlerType: (*BreezBackuperServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetBackup",
			Handler:    _BreezBackuper_GetBackup_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "breezbackuprpc/breezbackuprpc.proto",
}
