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

package logstash

import (
	"strings"
	"time"

	"github.com/elastic/beats/v8/libbeat/beat"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v8/libbeat/common/transport"
	"github.com/elastic/beats/v8/libbeat/common/transport/tlscommon"
)

type Config struct {
	Index            string                `config:"index"`
	LoadBalance      bool                  `config:"loadbalance"`
	BulkMaxSize      int                   `config:"bulk_max_size"`
	SlowStart        bool                  `config:"slow_start"`
	Timeout          time.Duration         `config:"timeout"`
	TTL              time.Duration         `config:"ttl"               validate:"min=0"`
	Pipelining       int                   `config:"pipelining"        validate:"min=0"`
	CompressionLevel int                   `config:"compression_level" validate:"min=0, max=9"`
	MaxRetries       int                   `config:"max_retries"       validate:"min=-1"`
	TLS              *tlscommon.Config     `config:"ssl"`
	Proxy            transport.ProxyConfig `config:",inline"`
	Backoff          Backoff               `config:"backoff"`
	EscapeHTML       bool                  `config:"escape_html"`
}

type Backoff struct {
	Init time.Duration
	Max  time.Duration
}

func defaultConfig() Config {
	return Config{
		LoadBalance:      false,
		Pipelining:       2,
		BulkMaxSize:      2048,
		SlowStart:        false,
		CompressionLevel: 3,
		Timeout:          30 * time.Second,
		MaxRetries:       3,
		TTL:              0 * time.Second,
		Backoff: Backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		EscapeHTML: false,
	}
}

func readConfig(cfg *common.Config, info beat.Info) (*Config, error) {
	c := defaultConfig()

	err := cfgwarn.CheckRemoved6xSettings(cfg, "port")
	if err != nil {
		return nil, err
	}

	if err := cfg.Unpack(&c); err != nil {
		return nil, err
	}

	if c.Index == "" {
		c.Index = strings.ToLower(info.IndexPrefix)
	}

	return &c, nil
}
