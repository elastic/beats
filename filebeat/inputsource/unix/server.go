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

package unix

import (
	"context"
	"fmt"
	"net"

	"golang.org/x/net/netutil"

	"github.com/menderesk/beats/v7/filebeat/inputsource"
	"github.com/menderesk/beats/v7/filebeat/inputsource/common/dgram"
	"github.com/menderesk/beats/v7/filebeat/inputsource/common/streaming"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

// Server is run by the input.
type Server interface {
	inputsource.Network
	Run(context.Context) error
}

// streamServer is a server for reading from Unix stream sockets.
type streamServer struct {
	*streaming.Listener
	config *Config
}

// datagramServer is a server for reading from Unix datagram sockets.
type datagramServer struct {
	*dgram.Listener
	config *Config
}

// New creates a new unix server.
func New(log *logp.Logger, config *Config, nf inputsource.NetworkFunc) (Server, error) {
	switch config.SocketType {
	case StreamSocket:
		splitFunc, err := streaming.SplitFunc(config.Framing, []byte(config.LineDelimiter))
		if err != nil {
			return nil, err
		}
		factory := streaming.SplitHandlerFactory(inputsource.FamilyUnix, log, MetadataCallback, nf, splitFunc)
		server := &streamServer{config: config}
		server.Listener = streaming.NewListener(inputsource.FamilyUnix, config.Path, factory, server.createServer, &streaming.ListenerConfig{
			Timeout:        config.Timeout,
			MaxMessageSize: config.MaxMessageSize,
			MaxConnections: config.MaxConnections,
		})
		return server, nil

	case DatagramSocket:
		server := &datagramServer{config: config}
		factory := dgram.DatagramReaderFactory(inputsource.FamilyUnix, log, nf)
		server.Listener = dgram.NewListener(inputsource.FamilyUnix, config.Path, factory, server.createConn, &dgram.ListenerConfig{
			Timeout:        config.Timeout,
			MaxMessageSize: config.MaxMessageSize,
		})
		return server, nil

	default:
	}
	return nil, fmt.Errorf("unknown unix server type")
}

func (s *streamServer) createServer() (net.Listener, error) {
	if err := cleanupStaleSocket(s.config.Path); err != nil {
		return nil, err
	}

	l, err := net.Listen("unix", s.config.Path)
	if err != nil {
		return nil, err
	}

	if err := setSocketOwnership(s.config.Path, s.config.Group); err != nil {
		return nil, err
	}

	if err := setSocketMode(s.config.Path, s.config.Mode); err != nil {
		return nil, err
	}

	if s.config.MaxConnections > 0 {
		return netutil.LimitListener(l, s.config.MaxConnections), nil
	}
	return l, nil
}

func (s *datagramServer) createConn() (net.PacketConn, error) {
	if err := cleanupStaleSocket(s.config.Path); err != nil {
		return nil, err
	}

	addr, err := net.ResolveUnixAddr("unixgram", s.config.Path)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		return nil, err
	}

	if err := setSocketOwnership(s.config.Path, s.config.Group); err != nil {
		return nil, err
	}

	if err := setSocketMode(s.config.Path, s.config.Mode); err != nil {
		return nil, err
	}
	return conn, nil
}
