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

package compose

import (
	"net"
	"testing"
)

// R extends the testing.T interface with methods that expose information about current scenario
type R interface {
	testing.TB

	WithT(t *testing.T) R

	Host() string
	Hostname() string
	Port() string

	Option(string) string
}

type runnerControl struct {
	*testing.T

	host     string
	scenario map[string]string
}

// WithT creates a copy of R with the given T
func (r *runnerControl) WithT(t *testing.T) R {
	ctl := *r
	ctl.T = t
	return &ctl
}

// Host returns the host:port the test should use to connect to the service
func (r *runnerControl) Host() string {
	return r.host
}

// Hostname is the address of the host
func (r *runnerControl) Hostname() string {
	hostname, _, _ := net.SplitHostPort(r.host)
	return hostname
}

// Port is the port of the host
func (r *runnerControl) Port() string {
	_, port, _ := net.SplitHostPort(r.host)
	return port
}

// Option returns the value of an option for the current scenario
func (r *runnerControl) Option(key string) string {
	return r.scenario[key]
}
