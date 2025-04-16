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

package elasticsearchtranslate

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/config"
)

func TestToOtelConfig(t *testing.T) {

	t.Run("basic config translation", func(t *testing.T) {
		beatCfg := `
hosts:
  - localhost:9200
  - localhost:9300
protocol: http
path: /foo/bar
username: elastic
password: changeme
index: "some-index"
pipeline: "some-ingest-pipeline"
proxy_url: "https://proxy.url"
backoff:
  init: 42s
  max: 420s
workers: 30
headers:
  X-Header-1: foo
  X-Bar-Header: bar`

		OTelCfg := `
api_key: ""
endpoints:
  - http://localhost:9200/foo/bar
  - http://localhost:9300/foo/bar
idle_conn_timeout: 3s
logs_index: some-index
num_workers: 30
password: changeme
pipeline: some-ingest-pipeline
proxy_url: https://proxy.url
retry:
  enabled: true
  initial_interval: 42s
  max_interval: 7m0s
  max_retries: 3
timeout: 1m30s
user: elastic
headers:
  X-Header-1: foo
  X-Bar-Header: bar
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
mapping:
  mode: bodymap  
 `
		input := newFromYamlString(t, beatCfg)
		cfg := config.MustNewConfigFrom(input.ToStringMap())
		got, err := ToOTelConfig(cfg)
		require.NoError(t, err, "error translating elasticsearch output to OTel ES exporter type")
		expOutput := newFromYamlString(t, OTelCfg)
		compareAndAssert(t, expOutput, confmap.NewFromStringMap(got))

	})

	// when preset is configured, we only test worker, bulk_max_size, idle_connection_timeout here
	// TODO: Check for compression_level when we add support upstream
	t.Run("check preset config translation", func(t *testing.T) {
		commonBeatCfg := `
hosts:
  - localhost:9200
index: "some-index"
username: elastic
password: changeme
preset: %s
`

		commonOTelCfg := `
api_key: ""
endpoints:
  - http://localhost:9200
retry:
  enabled: true
  initial_interval: 1s
  max_interval: 1m0s
  max_retries: 3
logs_index: some-index
password: changeme
user: elastic
timeout: 1m30s
mapping:
  mode: bodymap 
`

		tests := []struct {
			presetName string
			output     string
		}{
			{
				presetName: "balanced",
				output: commonOTelCfg + `
idle_conn_timeout: 3s
num_workers: 1
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
 `,
			},
			{
				presetName: "throughput",
				output: commonOTelCfg + `
idle_conn_timeout: 15s
num_workers: 4
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
 `,
			},
			{
				presetName: "scale",
				output: `
api_key: ""
endpoints:
  - http://localhost:9200
retry:
  enabled: true
  initial_interval: 5s
  max_interval: 5m0s
  max_retries: 3
logs_index: some-index
password: changeme
user: elastic
timeout: 1m30s
idle_conn_timeout: 1s
num_workers: 1
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
mapping:
  mode: bodymap    
 `,
			},
			{
				presetName: "latency",
				output: commonOTelCfg + `
idle_conn_timeout: 1m0s
num_workers: 1
batcher:
  enabled: true
  max_size: 50
  min_size: 0
 `,
			},
			{
				presetName: "custom",
				output: commonOTelCfg + `
idle_conn_timeout: 3s
num_workers: 0
batcher:
  enabled: true
  max_size: 1600
  min_size: 0
 `,
			},
		}

		for _, test := range tests {
			t.Run("config translation w/"+test.presetName, func(t *testing.T) {
				input := newFromYamlString(t, fmt.Sprintf(commonBeatCfg, test.presetName))
				cfg := config.MustNewConfigFrom(input.ToStringMap())
				got, err := ToOTelConfig(cfg)
				require.NoError(t, err, "error translating elasticsearch output to OTel ES exporter type")
				expOutput := newFromYamlString(t, test.output)
				compareAndAssert(t, expOutput, confmap.NewFromStringMap(got))
			})
		}

	})

}

func newFromYamlString(t *testing.T, input string) *confmap.Conf {
	t.Helper()
	var rawConf map[string]any
	err := yaml.Unmarshal([]byte(input), &rawConf)
	require.NoError(t, err)

	return confmap.NewFromStringMap(rawConf)
}

func compareAndAssert(t *testing.T, expectedOutput *confmap.Conf, gotOutput *confmap.Conf) {
	t.Helper()
	// convert it to a common type
	want, err := yaml.Marshal(expectedOutput.ToStringMap())
	require.NoError(t, err)
	got, err := yaml.Marshal(gotOutput.ToStringMap())
	require.NoError(t, err)

	assert.Equal(t, string(want), string(got))
}
