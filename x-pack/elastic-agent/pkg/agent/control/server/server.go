// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"context"
	"net"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/reexec"

	"google.golang.org/grpc"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/proto"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// Server is the daemon side of the control protocol.
type Server struct {
	logger   *logger.Logger
	rex      reexec.ExecManager
	listener net.Listener
	server   *grpc.Server
}

// New creates a new control protocol server.
func New(log *logger.Logger, rex reexec.ExecManager) *Server {
	return &Server{
		logger: log,
		rex:    rex,
	}
}

// Start starts the GRPC endpoint and accepts new connections.
func (s *Server) Start() error {
	if s.server != nil {
		// already started
		return nil
	}

	lis, err := createListener(s.logger)
	if err != nil {
		return err
	}
	s.listener = lis
	s.server = grpc.NewServer()
	proto.RegisterElasticAgentControlServer(s.server, s)

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
		cleanupListener(s.logger)
	}
}

// Version returns the currently running version.
func (s *Server) Version(_ context.Context, _ *proto.Empty) (*proto.VersionResponse, error) {
	return &proto.VersionResponse{
		Version:   release.Version(),
		Commit:    release.Commit(),
		BuildTime: release.BuildTime().Format(control.TimeFormat()),
		Snapshot:  release.Snapshot(),
	}, nil
}

// Status returns the overall status of the agent.
func (s *Server) Status(_ context.Context, _ *proto.Empty) (*proto.StatusResponse, error) {
	// not implemented
	return &proto.StatusResponse{
		Status:       proto.Status_HEALTHY,
		Message:      "not implemented",
		Applications: nil,
	}, nil
}

// Restart performs re-exec.
func (s *Server) Restart(_ context.Context, _ *proto.Empty) (*proto.RestartResponse, error) {
	s.rex.ReExec()
	return &proto.RestartResponse{
		Status: proto.ActionStatus_SUCCESS,
	}, nil
}

// Upgrade performs the upgrade operation.
func (s *Server) Upgrade(ctx context.Context, request *proto.UpgradeRequest) (*proto.UpgradeResponse, error) {
	// not implemented
	return &proto.UpgradeResponse{
		Status:  proto.ActionStatus_FAILURE,
		Version: "",
		Error:   "not implemented",
	}, nil
}
