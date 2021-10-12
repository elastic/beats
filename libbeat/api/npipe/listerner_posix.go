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

package npipe

import (
	"context"
	"errors"
	"net"
)

// DialContext create a Dial to be use with an http.Client to connect to a pipe.
func DialContext(npipe string) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		return nil, errors.New("named pipe doesn't work on linux")
	}
}

// Dial create a Dial to be use with an http.Client to connect to a pipe.
func Dial(npipe string) func(string, string) (net.Conn, error) {
	return func(_, _ string) (net.Conn, error) {
		return nil, errors.New("named pipe doesn't work on linux")
	}
}
