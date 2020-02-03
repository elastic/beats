// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/agent/pkg/core/remoteconfig/grpc"
)

const (
	defaultTimeout = 15 * time.Second
)

// Server is a server for handling communication between
// beat and Elastic Agent.
type Server struct {
	configChan chan<- map[string]interface{}
}

// NewConfigServer creates a new grpc configuration server for receiving
// configurations from Elastic Agent.
func NewConfigServer(configChan chan<- map[string]interface{}) *Server {
	return &Server{
		configChan: configChan,
	}
}

// Config is a handler of a call made by agent pushing latest configuration.
func (s *Server) Config(ctx context.Context, req *grpc.ConfigRequest) (*grpc.ConfigResponse, error) {
	cfgString := req.GetConfig()

	var configMap common.MapStr
	uconfig, err := common.NewConfigFrom(cfgString)
	if err != nil {
		return &grpc.ConfigResponse{}, fmt.Errorf("config blocks unsuccessfully generated: %+v", err)
	}

	err = uconfig.Unpack(&configMap)
	if err != nil {
		return &grpc.ConfigResponse{}, fmt.Errorf("config blocks unsuccessfully generated: %+v", err)
	}

	select {
	case s.configChan <- configMap:
	case <-time.After(defaultTimeout):
		return &grpc.ConfigResponse{}, errors.New("failed to push configuration: Timeout")
	}
	return &grpc.ConfigResponse{}, nil
}

// Status returns OK.
func (s *Server) Status(ctx context.Context, req *grpc.StatusRequest) (*grpc.StatusResponse, error) {
	return &grpc.StatusResponse{Status: "ok"}, nil
}
