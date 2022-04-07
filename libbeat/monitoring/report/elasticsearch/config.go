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
	"fmt"
	"time"

	"github.com/elastic/beats/v8/libbeat/common/transport/httpcommon"
)

// config is subset of libbeat/outputs/elasticsearch config tailored
// for reporting metrics only
type config struct {
	Hosts            []string
	Protocol         string
	Params           map[string]string `config:"parameters"`
	Headers          map[string]string `config:"headers"`
	Username         string            `config:"username"`
	Password         string            `config:"password"`
	APIKey           string            `config:"api_key"`
	ProxyURL         string            `config:"proxy_url"`
	CompressionLevel int               `config:"compression_level" validate:"min=0, max=9"`
	MaxRetries       int               `config:"max_retries"`
	MetricsPeriod    time.Duration     `config:"metrics.period"`
	StatePeriod      time.Duration     `config:"state.period"`
	BulkMaxSize      int               `config:"bulk_max_size" validate:"min=0"`
	BufferSize       int               `config:"buffer_size"`
	Tags             []string          `config:"tags"`
	Backoff          backoff           `config:"backoff"`
	ClusterUUID      string            `config:"cluster_uuid"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

type backoff struct {
	Init time.Duration
	Max  time.Duration
}

func (c *config) Validate() error {
	if c.APIKey != "" && (c.Username != "" && c.Password != "") {
		return fmt.Errorf("cannot set both api_key and username/password for monitoring client")
	}

	return nil
}
