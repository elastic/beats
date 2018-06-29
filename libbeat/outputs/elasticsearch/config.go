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

package elasticsearch

import (
	"time"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
)

type elasticsearchConfig struct {
	Protocol         string            `config:"protocol"`
	Path             string            `config:"path"`
	Params           map[string]string `config:"parameters"`
	Headers          map[string]string `config:"headers"`
	Username         string            `config:"username"`
	Password         string            `config:"password"`
	ProxyURL         string            `config:"proxy_url"`
	LoadBalance      bool              `config:"loadbalance"`
	CompressionLevel int               `config:"compression_level" validate:"min=0, max=9"`
	EscapeHTML       bool              `config:"escape_html"`
	TLS              *tlscommon.Config `config:"ssl"`
	BulkMaxSize      int               `config:"bulk_max_size"`
	MaxRetries       int               `config:"max_retries"`
	Timeout          time.Duration     `config:"timeout"`
	Backoff          Backoff           `config:"backoff"`
}

type Backoff struct {
	Init time.Duration
	Max  time.Duration
}

const (
	defaultBulkSize = 50
)

var (
	defaultConfig = elasticsearchConfig{
		Protocol:         "",
		Path:             "",
		ProxyURL:         "",
		Params:           nil,
		Username:         "",
		Password:         "",
		Timeout:          90 * time.Second,
		MaxRetries:       3,
		CompressionLevel: 0,
		EscapeHTML:       true,
		TLS:              nil,
		LoadBalance:      true,
		Backoff: Backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
	}
)

func (c *elasticsearchConfig) Validate() error {
	if c.ProxyURL != "" {
		if _, err := parseProxyURL(c.ProxyURL); err != nil {
			return err
		}
	}

	return nil
}
