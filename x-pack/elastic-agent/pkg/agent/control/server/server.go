// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"context"
	"net"

	"google.golang.org/grpc"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/proto"
)

type Server struct {
	logger       *logger.Logger
	listener     net.Listener
	server       *grpc.Server
}

func New(log *logger.Logger) *Server {
	return &Server{
		logger: log,
	}
}

// Start starts the GRPC endpoint and accepts new connections.
func (s *Server) Start() error {
	if s.server != nil {
		// already started
		return nil
	}

	lis, err := createListener()
	if err != nil {
		return err
	}
	s.listener = lis
	s.server = grpc.NewServer()
	proto.RegisterElasticAgentServer(s.server, s)

	// start serving GRPC connections
	go func() {
		err := s.server.Serve(lis)
		if err != nil {
			s.logger.Errorf("error listening for GRPC: %s", err)
		}
	}()

	return nil
}

// Stop stops the GRPC endpoint.
func (s *Server) Stop() {
	if s.server != nil {
		s.server.Stop()
		s.server = nil
		s.listener = nil
	}
}

func (s *Server) Version(ctx context.Context, empty *proto.Empty) (*proto.VersionResponse, error) {
	panic("implement me")
}

func (s *Server) Status(ctx context.Context, empty *proto.Empty) (*proto.StatusResponse, error) {
	panic("implement me")
}

func (s *Server) Restart(ctx context.Context, empty *proto.Empty) (*proto.Empty, error) {
	panic("implement me")
}

func (s *Server) Upgrade(ctx context.Context, request *proto.UpgradeRequest) (*proto.UpgradeResponse, error) {
	panic("implement me")
}
