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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"gopkg.in/yaml.v2"
)

var esCommonOutput = `
exporters:
  elasticsearch:
    endpoints:
      - https://localhost:9200
    idle_conn_timeout: 3s
    logs_index: form-otel-exporter
    password: changeme
    retry:
      enabled: true
      initial_interval: 1s
      max_interval: 1m0s
      max_retries: 3
    user: elastic
    timeout: 1m30s
    batcher:
      enabled: true
      max_size: 1600
      min_size: 0
    mapping:
      mode: bodymap       
`

func TestConverter(t *testing.T) {
	c := converter{}
	t.Run("test converter functionality", func(t *testing.T) {
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
		var expectedOutput = esCommonOutput + `
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

		input := newFromYamlString(t, supportedInput)
		err := c.Convert(context.Background(), input)
		require.NoError(t, err, "error converting beats output config")

		expOutput := newFromYamlString(t, expectedOutput)
		compareAndAssert(t, expOutput, input)

	})

	t.Run("test failure if unsupported config is provided", func(t *testing.T) {
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

		input := newFromYamlString(t, unsupportedOutputConfig)
		err := c.Convert(context.Background(), input)
		require.ErrorContains(t, err, "output type \"kafka\" is unsupported in OTel mode")

	})

	t.Run("test cloud id conversion", func(t *testing.T) {
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
    output:
      elasticsearch:
        hosts: ["https://localhost:9200"]
        username: elastic
        password: changeme
        index: form-otel-exporter
    cloud:
      id: ZWxhc3RpYy5jbyRlcy1ob3N0bmFtZSRraWJhbmEtaG9zdG5hbWU=
      auth: elastic-cloud:password
service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`
		var expectedOutput = `
exporters:
  elasticsearch:
    endpoints:
      - https://es-hostname.elastic.co:443
    idle_conn_timeout: 3s
    logs_index: form-otel-exporter
    password: password
    retry:
      enabled: true
      initial_interval: 1s
      max_interval: 1m0s
      max_retries: 3
    user: elastic-cloud
    timeout: 1m30s
    batcher:
      enabled: true
      max_size: 1600
      min_size: 0
    mapping:
      mode: bodymap       
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - enabled: true
          id: filestream-input-id
          paths:
            - /tmp/flog.log
          type: filestream  
    output:
      otelconsumer: null
    cloud: null
    setup:
      kibana:
        host: https://kibana-hostname.elastic.co:443
service:
  pipelines:
    logs:
      exporters:
        - elasticsearch
      receivers:
        - filebeatreceiver
`

		input := newFromYamlString(t, supportedInput)
		err := c.Convert(context.Background(), input)
		require.NoError(t, err, "error converting beats output config")

		expOutput := newFromYamlString(t, expectedOutput)
		compareAndAssert(t, expOutput, input)

	})

	t.Run("test local queue setting is promoted to global level", func(t *testing.T) {
		var supportedInput = `
receivers:
  filebeatreceiver:
    output:
      elasticsearch:
        hosts: ["https://localhost:9200"]
        username: elastic
        password: changeme
        index: form-otel-exporter
        queue:
          mem:
            events: 3200
            flush:
              min_events: 1600
              timeout: 10s

service:
  pipelines:
    logs:
      receivers:
        - "filebeatreceiver"
`

		var expectedOutput = esCommonOutput + `
receivers:
  filebeatreceiver:
    queue:
      mem:
        events: 3200
        flush:
          min_events: 1600
          timeout: 10s    	  
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

		input := newFromYamlString(t, supportedInput)
		err := c.Convert(context.Background(), input)
		require.NoError(t, err, "error converting beats output config")

		expOutput := newFromYamlString(t, expectedOutput)
		compareAndAssert(t, expOutput, input)

	})
}

func TestLogLevel(t *testing.T) {
	c := converter{}
	tests := []struct {
		name          string
		level         string
		expectedLevel string
		expectedError string
	}{
		{
			name:          "test-debug",
			level:         "debug",
			expectedLevel: "DEBUG",
		},
		{
			name:          "test-info",
			level:         "info",
			expectedLevel: "INFO",
		},
		{
			name:          "test-warn",
			level:         "warning",
			expectedLevel: "WARN",
		},
		{
			name:          "test-error",
			level:         "error",
			expectedLevel: "ERROR",
		},
		{
			name:          "test-critical",
			level:         "critical",
			expectedLevel: "ERROR",
		},
		{
			name:          "test-error",
			level:         "blabla",
			expectedError: "unrecognized level: blabla",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			supportedInput := fmt.Sprintf(`
      receivers:
        filebeatreceiver:
          logging: 
            level: %s
          filebeat:
            inputs:
              - type: filestream
                enabled: true
                id: filestream-input-id
                paths:
                  - /tmp/flog.log
      `, test.level)
			input := newFromYamlString(t, supportedInput)
			err := c.Convert(context.Background(), input)
			if test.expectedError != "" {
				require.ErrorContains(t, err, test.expectedError)
			} else {
				require.NoError(t, err)
				inputMap := input.Get("service::telemetry::logs::level")
				require.Equal(t, test.expectedLevel, inputMap)
			}
		})
	}

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
