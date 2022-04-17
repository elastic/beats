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

//go:build linux || darwin || windows
// +build linux darwin windows

package docker

import (
	"time"

	"github.com/menderesk/beats/v7/libbeat/autodiscover/template"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/docker"
)

// Config for docker autodiscover provider
type Config struct {
	Host           string                  `config:"host"`
	TLS            *docker.TLSConfig       `config:"ssl"`
	Prefix         string                  `config:"prefix"`
	Hints          *common.Config          `config:"hints"`
	Builders       []*common.Config        `config:"builders"`
	Appenders      []*common.Config        `config:"appenders"`
	Templates      template.MapperSettings `config:"templates"`
	Dedot          bool                    `config:"labels.dedot"`
	CleanupTimeout time.Duration           `config:"cleanup_timeout" validate:"positive"`
}

// Public variable, so specific beats (as Filebeat) can set a different cleanup timeout if they need it.
var DefaultCleanupTimeout time.Duration = 0

func defaultConfig() *Config {
	return &Config{
		Host:           "unix:///var/run/docker.sock",
		Prefix:         "co.elastic",
		Dedot:          true,
		CleanupTimeout: DefaultCleanupTimeout,
	}
}

// Validate ensures correctness of config
func (c *Config) Validate() {
	// Make sure that prefix doesn't ends with a '.'
	if c.Prefix[len(c.Prefix)-1] == '.' && c.Prefix != "." {
		c.Prefix = c.Prefix[:len(c.Prefix)-2]
	}
}
