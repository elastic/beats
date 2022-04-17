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
	"time"

	"github.com/menderesk/beats/v7/heartbeat/monitors"
	"github.com/menderesk/beats/v7/libbeat/common/transport"
	"github.com/menderesk/beats/v7/libbeat/common/transport/tlscommon"
)

type config struct {
	// check all ports if host does not contain port
	Hosts []string `config:"hosts" validate:"required"`
	Ports []uint16 `config:"ports"`

	Mode monitors.IPSettings `config:",inline"`

	Socks5 transport.ProxyConfig `config:",inline"`

	// configure tls
	TLS *tlscommon.Config `config:"ssl"`

	Timeout time.Duration `config:"timeout"`

	// validate connection
	SendString    string `config:"check.send"`
	ReceiveString string `config:"check.receive"`
}

func defaultConfig() config {
	return config{
		Timeout: 16 * time.Second,
		Mode:    monitors.DefaultIPSettings,
	}
}

func (c *config) Validate() error {
	if c.Socks5.URL != "" {
		if c.Mode.Mode != monitors.PingAny && !c.Socks5.LocalResolve {
			return errors.New("ping all ips only supported if proxy_use_local_resolver is enabled`")
		}
	}

	return nil
}
