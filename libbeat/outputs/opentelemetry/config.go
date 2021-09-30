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

package opentelemetry

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

type otelConfig struct {
	UserAgent   string            `config:"user_agent"`
	Endpoint    string            `config:"endpoint"`
	LoadBalance bool              `config:"loadbalance"`
	Timeout     time.Duration     `config:"timeout"`
	TLS         *tlscommon.Config `config:"ssl"`
	DataSource  string            `config:"datasource"`
}

type backoff struct {
	Init time.Duration
	Max  time.Duration
}

var (
	defaultConfig = otelConfig{
		LoadBalance: true,
		Timeout:     5 * time.Second,
		TLS:         nil,
		DataSource:  "metrics",
		Endpoint:    "0.0.0.0:4317",
	}
)

func (c *otelConfig) Validate() error {
	return nil
}
