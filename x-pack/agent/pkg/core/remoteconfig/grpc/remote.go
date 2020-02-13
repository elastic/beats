// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package grpc

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ConfiguratorClient is the client API for Configurator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ConfiguratorClient interface {
	Config(ctx context.Context, in *ConfigRequest, opts ...grpc.CallOption) (*ConfigResponse, error)
	Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)
}

type configuratorClient struct {
	cc *grpc.ClientConn
}

func NewConfiguratorClient(cc *grpc.ClientConn) ConfiguratorClient {
	return &configuratorClient{cc}
}

func (c *configuratorClient) Config(ctx context.Context, in *ConfigRequest, opts ...grpc.CallOption) (*ConfigResponse, error) {
	out := new(ConfigResponse)
	err := c.cc.Invoke(ctx, "/grpc.Configurator/Config", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *configuratorClient) Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, "/grpc.Configurator/Status", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ConfiguratorServer is the server API for Configurator service.
type ConfiguratorServer interface {
	Config(context.Context, *ConfigRequest) (*ConfigResponse, error)
	Status(context.Context, *StatusRequest) (*StatusResponse, error)
}

// UnimplementedConfiguratorServer can be embedded to have forward compatible implementations.
type UnimplementedConfiguratorServer struct {
}

func (*UnimplementedConfiguratorServer) Config(ctx context.Context, req *ConfigRequest) (*ConfigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Config not implemented")
}
func (*UnimplementedConfiguratorServer) Status(ctx context.Context, req *StatusRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}

func RegisterConfiguratorServer(s *grpc.Server, srv ConfiguratorServer) {
	s.RegisterService(&_Configurator_serviceDesc, srv)
}

func _Configurator_Config_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConfiguratorServer).Config(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/grpc.Configurator/Config",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConfiguratorServer).Config(ctx, req.(*ConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Configurator_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConfiguratorServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/grpc.Configurator/Status",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConfiguratorServer).Status(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Configurator_serviceDesc = grpc.ServiceDesc{
	ServiceName: "grpc.Configurator",
	HandlerType: (*ConfiguratorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Config",
			Handler:    _Configurator_Config_Handler,
		},
		{
			MethodName: "Status",
			Handler:    _Configurator_Status_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "remote_config.proto",
}
