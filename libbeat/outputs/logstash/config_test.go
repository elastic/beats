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
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {

	info := beat.Info{Beat: "testbeat", Name: "foo", IndexPrefix: "bar"}
	for name, test := range map[string]struct {
		config         *common.Config
		expectedConfig *Config
		err            bool
	}{
		"default config": {
			config: common.MustNewConfigFrom([]byte(`{ }`)),
			expectedConfig: &Config{
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
				Index:      "bar",
			},
		},
		"config given": {
			config: common.MustNewConfigFrom(common.MapStr{
				"index":         "beat-index",
				"loadbalance":   true,
				"bulk_max_size": 1024,
				"slow_start":    false,
			}),
			expectedConfig: &Config{
				LoadBalance:      true,
				BulkMaxSize:      1024,
				Pipelining:       2,
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
				Index:      "beat-index",
			},
		},
		"removed config setting": {
			config: common.MustNewConfigFrom(common.MapStr{
				"port": "8080",
			}),
			expectedConfig: nil,
			err:            true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := readConfig(test.config, info)
			if test.err {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedConfig, cfg)
			}
		})
	}
}
