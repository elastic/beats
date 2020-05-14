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

package syslog

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type config struct {
	Hosts          []string              `config:"hosts"                    validate:"required"`
	Port           int                   `config:"port"`
	MaxRetries     int                   `config:"max_retries"`
	Timeout        time.Duration         `config:"timeout"                  validate:"min=1"`
	SyslogProgram  string                `config:"default_syslog_program"`
	SyslogPriority uint64                `config:"default_syslog_priority"`
	SyslogSeverity uint64                `config:"default_syslog_severity"`
	TLS            *outputs.TLSConfig    `config:"tls"`
	Proxy          transport.ProxyConfig `config:",inline"`
	Backoff        Backoff               `config:"backoff"`
	Codec          codec.Config          `config:"codec"`
	Network        string                `config:"network"`
}

// Backoff defines the interface for backoff strategies.
type Backoff struct {
	Init time.Duration
	Max  time.Duration
}

// We set the default values for program, priority and severity here,
// and override them in PublishEvents if they're set on individual files.
// 	 Priority 1: user-level messages.
//   Severity 6: infomational messages.
var (
	defaultConfig = config{
		Port:           514,
		MaxRetries:     3,
		Timeout:        5 * time.Second,
		SyslogProgram:  "filebeat",
		SyslogPriority: 1,
		SyslogSeverity: 6,
		Network:        "udp",
		Backoff: Backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
	}
)
