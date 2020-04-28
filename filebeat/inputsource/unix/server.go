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
	"fmt"
	"net"
	"os"
	"os/user"
	"runtime"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/net/netutil"

	"github.com/elastic/beats/v7/filebeat/inputsource/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// Server represent a unix server
type Server struct {
	*common.Listener

	config *Config
}

// New creates a new unix server
func New(
	config *Config,
	factory common.HandlerFactory,
) (*Server, error) {
	if factory == nil {
		return nil, fmt.Errorf("HandlerFactory can't be empty")
	}

	server := &Server{
		config: config,
	}
	server.Listener = common.NewListener(common.FamilyUnix, config.Path, factory, server.createServer, &common.ListenerConfig{
		Timeout:        config.Timeout,
		MaxMessageSize: config.MaxMessageSize,
		MaxConnections: config.MaxConnections,
	})

	return server, nil
}

func (s *Server) createServer() (net.Listener, error) {
	if err := s.cleanupStaleSocket(); err != nil {
		return nil, err
	}

	l, err := net.Listen("unix", s.config.Path)
	if err != nil {
		return nil, err
	}

	if err := s.setSocketOwnership(); err != nil {
		return nil, err
	}

	if s.config.MaxConnections > 0 {
		return netutil.LimitListener(l, s.config.MaxConnections), nil
	}
	return l, nil
}

func (s *Server) cleanupStaleSocket() error {
	path := s.config.Path
	if info, err := os.Stat(path); !os.IsNotExist(err) {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("refusing to remove file at location %s, it is not a socket", path)
		}

		if err := os.Remove(path); err != nil {
			return errors.Wrapf(
				err,
				"cannot remove existing unix socket file at location %s",
				path,
			)
		}
	}
	return nil
}

func (s *Server) setSocketOwnership() error {
	if runtime.GOOS == "windows" {
		logger := logp.NewLogger("unix")
		if s.config.User != nil {
			logger.Warn("windows does not support the 'user' configuration option, ignoring")
		}
		if s.config.Group != nil {
			logger.Warn("windows does not support the 'group' configuration option, ignoring")
		}
		return nil
	}
	// -1 == do not change
	uid := -1
	gid := -1
	if s.config.User != nil {
		u, err := user.Lookup(*s.config.User)
		if err != nil {
			return err
		}
		uid, err = strconv.Atoi(u.Uid)
		if err != nil {
			return err
		}
	}
	if s.config.Group != nil {
		g, err := user.LookupGroup(*s.config.Group)
		if err != nil {
			return err
		}
		gid, err = strconv.Atoi(g.Gid)
		if err != nil {
			return err
		}
	}
	if uid != -1 || gid != -1 {
		return os.Chown(s.config.Path, uid, gid)
	}
	return nil
}
