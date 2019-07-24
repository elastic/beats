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

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

var errClosed = errors.New("connection closed")

// Server represent a TCP server
type Server struct {
	config       *Config
	Listener     net.Listener
	wg           sync.WaitGroup
	done         chan struct{}
	factory      ClientFactory
	log          *logp.Logger
	tlsConfig    *transport.TLSConfig
	closer       *closer
	clientsCount atomic.Int
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
		done:      make(chan struct{}),
		factory:   factory,
		log:       logp.NewLogger("tcp").With("address", config.Host),
		tlsConfig: tlsConfig,
		closer:    &closer{done: make(chan struct{})},
	}, nil
}

// Start listen to the TCP socket.
func (s *Server) Start() error {
	var err error
	s.Listener, err = s.createServer()
	if err != nil {
		return err
	}

	s.closer.callback = func() { s.Listener.Close() }
	s.log.Info("Started listening for TCP connection")

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.run()
	}()
	return nil
}

// Run start and run a new TCP listener to receive new data. When a new connection is accepted, the factory is used
// to create a ConnectionHandler. The ConnectionHandler takes the connection as input and handles the data that is
// being received via tha io.Reader. Most clients use the splitHandler which can take a bufio.SplitFunc and parse
// out each message into an appropriate event. The Close() of the ConnectionHandler can be used to clean up the
// connection either by client or server based on need.
func (s *Server) run() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			select {
			case <-s.closer.Done():
				return
			default:
				s.log.Debugw("Can not accept the connection", "error", err)
				continue
			}
		}

		client := s.factory(*s.config)
		closer := withCloser(s.closer, conn)

		s.wg.Add(1)
		go func() {
			defer logp.Recover("recovering from a tcp client crash")
			defer s.wg.Done()
			defer closer.Close()

			s.registerClient(client)
			defer s.unregisterClient(client)
			s.log.Debugw("New client", "remote_address", conn.RemoteAddr(), "total", s.clientsCount.Load())

			err := client.Handle(closer, conn)
			if err != nil {
				s.log.Debugw("client error", "error", err)
			}

			defer s.log.Debugw(
				"client disconnected",
				"remote_address",
				conn.RemoteAddr(),
				"total",
				s.clientsCount.Load(),
			)
		}()
	}
}

// Stop stops accepting new incoming TCP connection and Close any active clients
func (s *Server) Stop() {
	s.log.Info("Stopping TCP server")
	s.closer.Close()
	s.wg.Wait()
	s.log.Info("TCP server stopped")
}

func (s *Server) registerClient(client ConnectionHandler) {
	s.clientsCount.Inc()
}

func (s *Server) unregisterClient(client ConnectionHandler) {
	s.clientsCount.Dec()
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

func withCloser(parent *closer, conn net.Conn) *closer {
	child := &closer{
		done:   make(chan struct{}),
		parent: parent,
		callback: func() {
			conn.Close()
		},
	}
	parent.addChild(child)
	return child
}

type closeRef interface {
	Done() <-chan struct{}
	Err() error
}

type closer struct {
	mu       sync.Mutex
	done     chan struct{}
	err      error
	parent   *closer
	children map[*closer]struct{}
	callback func()
}

func (c *closer) Close() {
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return
	}

	if c.callback != nil {
		c.callback()
	}

	close(c.done)

	// propagate close to children.
	if c.children != nil {
		for child := range c.children {
			child.Close()
		}
		c.children = nil
	}

	c.err = errClosed
	c.mu.Unlock()

	if c.parent != nil {
		c.removeChild(c)
	}
}

func (c *closer) Done() <-chan struct{} {
	return c.done
}

func (c *closer) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

func (c *closer) removeChild(child *closer) {
	c.mu.Lock()
	delete(c.children, child)
	c.mu.Unlock()
}

func (c *closer) addChild(child *closer) {
	c.mu.Lock()
	if c.children == nil {
		c.children = make(map[*closer]struct{})
	}
	c.children[child] = struct{}{}
	c.mu.Unlock()
}
