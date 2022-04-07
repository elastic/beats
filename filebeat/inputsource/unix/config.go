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
	"time"

	"github.com/elastic/beats/v8/filebeat/inputsource/common/streaming"
	"github.com/elastic/beats/v8/libbeat/common/cfgtype"
)

type SocketType uint8

const (
	// StreamSocket is used when reading from a Unix stream socket.
	StreamSocket SocketType = iota
	// DatagramSocket is used when reading from a Unix datagram socket.
	DatagramSocket
)

const (
	// Name is the human readable name and identifier.
	Name = "unix"
)

var socketTypes = map[string]SocketType{
	"stream":   StreamSocket,
	"datagram": DatagramSocket,
}

// Config exposes the unix configuration.
type Config struct {
	Path           string                `config:"path"`
	Group          *string               `config:"group"`
	Mode           *string               `config:"mode"`
	Timeout        time.Duration         `config:"timeout" validate:"nonzero,positive"`
	MaxMessageSize cfgtype.ByteSize      `config:"max_message_size" validate:"nonzero,positive"`
	MaxConnections int                   `config:"max_connections"`
	LineDelimiter  string                `config:"line_delimiter"`
	Framing        streaming.FramingType `config:"framing"`
	SocketType     SocketType            `config:"socket_type"`
}

// Validate validates the Config option for the unix input.
func (c *Config) Validate() error {
	if len(c.Path) == 0 {
		return fmt.Errorf("need to specify the path to the unix socket")
	}

	if c.SocketType == StreamSocket && c.LineDelimiter == "" {
		return fmt.Errorf("line_delimiter cannot be empty when using stream socket")
	}
	return nil
}

func (s *SocketType) Unpack(value string) error {
	setting, ok := socketTypes[value]
	if !ok {
		availableTypes := make([]string, len(socketTypes))
		i := 0
		for t := range socketTypes {
			availableTypes[i] = t
			i++
		}
		return fmt.Errorf("unknown socket type: %s, supported types: %v", value, availableTypes)
	}

	*s = setting
	return nil
}
