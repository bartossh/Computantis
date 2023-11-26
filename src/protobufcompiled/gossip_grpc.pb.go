// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.25.0
// source: gossip.proto

package protobufcompiled

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	GossipAPI_Alive_FullMethodName     = "/computantis.GossipAPI/Alive"
	GossipAPI_LoadDag_FullMethodName   = "/computantis.GossipAPI/LoadDag"
	GossipAPI_Announce_FullMethodName  = "/computantis.GossipAPI/Announce"
	GossipAPI_Discover_FullMethodName  = "/computantis.GossipAPI/Discover"
	GossipAPI_GossipVrx_FullMethodName = "/computantis.GossipAPI/GossipVrx"
	GossipAPI_GossipTrx_FullMethodName = "/computantis.GossipAPI/GossipTrx"
)

// GossipAPIClient is the client API for GossipAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GossipAPIClient interface {
	Alive(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*AliveData, error)
	LoadDag(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (GossipAPI_LoadDagClient, error)
	Announce(ctx context.Context, in *ConnectionData, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Discover(ctx context.Context, in *ConnectionData, opts ...grpc.CallOption) (*ConnectedNodes, error)
	GossipVrx(ctx context.Context, in *VrxMsgGossip, opts ...grpc.CallOption) (*emptypb.Empty, error)
	GossipTrx(ctx context.Context, in *TrxMsgGossip, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type gossipAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewGossipAPIClient(cc grpc.ClientConnInterface) GossipAPIClient {
	return &gossipAPIClient{cc}
}

func (c *gossipAPIClient) Alive(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*AliveData, error) {
	out := new(AliveData)
	err := c.cc.Invoke(ctx, GossipAPI_Alive_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gossipAPIClient) LoadDag(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (GossipAPI_LoadDagClient, error) {
	stream, err := c.cc.NewStream(ctx, &GossipAPI_ServiceDesc.Streams[0], GossipAPI_LoadDag_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &gossipAPILoadDagClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type GossipAPI_LoadDagClient interface {
	Recv() (*Vertex, error)
	grpc.ClientStream
}

type gossipAPILoadDagClient struct {
	grpc.ClientStream
}

func (x *gossipAPILoadDagClient) Recv() (*Vertex, error) {
	m := new(Vertex)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *gossipAPIClient) Announce(ctx context.Context, in *ConnectionData, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, GossipAPI_Announce_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gossipAPIClient) Discover(ctx context.Context, in *ConnectionData, opts ...grpc.CallOption) (*ConnectedNodes, error) {
	out := new(ConnectedNodes)
	err := c.cc.Invoke(ctx, GossipAPI_Discover_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gossipAPIClient) GossipVrx(ctx context.Context, in *VrxMsgGossip, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, GossipAPI_GossipVrx_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gossipAPIClient) GossipTrx(ctx context.Context, in *TrxMsgGossip, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, GossipAPI_GossipTrx_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GossipAPIServer is the server API for GossipAPI service.
// All implementations must embed UnimplementedGossipAPIServer
// for forward compatibility
type GossipAPIServer interface {
	Alive(context.Context, *emptypb.Empty) (*AliveData, error)
	LoadDag(*emptypb.Empty, GossipAPI_LoadDagServer) error
	Announce(context.Context, *ConnectionData) (*emptypb.Empty, error)
	Discover(context.Context, *ConnectionData) (*ConnectedNodes, error)
	GossipVrx(context.Context, *VrxMsgGossip) (*emptypb.Empty, error)
	GossipTrx(context.Context, *TrxMsgGossip) (*emptypb.Empty, error)
	mustEmbedUnimplementedGossipAPIServer()
}

// UnimplementedGossipAPIServer must be embedded to have forward compatible implementations.
type UnimplementedGossipAPIServer struct {
}

func (UnimplementedGossipAPIServer) Alive(context.Context, *emptypb.Empty) (*AliveData, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Alive not implemented")
}
func (UnimplementedGossipAPIServer) LoadDag(*emptypb.Empty, GossipAPI_LoadDagServer) error {
	return status.Errorf(codes.Unimplemented, "method LoadDag not implemented")
}
func (UnimplementedGossipAPIServer) Announce(context.Context, *ConnectionData) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Announce not implemented")
}
func (UnimplementedGossipAPIServer) Discover(context.Context, *ConnectionData) (*ConnectedNodes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Discover not implemented")
}
func (UnimplementedGossipAPIServer) GossipVrx(context.Context, *VrxMsgGossip) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GossipVrx not implemented")
}
func (UnimplementedGossipAPIServer) GossipTrx(context.Context, *TrxMsgGossip) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GossipTrx not implemented")
}
func (UnimplementedGossipAPIServer) mustEmbedUnimplementedGossipAPIServer() {}

// UnsafeGossipAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GossipAPIServer will
// result in compilation errors.
type UnsafeGossipAPIServer interface {
	mustEmbedUnimplementedGossipAPIServer()
}

func RegisterGossipAPIServer(s grpc.ServiceRegistrar, srv GossipAPIServer) {
	s.RegisterService(&GossipAPI_ServiceDesc, srv)
}

func _GossipAPI_Alive_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GossipAPIServer).Alive(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GossipAPI_Alive_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GossipAPIServer).Alive(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _GossipAPI_LoadDag_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(emptypb.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GossipAPIServer).LoadDag(m, &gossipAPILoadDagServer{stream})
}

type GossipAPI_LoadDagServer interface {
	Send(*Vertex) error
	grpc.ServerStream
}

type gossipAPILoadDagServer struct {
	grpc.ServerStream
}

func (x *gossipAPILoadDagServer) Send(m *Vertex) error {
	return x.ServerStream.SendMsg(m)
}

func _GossipAPI_Announce_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConnectionData)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GossipAPIServer).Announce(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GossipAPI_Announce_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GossipAPIServer).Announce(ctx, req.(*ConnectionData))
	}
	return interceptor(ctx, in, info, handler)
}

func _GossipAPI_Discover_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConnectionData)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GossipAPIServer).Discover(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GossipAPI_Discover_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GossipAPIServer).Discover(ctx, req.(*ConnectionData))
	}
	return interceptor(ctx, in, info, handler)
}

func _GossipAPI_GossipVrx_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VrxMsgGossip)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GossipAPIServer).GossipVrx(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GossipAPI_GossipVrx_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GossipAPIServer).GossipVrx(ctx, req.(*VrxMsgGossip))
	}
	return interceptor(ctx, in, info, handler)
}

func _GossipAPI_GossipTrx_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TrxMsgGossip)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GossipAPIServer).GossipTrx(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GossipAPI_GossipTrx_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GossipAPIServer).GossipTrx(ctx, req.(*TrxMsgGossip))
	}
	return interceptor(ctx, in, info, handler)
}

// GossipAPI_ServiceDesc is the grpc.ServiceDesc for GossipAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var GossipAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "computantis.GossipAPI",
	HandlerType: (*GossipAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Alive",
			Handler:    _GossipAPI_Alive_Handler,
		},
		{
			MethodName: "Announce",
			Handler:    _GossipAPI_Announce_Handler,
		},
		{
			MethodName: "Discover",
			Handler:    _GossipAPI_Discover_Handler,
		},
		{
			MethodName: "GossipVrx",
			Handler:    _GossipAPI_GossipVrx_Handler,
		},
		{
			MethodName: "GossipTrx",
			Handler:    _GossipAPI_GossipTrx_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "LoadDag",
			Handler:       _GossipAPI_LoadDag_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "gossip.proto",
}
