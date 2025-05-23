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

package udp

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
)

// Config options for the UDPServer
type Config struct {
	Host           string           `config:"host"`
	MaxMessageSize cfgtype.ByteSize `config:"max_message_size" validate:"positive,nonzero"`
	Timeout        time.Duration    `config:"timeout"`
	ReadBuffer     cfgtype.ByteSize `config:"read_buffer" validate:"positive"`
	Network        string           `config:"network"`
}

const (
	networkUDP  = "udp"
	networkUDP4 = "udp4"
	networkUDP6 = "udp6"
)

var ErrInvalidNetwork = errors.New("invalid network value")

// Validate validates the Config option for the udp input.
func (c *Config) Validate() error {
	switch c.Network {
	case "", networkUDP, networkUDP4, networkUDP6:
	default:
		return fmt.Errorf("%w: %s, expected: %v or %v or %v", ErrInvalidNetwork, c.Network, networkUDP, networkUDP4, networkUDP6)
	}
	return nil
}
