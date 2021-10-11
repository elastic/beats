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

package dialer

import (
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/transport"
)

// UnixDialerBuilder creates a builder to dial over unix domain socket.
type UnixDialerBuilder struct {
	Path string
}

// Make creates a dialer.
func (t *UnixDialerBuilder) Make(timeout time.Duration) (transport.Dialer, error) {
	return transport.UnixDialer(timeout, strings.TrimSuffix(t.Path, "/")), nil
}

func (t *UnixDialerBuilder) String() string {
	return "Unix: " + t.Path
}

// NpipeDialerBuilder creates a builder to dial over a named pipe.
type NpipeDialerBuilder struct {
	Path string
}

// Make creates a dialer.
func (t *NpipeDialerBuilder) Make(_ time.Duration) (transport.Dialer, error) {
	return nil, errors.New("cannot the URI, named pipes are only supported on Windows")
}

func (t *NpipeDialerBuilder) String() string {
	return "Npipe: " + t.Path
}
