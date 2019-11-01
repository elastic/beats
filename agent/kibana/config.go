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

package kibana

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
)

// Config is the configuration for the Kibana client.
type Config struct {
	Protocol Protocol          `config:"protocol" yaml:"protocol"`
	SpaceID  string            `config:"space.id" yaml:"space.id,omitempty"`
	Username string            `config:"username" yaml:"username,omitempty"`
	Password string            `config:"password" yaml:"password,omitempty"`
	Path     string            `config:"path" yaml:"path,omitempty"`
	Host     string            `config:"host" yaml:"host,omitempty"`
	Timeout  time.Duration     `config:"timeout" yaml:"timeout,omitempty"`
	TLS      *tlscommon.Config `config:"ssl" yaml:"ssl,omitempty"`
}

// Protocol define the protocol to use to make the connection. (Either HTTPS or HTTP)
type Protocol string

// Unpack the protocol.
func (p *Protocol) Unpack(from string) error {
	if from != "https" && from != "http" {
		return fmt.Errorf("invalid protocol %s, accepted values are 'http' and 'https'", from)
	}
	return nil
}

func defaultClientConfig() Config {
	return Config{
		Protocol: Protocol("http"),
		Host:     "localhost:5601",
		Path:     "",
		SpaceID:  "",
		Username: "",
		Password: "",
		Timeout:  90 * time.Second,
		TLS:      nil,
	}
}

// IsBasicAuth returns true if the username and password are both defined.
func (c *Config) IsBasicAuth() bool {
	return len(c.Username) > 0 && len(c.Password) > 0
}
