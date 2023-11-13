// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.25.0
// source: notary.proto

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
	NotaryAPI_Alive_FullMethodName   = "/computantis.NotaryAPI/Alive"
	NotaryAPI_Propose_FullMethodName = "/computantis.NotaryAPI/Propose"
	NotaryAPI_Confirm_FullMethodName = "/computantis.NotaryAPI/Confirm"
	NotaryAPI_Reject_FullMethodName  = "/computantis.NotaryAPI/Reject"
	NotaryAPI_Waiting_FullMethodName = "/computantis.NotaryAPI/Waiting"
	NotaryAPI_Saved_FullMethodName   = "/computantis.NotaryAPI/Saved"
	NotaryAPI_Data_FullMethodName    = "/computantis.NotaryAPI/Data"
)

// NotaryAPIClient is the client API for NotaryAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type NotaryAPIClient interface {
	Alive(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*AliveData, error)
	Propose(ctx context.Context, in *Transaction, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Confirm(ctx context.Context, in *Transaction, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Reject(ctx context.Context, in *SignedHash, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Waiting(ctx context.Context, in *SignedHash, opts ...grpc.CallOption) (*Transactions, error)
	Saved(ctx context.Context, in *SignedHash, opts ...grpc.CallOption) (*Transaction, error)
	Data(ctx context.Context, in *Address, opts ...grpc.CallOption) (*DataBlob, error)
}

type notaryAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewNotaryAPIClient(cc grpc.ClientConnInterface) NotaryAPIClient {
	return &notaryAPIClient{cc}
}

func (c *notaryAPIClient) Alive(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*AliveData, error) {
	out := new(AliveData)
	err := c.cc.Invoke(ctx, NotaryAPI_Alive_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *notaryAPIClient) Propose(ctx context.Context, in *Transaction, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, NotaryAPI_Propose_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *notaryAPIClient) Confirm(ctx context.Context, in *Transaction, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, NotaryAPI_Confirm_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *notaryAPIClient) Reject(ctx context.Context, in *SignedHash, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, NotaryAPI_Reject_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *notaryAPIClient) Waiting(ctx context.Context, in *SignedHash, opts ...grpc.CallOption) (*Transactions, error) {
	out := new(Transactions)
	err := c.cc.Invoke(ctx, NotaryAPI_Waiting_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *notaryAPIClient) Saved(ctx context.Context, in *SignedHash, opts ...grpc.CallOption) (*Transaction, error) {
	out := new(Transaction)
	err := c.cc.Invoke(ctx, NotaryAPI_Saved_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *notaryAPIClient) Data(ctx context.Context, in *Address, opts ...grpc.CallOption) (*DataBlob, error) {
	out := new(DataBlob)
	err := c.cc.Invoke(ctx, NotaryAPI_Data_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NotaryAPIServer is the server API for NotaryAPI service.
// All implementations must embed UnimplementedNotaryAPIServer
// for forward compatibility
type NotaryAPIServer interface {
	Alive(context.Context, *emptypb.Empty) (*AliveData, error)
	Propose(context.Context, *Transaction) (*emptypb.Empty, error)
	Confirm(context.Context, *Transaction) (*emptypb.Empty, error)
	Reject(context.Context, *SignedHash) (*emptypb.Empty, error)
	Waiting(context.Context, *SignedHash) (*Transactions, error)
	Saved(context.Context, *SignedHash) (*Transaction, error)
	Data(context.Context, *Address) (*DataBlob, error)
	mustEmbedUnimplementedNotaryAPIServer()
}

// UnimplementedNotaryAPIServer must be embedded to have forward compatible implementations.
type UnimplementedNotaryAPIServer struct {
}

func (UnimplementedNotaryAPIServer) Alive(context.Context, *emptypb.Empty) (*AliveData, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Alive not implemented")
}
func (UnimplementedNotaryAPIServer) Propose(context.Context, *Transaction) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Propose not implemented")
}
func (UnimplementedNotaryAPIServer) Confirm(context.Context, *Transaction) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Confirm not implemented")
}
func (UnimplementedNotaryAPIServer) Reject(context.Context, *SignedHash) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Reject not implemented")
}
func (UnimplementedNotaryAPIServer) Waiting(context.Context, *SignedHash) (*Transactions, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Waiting not implemented")
}
func (UnimplementedNotaryAPIServer) Saved(context.Context, *SignedHash) (*Transaction, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Saved not implemented")
}
func (UnimplementedNotaryAPIServer) Data(context.Context, *Address) (*DataBlob, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Data not implemented")
}
func (UnimplementedNotaryAPIServer) mustEmbedUnimplementedNotaryAPIServer() {}

// UnsafeNotaryAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to NotaryAPIServer will
// result in compilation errors.
type UnsafeNotaryAPIServer interface {
	mustEmbedUnimplementedNotaryAPIServer()
}

func RegisterNotaryAPIServer(s grpc.ServiceRegistrar, srv NotaryAPIServer) {
	s.RegisterService(&NotaryAPI_ServiceDesc, srv)
}

func _NotaryAPI_Alive_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NotaryAPIServer).Alive(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NotaryAPI_Alive_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NotaryAPIServer).Alive(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _NotaryAPI_Propose_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Transaction)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NotaryAPIServer).Propose(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NotaryAPI_Propose_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NotaryAPIServer).Propose(ctx, req.(*Transaction))
	}
	return interceptor(ctx, in, info, handler)
}

func _NotaryAPI_Confirm_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Transaction)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NotaryAPIServer).Confirm(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NotaryAPI_Confirm_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NotaryAPIServer).Confirm(ctx, req.(*Transaction))
	}
	return interceptor(ctx, in, info, handler)
}

func _NotaryAPI_Reject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignedHash)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NotaryAPIServer).Reject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NotaryAPI_Reject_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NotaryAPIServer).Reject(ctx, req.(*SignedHash))
	}
	return interceptor(ctx, in, info, handler)
}

func _NotaryAPI_Waiting_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignedHash)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NotaryAPIServer).Waiting(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NotaryAPI_Waiting_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NotaryAPIServer).Waiting(ctx, req.(*SignedHash))
	}
	return interceptor(ctx, in, info, handler)
}

func _NotaryAPI_Saved_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignedHash)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NotaryAPIServer).Saved(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NotaryAPI_Saved_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NotaryAPIServer).Saved(ctx, req.(*SignedHash))
	}
	return interceptor(ctx, in, info, handler)
}

func _NotaryAPI_Data_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Address)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NotaryAPIServer).Data(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NotaryAPI_Data_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NotaryAPIServer).Data(ctx, req.(*Address))
	}
	return interceptor(ctx, in, info, handler)
}

// NotaryAPI_ServiceDesc is the grpc.ServiceDesc for NotaryAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var NotaryAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "computantis.NotaryAPI",
	HandlerType: (*NotaryAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Alive",
			Handler:    _NotaryAPI_Alive_Handler,
		},
		{
			MethodName: "Propose",
			Handler:    _NotaryAPI_Propose_Handler,
		},
		{
			MethodName: "Confirm",
			Handler:    _NotaryAPI_Confirm_Handler,
		},
		{
			MethodName: "Reject",
			Handler:    _NotaryAPI_Reject_Handler,
		},
		{
			MethodName: "Waiting",
			Handler:    _NotaryAPI_Waiting_Handler,
		},
		{
			MethodName: "Saved",
			Handler:    _NotaryAPI_Saved_Handler,
		},
		{
			MethodName: "Data",
			Handler:    _NotaryAPI_Data_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "notary.proto",
}
