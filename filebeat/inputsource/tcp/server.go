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
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"golang.org/x/net/netutil"

	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

// Server represent a TCP server
type Server struct {
	sync.RWMutex
	config    *Config
	Listener  net.Listener
	clients   map[string]*client
	wg        sync.WaitGroup
	done      chan struct{}
	factory   ClientFactory
	log       *logp.Logger
	tlsConfig *transport.TLSConfig
}

// New creates a new tcp server
func New(
	config *Config,
	factory ClientFactory,
) (*Server, error) {
	tlsConfig, err := tlscommon.LoadTLSServerConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	if factory == nil {
		return nil, fmt.Errorf("ClientFactory can't be empty")
	}

	return &Server{
		config:    config,
		clients:   make(map[string]*client, 0),
		done:      make(chan struct{}),
		factory:   factory,
		log:       logp.NewLogger("tcp").With("address", config.Host),
		tlsConfig: tlsConfig,
	}, nil
}

// Start listen to the TCP socket.
func (s *Server) Start() error {
	var err error
	s.Listener, err = s.createServer()
	if err != nil {
		return err
	}

	s.log.Info("Started listening for TCP connection")

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.run()
	}()
	return nil
}

// Run start and run a new TCP listener to receive new data
func (s *Server) run() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				s.log.Debugw("Can not accept the connection", "error", err)
				continue
			}
		}

		networkFunc, splitFunc, onConnect, onDisconnect := s.factory()
		client := newClient(
			conn,
			s.log,
			networkFunc,
			splitFunc,
			uint64(s.config.MaxMessageSize),
			s.config.Timeout,
			onConnect,
			onDisconnect,
		)

		s.wg.Add(1)
		go func() {
			defer logp.Recover("recovering from a tcp client crash")
			defer s.wg.Done()
			defer conn.Close()

			s.registerClient(client)
			defer s.unregisterClient(client)
			s.log.Debugw("New client", "remote_address", conn.RemoteAddr(), "total", s.clientsCount())

			err := client.handle()
			if err != nil {
				s.log.Debugw("ClientInfo error", "error", err)
			}

			defer s.log.Debugw(
				"ClientInfo disconnected",
				"remote_address",
				conn.RemoteAddr(),
				"total",
				s.clientsCount(),
			)
		}()
	}
}

// Stop stops accepting new incoming TCP connection and Close any active clients
func (s *Server) Stop() {
	s.log.Info("Stopping TCP server")
	close(s.done)
	s.Listener.Close()
	for _, client := range s.allClients() {
		client.close()
	}
	s.wg.Wait()
	s.log.Info("TCP server stopped")
}

func (s *Server) registerClient(client *client) {
	s.Lock()
	defer s.Unlock()
	s.clients[client.ID()] = client
}

func (s *Server) unregisterClient(client *client) {
	s.Lock()
	defer s.Unlock()
	delete(s.clients, client.ID())
}

func (s *Server) allClients() []*client {
	s.RLock()
	defer s.RUnlock()
	currentClients := make([]*client, len(s.clients))
	idx := 0
	for _, client := range s.clients {
		currentClients[idx] = client
		idx++
	}
	return currentClients
}

func (s *Server) createServer() (net.Listener, error) {
	var l net.Listener
	var err error
	if s.tlsConfig != nil {
		t := s.tlsConfig.BuildModuleConfig(s.config.Host)
		s.log.Info("Listening over TLS")
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

func (s *Server) clientsCount() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.clients)
}

// SplitFunc allows to create a `bufio.SplitFunc` based on a delimiter provided.
func SplitFunc(lineDelimiter []byte) bufio.SplitFunc {
	ld := []byte(lineDelimiter)
	if bytes.Equal(ld, []byte("\n")) {
		// This will work for most usecases and will also strip \r if present.
		// CustomDelimiter, need to match completely and the delimiter will be completely removed from
		// the returned byte slice
		return bufio.ScanLines
	}
	return factoryDelimiter(ld)
}
