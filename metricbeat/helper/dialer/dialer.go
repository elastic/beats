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

package dialer

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/outputs/transport"
)

// Builder is a dialer builder.
type Builder interface {
	fmt.Stringer
	Make(time.Duration) (transport.Dialer, error)
}

// DefaultDialerBuilder create a builder to dialer over TCP and UDP.
type DefaultDialerBuilder struct{}

// Make creates a dialer.
func (t *DefaultDialerBuilder) Make(timeout time.Duration) (transport.Dialer, error) {
	return transport.NetDialer(timeout), nil
}

func (t *DefaultDialerBuilder) String() string {
	return "TCP/UDP"
}

// NewDefaultDialerBuilder creates a DefaultDialerBuilder.
func NewDefaultDialerBuilder() *DefaultDialerBuilder {
	return &DefaultDialerBuilder{}
}

// NewNpipeDialerBuilder creates a NpipeDialerBuilder.
func NewNpipeDialerBuilder(path string) *NpipeDialerBuilder {
	return &NpipeDialerBuilder{Path: path}
}

// NewUnixDialerBuilder returns a new TransportUnix instance that will allow the HTTP client to communicate
// over a unix domain socket it require a valid path to the socket on the filesystem.
func NewUnixDialerBuilder(path string) *UnixDialerBuilder {
	return &UnixDialerBuilder{Path: path}
}
