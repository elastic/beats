// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package api

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

// ProducerClient is the client API for Producer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ProducerClient interface {
	// Publishes an event via the Elastic agent shipper.
	//
	// Blocks until all processing steps complete and data is written to the queue. Returns a
	// RESOURCE_EXHAUSTED gRPC status code if the queue is full.
	//
	// Inputs may execute multiple concurrent Produce requests for independent data streams.
	// The order in which concurrent requests complete is not guaranteed. Use sequential requests to
	// control ordering.
	PublishEvents(ctx context.Context, in *PublishRequest, opts ...grpc.CallOption) (*PublishReply, error)
	// Returns a stream of acknowledgements from outputs.
	StreamAcknowledgements(ctx context.Context, in *StreamAcksRequest, opts ...grpc.CallOption) (Producer_StreamAcknowledgementsClient, error)
}

type producerClient struct {
	cc grpc.ClientConnInterface
}

func NewProducerClient(cc grpc.ClientConnInterface) ProducerClient {
	return &producerClient{cc}
}

func (c *producerClient) PublishEvents(ctx context.Context, in *PublishRequest, opts ...grpc.CallOption) (*PublishReply, error) {
	out := new(PublishReply)
	err := c.cc.Invoke(ctx, "/elastic.agent.shipper.v1.Producer/PublishEvents", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *producerClient) StreamAcknowledgements(ctx context.Context, in *StreamAcksRequest, opts ...grpc.CallOption) (Producer_StreamAcknowledgementsClient, error) {
	stream, err := c.cc.NewStream(ctx, &Producer_ServiceDesc.Streams[0], "/elastic.agent.shipper.v1.Producer/StreamAcknowledgements", opts...)
	if err != nil {
		return nil, err
	}
	x := &producerStreamAcknowledgementsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Producer_StreamAcknowledgementsClient interface {
	Recv() (*StreamAcksReply, error)
	grpc.ClientStream
}

type producerStreamAcknowledgementsClient struct {
	grpc.ClientStream
}

func (x *producerStreamAcknowledgementsClient) Recv() (*StreamAcksReply, error) {
	m := new(StreamAcksReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ProducerServer is the server API for Producer service.
// All implementations must embed UnimplementedProducerServer
// for forward compatibility
type ProducerServer interface {
	// Publishes an event via the Elastic agent shipper.
	//
	// Blocks until all processing steps complete and data is written to the queue. Returns a
	// RESOURCE_EXHAUSTED gRPC status code if the queue is full.
	//
	// Inputs may execute multiple concurrent Produce requests for independent data streams.
	// The order in which concurrent requests complete is not guaranteed. Use sequential requests to
	// control ordering.
	PublishEvents(context.Context, *PublishRequest) (*PublishReply, error)
	// Returns a stream of acknowledgements from outputs.
	StreamAcknowledgements(*StreamAcksRequest, Producer_StreamAcknowledgementsServer) error
	mustEmbedUnimplementedProducerServer()
}

// UnimplementedProducerServer must be embedded to have forward compatible implementations.
type UnimplementedProducerServer struct {
}

func (UnimplementedProducerServer) PublishEvents(context.Context, *PublishRequest) (*PublishReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PublishEvents not implemented")
}
func (UnimplementedProducerServer) StreamAcknowledgements(*StreamAcksRequest, Producer_StreamAcknowledgementsServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamAcknowledgements not implemented")
}
func (UnimplementedProducerServer) mustEmbedUnimplementedProducerServer() {}

// UnsafeProducerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ProducerServer will
// result in compilation errors.
type UnsafeProducerServer interface {
	mustEmbedUnimplementedProducerServer()
}

func RegisterProducerServer(s grpc.ServiceRegistrar, srv ProducerServer) {
	s.RegisterService(&Producer_ServiceDesc, srv)
}

func _Producer_PublishEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PublishRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProducerServer).PublishEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/elastic.agent.shipper.v1.Producer/PublishEvents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProducerServer).PublishEvents(ctx, req.(*PublishRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Producer_StreamAcknowledgements_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamAcksRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ProducerServer).StreamAcknowledgements(m, &producerStreamAcknowledgementsServer{stream})
}

type Producer_StreamAcknowledgementsServer interface {
	Send(*StreamAcksReply) error
	grpc.ServerStream
}

type producerStreamAcknowledgementsServer struct {
	grpc.ServerStream
}

func (x *producerStreamAcknowledgementsServer) Send(m *StreamAcksReply) error {
	return x.ServerStream.SendMsg(m)
}

// Producer_ServiceDesc is the grpc.ServiceDesc for Producer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Producer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "elastic.agent.shipper.v1.Producer",
	HandlerType: (*ProducerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PublishEvents",
			Handler:    _Producer_PublishEvents_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamAcknowledgements",
			Handler:       _Producer_StreamAcknowledgements_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "shipper.proto",
}
