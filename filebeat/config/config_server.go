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

package config

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/fleet/x-pack/pkg/core/remoteconfig/grpc"
)

const (
	defaultTimeout = 5 * time.Second
)

// Server is a server for handling communication between
// beat and Elastic Agent
type Server struct {
	configChan chan<- map[string]interface{}
}

// NewConfigServer creates a new grpc configuration server for receiveing
// configurations from Elastic Agent
func NewConfigServer(configChan chan<- map[string]interface{}) *Server {
	return &Server{
		configChan: configChan,
	}
}

// Config is a handler of a call made by agent pushing latest configuration
func (s *Server) Config(ctx context.Context, req *grpc.ConfigRequest) (*grpc.ConfigResponse, error) {
	cfgString := req.GetConfig()

	var configMap common.MapStr
	uconfig, err := common.NewConfigFrom(cfgString)
	if err != nil {
		return &grpc.ConfigResponse{}, fmt.Errorf("Config blocks unsuccessfully generated: %+v", err)
	}

	err = uconfig.Unpack(&configMap)
	if err != nil {
		return &grpc.ConfigResponse{}, fmt.Errorf("Config blocks unsuccessfully generated: %+v", err)
	}

	select {
	case s.configChan <- configMap:
	case <-time.After(defaultTimeout):
		return &grpc.ConfigResponse{}, errors.New("failed to push configuration: Timeout")
	}
	return &grpc.ConfigResponse{}, nil
}
