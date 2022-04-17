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

//go:build !windows
// +build !windows

package api

import (
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/api/npipe"
)

func makeListener(cfg Config) (net.Listener, error) {
	if len(cfg.User) > 0 {
		return nil, errors.New("specifying a user is not supported under this platform")
	}

	if len(cfg.SecurityDescriptor) > 0 {
		return nil, errors.New("security_descriptor option for the HTTP endpoint only work on Windows")
	}

	if npipe.IsNPipe(cfg.Host) {
		return nil, fmt.Errorf(
			"cannot use %s as the host, named pipes are only supported on Windows",
			cfg.Host,
		)
	}

	network, path, err := parse(cfg.Host, cfg.Port)
	if err != nil {
		return nil, err
	}

	if network == "unix" {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			if err := os.Remove(path); err != nil {
				return nil, errors.Wrapf(
					err,
					"cannot remove existing unix socket file at location %s",
					path,
				)
			}
		}
	}

	l, err := net.Listen(network, path)
	if err != nil {
		return nil, err
	}

	// Ensure file mode
	if network == "unix" {
		if err := os.Chmod(path, socketFileMode); err != nil {
			return nil, errors.Wrapf(
				err,
				"could not set mode %d for unix socket file at location %s",
				socketFileMode,
				path,
			)
		}
	}

	return l, nil
}
