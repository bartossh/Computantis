// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.25.3
// source: addons.proto

package protobufcompiled

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// AddonsAPIClient is the client API for AddonsAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AddonsAPIClient interface {
	AnalyzeTransaction(ctx context.Context, in *AddonsMessage, opts ...grpc.CallOption) (*AddonsError, error)
}

type addonsAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewAddonsAPIClient(cc grpc.ClientConnInterface) AddonsAPIClient {
	return &addonsAPIClient{cc}
}

func (c *addonsAPIClient) AnalyzeTransaction(ctx context.Context, in *AddonsMessage, opts ...grpc.CallOption) (*AddonsError, error) {
	out := new(AddonsError)
	err := c.cc.Invoke(ctx, "/computantis.AddonsAPI/AnalyzeTransaction", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AddonsAPIServer is the server API for AddonsAPI service.
// All implementations must embed UnimplementedAddonsAPIServer
// for forward compatibility
type AddonsAPIServer interface {
	AnalyzeTransaction(context.Context, *AddonsMessage) (*AddonsError, error)
	mustEmbedUnimplementedAddonsAPIServer()
}

// UnimplementedAddonsAPIServer must be embedded to have forward compatible implementations.
type UnimplementedAddonsAPIServer struct {
}

func (UnimplementedAddonsAPIServer) AnalyzeTransaction(context.Context, *AddonsMessage) (*AddonsError, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AnalyzeTransaction not implemented")
}
func (UnimplementedAddonsAPIServer) mustEmbedUnimplementedAddonsAPIServer() {}

// UnsafeAddonsAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AddonsAPIServer will
// result in compilation errors.
type UnsafeAddonsAPIServer interface {
	mustEmbedUnimplementedAddonsAPIServer()
}

func RegisterAddonsAPIServer(s grpc.ServiceRegistrar, srv AddonsAPIServer) {
	s.RegisterService(&AddonsAPI_ServiceDesc, srv)
}

func _AddonsAPI_AnalyzeTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddonsMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AddonsAPIServer).AnalyzeTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/computantis.AddonsAPI/AnalyzeTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AddonsAPIServer).AnalyzeTransaction(ctx, req.(*AddonsMessage))
	}
	return interceptor(ctx, in, info, handler)
}

// AddonsAPI_ServiceDesc is the grpc.ServiceDesc for AddonsAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AddonsAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "computantis.AddonsAPI",
	HandlerType: (*AddonsAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AnalyzeTransaction",
			Handler:    _AddonsAPI_AnalyzeTransaction_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "addons.proto",
}
