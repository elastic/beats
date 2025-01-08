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

package beatconverter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"gopkg.in/yaml.v2"
)

var supportedInput = `
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          enabled: true
          id: filestream-input-id
          paths:
            - /tmp/flog.log
        - type: log
          enabled: true
          paths:
            - /var/log/*.log			
    output:
      elasticsearch:
        hosts: ["https://localhost:9200"]
        username: elastic
        password: changeme
        index: form-otel-exporter

service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`

var expectedOutput = `
exporters:
  elasticsearch:
    api_key: ""
    endpoints:
      - https://localhost:9200
    idle_conn_timeout: 3s
    logs_index: form-otel-exporter
    num_workers: 0
    password: changeme
    retry:
      enabled: true
      initial_interval: 1s
      max_interval: 1m0s
      max_retries: 3
    user: elastic
    timeout: 1m30s
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - enabled: true
          id: filestream-input-id
          paths:
            - /tmp/flog.log
          type: filestream
        - type: log
          enabled: true
          paths:
            - /var/log/*.log		  
    output:
      otelconsumer: null
service:
  pipelines:
    logs:
      exporters:
        - elasticsearch
      receivers:
        - filebeatreceiver
`

var unsupportedOutputConfig = `
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          enabled: true
          id: filestream-input-id
          paths:
            - /tmp/flog.log		
    output:
      kafka: 
        enabled: true 

service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`

func TestConverter(t *testing.T) {
	c := converter{}
	t.Run("test converter functionality", func(t *testing.T) {

		input := newFromYamlString(t, supportedInput)
		err := c.Convert(context.Background(), input)
		require.NoError(t, err, "error converting beats output config")

		expOutput := newFromYamlString(t, expectedOutput)

		// convert it to a common type
		want, err := yaml.Marshal(expOutput.ToStringMap())
		require.NoError(t, err)
		got, err := yaml.Marshal(input.ToStringMap())
		require.NoError(t, err)

		assert.Equal(t, string(want), string(got))

	})

	t.Run("test failure if unsupported config is provided", func(t *testing.T) {
		input := newFromYamlString(t, unsupportedOutputConfig)
		err := c.Convert(context.Background(), input)
		require.ErrorContains(t, err, "output type \"kafka\" is unsupported in OTel mode")

	})

	// TODO: Add a test case with cloud id set
}

func newFromYamlString(t *testing.T, input string) *confmap.Conf {
	t.Helper()
	var rawConf map[string]any
	err := yaml.Unmarshal([]byte(input), &rawConf)
	require.NoError(t, err)

	return confmap.NewFromStringMap(rawConf)
}
