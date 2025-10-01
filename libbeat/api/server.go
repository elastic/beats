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
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type serverState int

const (
	stateNew = iota
	stateStarted
	stateStopped
)

// Server takes care of correctly starting the HTTP component of the API
// and will answer all the routes defined in the received ServeMux.
type Server struct {
	log        *logp.Logger
	mux        *http.ServeMux
	l          net.Listener
	config     Config
	wg         sync.WaitGroup
	mutex      sync.Mutex
	httpServer *http.Server
	state      serverState
}

// New creates a new API Server with no routes attached.
func New(log *logp.Logger, config *config.C) (*Server, error) {
	cfg := DefaultConfig
	err := config.Unpack(&cfg)
	if err != nil {
		return nil, err
	}

	l, err := makeListener(cfg)
	if err != nil {
		return nil, err
	}

	return &Server{
		mux:    http.NewServeMux(),
		l:      l,
		config: cfg,
		log:    log.Named("api"),
		state:  stateNew,
	}, nil
}

// Start starts the HTTP server and accepting new connection.
func (s *Server) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch s.state {
	case stateNew:
		s.state = stateStarted
		s.log.Info("Starting stats endpoint")
		s.wg.Add(1)
		s.httpServer = &http.Server{Handler: s.mux} //nolint:gosec // Keep original behavior
		go func(l net.Listener) {
			defer s.wg.Done()
			s.log.Infof("metrics endpoint listening on: %s (configured: %s)", l.Addr().String(), s.config.Host)

			err := s.httpServer.Serve(l)
			s.log.Infof("stats endpoint (%s) finished: %v", l.Addr().String(), err)
		}(s.l)
		return
	case stateStarted:
		// only call Start once
		s.log.Debug("not starting stats endpoint because start was already called")
		return
	case stateStopped:
		s.log.Debug("not starting stats endpoint because stop was already called")
		return
	default:
		s.log.Errorf("unknown stats server state: %d", s.state)
	}
}

// Stop stops the API server and free any resource associated with the process like unix sockets.
func (s *Server) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch s.state {
	case stateNew:
		s.state = stateStopped
		// New always creates a listener, need to close it even if the server hasn't started
		if err := s.l.Close(); err != nil {
			s.log.Infof("error closing stats endpoint (%s): %v", s.l.Addr().String(), err)
		}
		return nil
	case stateStarted:
		s.state = stateStopped
		// Closing the server will also close the listener
		if err := s.httpServer.Close(); err != nil {
			return fmt.Errorf("error closing monitoring server: %w", err)
		}
		s.wg.Wait()
		return nil
	case stateStopped:
		// only need to call Stop once
		s.log.Debug("not stopping stats endpoint because stop was already called")
		return nil
	default:
		return fmt.Errorf("unknown stats server state: %d", s.state)
	}
}

// AttachHandler will attach a handler at the specified route. Routes are
// matched in the order in which that are attached.
// Attaching the same route twice will panic
func (s *Server) AttachHandler(route string, h http.Handler) (err error) {
	s.mux.Handle(route, h)
	if !strings.HasSuffix(route, "/") && !strings.HasSuffix(route, "{$}") {
		// register /route/ handler
		s.mux.Handle(route+"/{$}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// redirect /route/ to /route
			http.Redirect(w, r, strings.TrimSuffix(r.URL.String(), "/"), http.StatusMovedPermanently)
		}))
	}
	s.log.Debugf("Attached handler at %q to server.", route)
	return nil
}

// Router returns the mux.Router that handles all request to the server.
func (s *Server) Router() *http.ServeMux {
	return s.mux
}

func parse(host string, port int) (string, string, error) {
	url, err := url.Parse(host)
	if err != nil {
		return "", "", err
	}

	// When you don't explicitly define the Scheme we fall back on tcp + host.
	if len(url.Host) == 0 && len(url.Scheme) == 0 {
		addr := host + ":" + strconv.Itoa(port)
		return "tcp", addr, nil
	}

	switch url.Scheme {
	case "http":
		return "tcp", url.Host, nil
	case "unix":
		return url.Scheme, url.Path, nil
	default:
		return "", "", fmt.Errorf("unknown scheme %s for host string %s", url.Scheme, host)
	}
}
