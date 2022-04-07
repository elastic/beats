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

package tcp

import (
	"crypto/tls"
	"fmt"
	"net"

	"golang.org/x/net/netutil"

	"github.com/elastic/beats/v8/filebeat/inputsource"
	"github.com/elastic/beats/v8/filebeat/inputsource/common/streaming"
	"github.com/elastic/beats/v8/libbeat/common/transport/tlscommon"
)

// Server represent a TCP server
type Server struct {
	*streaming.Listener

	config    *Config
	tlsConfig *tlscommon.TLSConfig
}

// New creates a new tcp server
func New(
	config *Config,
	factory streaming.HandlerFactory,
) (*Server, error) {
	tlsConfig, err := tlscommon.LoadTLSServerConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	if factory == nil {
		return nil, fmt.Errorf("HandlerFactory can't be empty")
	}

	server := &Server{
		config:    config,
		tlsConfig: tlsConfig,
	}
	server.Listener = streaming.NewListener(inputsource.FamilyTCP, config.Host, factory, server.createServer, &streaming.ListenerConfig{
		Timeout:        config.Timeout,
		MaxMessageSize: config.MaxMessageSize,
		MaxConnections: config.MaxConnections,
	})

	return server, nil
}

func (s *Server) createServer() (net.Listener, error) {
	var l net.Listener
	var err error
	if s.tlsConfig != nil {
		t := s.tlsConfig.BuildServerConfig(s.config.Host)
		l, err = tls.Listen("tcp", s.config.Host, t)
		if err != nil {
			return nil, err
		}
	} else {
		l, err = net.Listen("tcp", s.config.Host)
		if err != nil {
			return nil, err
		}
	}

	if s.config.MaxConnections > 0 {
		return netutil.LimitListener(l, s.config.MaxConnections), nil
	}
	return l, nil
}
