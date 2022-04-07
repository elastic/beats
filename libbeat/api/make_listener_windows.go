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

//go:build windows
// +build windows

package api

import (
	"fmt"
	"net"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/api/npipe"
)

func makeListener(cfg Config) (net.Listener, error) {
	if len(cfg.User) > 0 && len(cfg.SecurityDescriptor) > 0 {
		return nil, errors.New("user and security_descriptor are mutually exclusive, define only one of them")
	}

	if npipe.IsNPipe(cfg.Host) {
		pipe := npipe.TransformString(cfg.Host)
		var sd string
		var err error
		if len(cfg.SecurityDescriptor) == 0 {
			sd, err = npipe.DefaultSD(cfg.User)
			if err != nil {
				return nil, errors.Wrap(err, "cannot generate security descriptor for the named pipe")
			}
		} else {
			sd = cfg.SecurityDescriptor
		}
		return npipe.NewListener(pipe, sd)
	}

	network, path, err := parse(cfg.Host, cfg.Port)
	if err != nil {
		return nil, err
	}

	if network == "unix" {
		return nil, fmt.Errorf(
			"cannot use %s as the host, unix sockets are not supported on Windows, use npipe instead",
			cfg.Host,
		)
	}

	return net.Listen(network, path)
}
