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
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	unixNetwork = "unix"
	tcpNetwork  = "tcp"
)

// Server takes cares of correctly starting the HTTP component of the API
// and will answers all the routes defined in the received ServeMux.
type Server struct {
	log    *logp.Logger
	mux    *http.ServeMux
	srv    *http.Server
	l      net.Listener
	config Config
}

// New creates a new API Server.
func New(log *logp.Logger, mux *http.ServeMux, c *config.C) (*Server, error) {
	if log == nil {
		log = logp.NewLogger("")
	}

	cfg := DefaultConfig()
	err := c.Unpack(&cfg)
	if err != nil {
		return nil, err
	}
	srv := &http.Server{ReadHeaderTimeout: cfg.Timeout}
	l, err := makeListener(cfg)
	if err != nil {
		return nil, err
	}

	return &Server{mux: mux, srv: srv, l: l, config: cfg, log: log.Named("api")}, nil
}

// AddRoute adds a route to the server mux
func (s *Server) AddRoute(path string, handler HandlerFunc) {
	s.mux.HandleFunc(path, handler)
}

// Start starts the HTTP server and accepting new connection.
func (s *Server) Start() {
	s.log.Info("Starting stats endpoint")
	go func(l net.Listener) {
		s.log.Infof("Metrics endpoint listening on: %s (configured: %s)", l.Addr().String(), s.config.Host)
		s.srv.Handler = s.mux
		err := s.srv.Serve(l)
		s.log.Infof("Stats endpoint (%s) finished: %v", l.Addr().String(), err)
	}(s.l)
}

// Stop stops the API server and free any resource associated with the process like unix sockets.
func (s *Server) Stop() error {
	return s.l.Close()
}

// AttachHandler will attach a handler at the specified route and return an error instead of panicing.
func (s *Server) AttachHandler(route string, h http.Handler) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case error:
				err = r
			case string:
				err = errors.New(r)
			default:
				err = fmt.Errorf("handle attempted to panic with %v", r)
			}
		}
	}()
	s.log.Infof("Attempting to attach %q to server.", route)
	s.mux.Handle(route, h)
	return //nolint:nakedret // returning from recover
}

func parse(host string, port int) (string, string, error) {
	url, err := url.Parse(host)
	if err != nil {
		return "", "", err
	}

	// When you don't explicitly define the Scheme we fallback on tcp + host.
	if len(url.Host) == 0 && len(url.Scheme) == 0 {
		addr := host + ":" + strconv.Itoa(port)
		return tcpNetwork, addr, nil
	}

	switch url.Scheme {
	case "http":
		return tcpNetwork, url.Host, nil
	case "unix":
		return url.Scheme, url.Path, nil
	default:
		return "", "", fmt.Errorf("unknown scheme %s for host string %s", url.Scheme, host)
	}
}
