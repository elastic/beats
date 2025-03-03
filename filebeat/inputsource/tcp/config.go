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

package tcp

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// Name is the human readable name and identifier.
const Name = "tcp"

// Config exposes the tcp configuration.
type Config struct {
	Host           string                  `config:"host"`
	Timeout        time.Duration           `config:"timeout" validate:"nonzero,positive"`
	MaxMessageSize cfgtype.ByteSize        `config:"max_message_size" validate:"nonzero,positive"`
	MaxConnections int                     `config:"max_connections"`
	TLS            *tlscommon.ServerConfig `config:"ssl"`
	Network        string                  `config:"network"`
}

const (
	networkTCP  = "tcp"
	networkTCP4 = "tcp4"
	networkTCP6 = "tcp6"
)

var (
	ErrInvalidNetwork  = errors.New("invalid network value")
	ErrMissingHostPort = errors.New("need to specify the host using the `host:port` syntax")
)

// Validate validates the Config option for the tcp input.
func (c *Config) Validate() error {
	if len(c.Host) == 0 {
		return ErrMissingHostPort
	}
	switch c.Network {
	case "", networkTCP, networkTCP4, networkTCP6:
	default:
		return fmt.Errorf("%w: %s, expected: %v or %v or %v ", ErrInvalidNetwork, c.Network, networkTCP, networkTCP4, networkTCP6)
	}
	return nil
}
